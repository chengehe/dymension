package keeper

import (
	"errors"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/dymensionxyz/sdk-utils/utils/uevent"
	"github.com/osmosis-labs/osmosis/v15/x/gamm/pool-models/balancer"
	gammtypes "github.com/osmosis-labs/osmosis/v15/x/gamm/types"

	"github.com/dymensionxyz/dymension/v3/x/iro/types"
	lockuptypes "github.com/dymensionxyz/dymension/v3/x/lockup/types"
)

// AfterTransfersEnabled called by the genesis transfer IBC module when a transfer is handled
// This is a rollapp module hook
func (k Keeper) AfterTransfersEnabled(ctx sdk.Context, rollappId, rollappIBCDenom string) error {
	return k.Settle(ctx, rollappId, rollappIBCDenom)
}

// Settle settles the iro plan with the given rollappId
//
// This function performs the following steps:
// - Validates that the "TotalAllocation.Amount" of the RA token are available in the module account.
// - Burns any unsold FUT tokens in the module account.
// - Marks the plan as settled, allowing users to claim tokens.
// - Starts the vesting schedule for the owner tokens.
// - Uses the raised liquidity and unsold tokens to bootstrap the rollapp's liquidity pool.
func (k Keeper) Settle(ctx sdk.Context, rollappId, rollappIBCDenom string) error {
	plan, found := k.GetPlanByRollapp(ctx, rollappId)
	if !found {
		return nil
	}

	if plan.IsSettled() {
		return errorsmod.Wrapf(errors.Join(gerrc.ErrInternal, types.ErrPlanSettled), "rollappId: %s", rollappId)
	}

	// validate the required funds are available in the module account
	// funds expected as it's validated in the genesis transfer handler
	balance := k.BK.GetBalance(ctx, k.AK.GetModuleAddress(types.ModuleName), rollappIBCDenom)
	if !balance.Amount.Equal(plan.TotalAllocation.Amount) {
		return errorsmod.Wrapf(gerrc.ErrInternal, "required: %s, available: %s", plan.TotalAllocation.String(), balance.String())
	}

	// burn all the remaining IRO token.
	iroTokenBalance := k.BK.GetBalance(ctx, k.AK.GetModuleAddress(types.ModuleName), plan.TotalAllocation.Denom)
	err := k.BK.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(iroTokenBalance))
	if err != nil {
		return err
	}

	raisedLiquidityAmt := k.BK.GetBalance(ctx, plan.GetAddress(), plan.LiquidityDenom).Amount
	poolTokens := raisedLiquidityAmt.ToLegacyDec().Mul(plan.LiquidityPart).TruncateInt()
	ownerTokens := raisedLiquidityAmt.Sub(poolTokens)

	// start the vesting schedule for the owner tokens
	plan.VestingPlan.Amount = ownerTokens
	plan.VestingPlan.StartTime = ctx.BlockHeader().Time.Add(plan.VestingPlan.StartTimeAfterSettlement)
	plan.VestingPlan.EndTime = plan.VestingPlan.StartTime.Add(plan.VestingPlan.VestingDuration)

	// mark the plan as `settled`, allowing users to claim tokens
	plan.SettledDenom = rollappIBCDenom
	k.SetPlan(ctx, plan)

	// uses the raised liquidity and unsold tokens to bootstrap the rollapp's liquidity pool
	poolID, gaugeID, err := k.bootstrapLiquidityPool(ctx, plan, poolTokens)
	if err != nil {
		return errors.Join(types.ErrFailedBootstrapLiquidityPool, err)
	}

	// Emit event
	err = uevent.EmitTypedEvent(ctx, &types.EventSettle{
		PlanId:        fmt.Sprintf("%d", plan.Id),
		RollappId:     rollappId,
		IBCDenom:      rollappIBCDenom,
		PoolId:        poolID,
		GaugeId:       gaugeID,
		VestingAmount: ownerTokens,
	})
	if err != nil {
		return err
	}

	return nil
}

// bootstrapLiquidityPool bootstraps the liquidity pool with the raised liquidity and unsold tokens.
//
// This function performs the following steps:
// - Sends the raised liquidity to the IRO module to be used as the pool creator.
// - Determines the required pool liquidity amounts to fulfill the last price.
// - Creates a balancer pool with the determined tokens and liquidity.
// - Uses leftover tokens as incentives to the pool LP token holders.
func (k Keeper) bootstrapLiquidityPool(ctx sdk.Context, plan types.Plan, poolTokens math.Int) (poolID, gaugeID uint64, err error) {
	// claimable amount is kept in the module account and used for user's claims
	claimableAmt := plan.SoldAmt.Sub(plan.ClaimedAmt)

	// the remaining tokens are used to bootstrap the liquidity pool
	unallocatedTokens := plan.TotalAllocation.Amount.Sub(claimableAmt)

	// send the raised liquidity token to the iro module as it will be used as the pool creator
	err = k.BK.SendCoinsFromAccountToModule(ctx, plan.GetAddress(), types.ModuleName, sdk.NewCoins(sdk.NewCoin(plan.LiquidityDenom, poolTokens)))
	if err != nil {
		return 0, 0, err
	}

	// find the raTokens needed to bootstrap the pool, to fulfill last price
	raTokens, liquidityTokens := types.CalcLiquidityPoolTokens(unallocatedTokens, poolTokens, plan.SpotPrice())
	rollappLiquidityCoin := sdk.NewCoin(plan.SettledDenom, raTokens)
	baseLiquidityCoin := sdk.NewCoin(plan.LiquidityDenom, liquidityTokens)

	// create pool
	gammGlobalParams := k.gk.GetParams(ctx).GlobalFees
	poolParams := balancer.NewPoolParams(gammGlobalParams.SwapFee, gammGlobalParams.ExitFee, nil)
	balancerPool := balancer.NewMsgCreateBalancerPool(k.AK.GetModuleAddress(types.ModuleName), poolParams, []balancer.PoolAsset{
		{
			Token:  baseLiquidityCoin,
			Weight: math.OneInt(),
		},
		{
			Token:  rollappLiquidityCoin,
			Weight: math.OneInt(),
		},
	}, "")

	// we call the pool manager directly, instead of the gamm keeper, to avoid the pool creation fee
	poolId, err := k.pm.CreatePool(ctx, balancerPool)
	if err != nil {
		return 0, 0, err
	}

	// Add incentives
	poolDenom := gammtypes.GetPoolShareDenom(poolId)
	incentives := sdk.NewCoins(
		sdk.NewCoin(baseLiquidityCoin.Denom, poolTokens.Sub(baseLiquidityCoin.Amount)),
		sdk.NewCoin(rollappLiquidityCoin.Denom, unallocatedTokens.Sub(rollappLiquidityCoin.Amount)),
	)
	distrTo := lockuptypes.QueryCondition{
		Denom:    poolDenom,
		LockAge:  k.ik.GetParams(ctx).MinLockAge,
		Duration: k.ik.GetParams(ctx).MinLockDuration,
	}
	gaugeID, err = k.ik.CreateAssetGauge(ctx, false, k.AK.GetModuleAddress(types.ModuleName), incentives, distrTo, ctx.BlockTime().Add(plan.IncentivePlanParams.StartTimeAfterSettlement), plan.IncentivePlanParams.NumEpochsPaidOver)
	if err != nil {
		return 0, 0, err
	}

	return poolID, gaugeID, nil
}
