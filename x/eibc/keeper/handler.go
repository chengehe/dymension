package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	"github.com/dymensionxyz/sdk-utils/utils/uevent"

	denomutils "github.com/dymensionxyz/dymension/v3/utils/denom"
	commontypes "github.com/dymensionxyz/dymension/v3/x/common/types"
	dacktypes "github.com/dymensionxyz/dymension/v3/x/delayedack/types"
	"github.com/dymensionxyz/dymension/v3/x/eibc/types"
)

// EIBCDemandOrderHandler handles the eibc packet by creating a demand order from the packet data and saving it in the store.
// the rollapp packet can be of type ON_RECV or ON_TIMEOUT/ON_ACK (with ack error).
// If the rollapp packet is of type ON_RECV, the function will validate the memo and create a demand order from the packet data.
// If the rollapp packet is of type ON_TIMEOUT/ON_ACK, the function will calculate the fee and create a demand order from the packet data.
func (k Keeper) EIBCDemandOrderHandler(ctx sdk.Context, rollappPacket commontypes.RollappPacket, data transfertypes.FungibleTokenPacketData) error {
	var (
		eibcDemandOrder *types.DemandOrder
		err             error
	)
	// Validate the fungible token packet data as we're going to use it to create the demand order
	if err := data.ValidateBasic(); err != nil {
		return err
	}
	// Verify the original recipient is not a blocked sender otherwise could potentially use eibc to bypass it
	if k.BlockedAddr(data.Receiver) {
		return types.ErrBlockedAddress
	}

	switch t := rollappPacket.Type; t {
	case commontypes.RollappPacket_ON_RECV:
		eibcDemandOrder, err = k.CreateDemandOrderOnRecv(ctx, data, &rollappPacket)
	case commontypes.RollappPacket_ON_TIMEOUT, commontypes.RollappPacket_ON_ACK:
		eibcDemandOrder, err = k.CreateDemandOrderOnErrAckOrTimeout(ctx, data, &rollappPacket)
	}
	if err != nil {
		return fmt.Errorf("create eibc demand order: %w", err)
	}
	if eibcDemandOrder == nil {
		return nil
	}
	if err := eibcDemandOrder.Validate(); err != nil {
		return fmt.Errorf("validate eibc data: %w", err)
	}
	err = k.SetDemandOrder(ctx, eibcDemandOrder)
	if err != nil {
		return fmt.Errorf("set eibc demand order: %w", err)
	}

	if err = uevent.EmitTypedEvent(ctx, types.GetCreatedEvent(eibcDemandOrder, rollappPacket.ProofHeight, data.Amount)); err != nil {
		return fmt.Errorf("emit event: %w", err)
	}

	return nil
}

// CreateDemandOrderOnRecv creates a demand order from an IBC packet.
// It extracts the fee from the memo,calculates the demand order price, and creates a new demand order.
// price calculated with the fee and the bridging fee. (price = amount - fee - bridging fee)
// It returns the created demand order or an error if there is any.
func (k *Keeper) CreateDemandOrderOnRecv(ctx sdk.Context, fungibleTokenPacketData transfertypes.FungibleTokenPacketData,
	rollappPacket *commontypes.RollappPacket,
) (*types.DemandOrder, error) {
	memoEIBC, err := GetEIBCMemo(fungibleTokenPacketData.Memo)
	if err != nil {
		return nil, fmt.Errorf("unpack fungible packet memo: %w", err)
	}

	// Calculate the demand order price and validate it,
	amt, _ := math.NewIntFromString(fungibleTokenPacketData.Amount) // guaranteed ok and positive by above validation
	fee, _ := memoEIBC.FeeInt()                                     // guaranteed ok by above validation
	demandOrderPrice, err := types.CalcPriceWithBridgingFee(amt, fee, k.dack.BridgingFee(ctx))
	if err != nil {
		return nil, err
	}

	demandOrderDenom := denomutils.GetIncomingTransferDenom(*rollappPacket.Packet, fungibleTokenPacketData)
	demandOrderRecipient := fungibleTokenPacketData.Receiver // who we tried to send to
	creationHeight := uint64(ctx.BlockHeight())              //nolint:gosec // block height is always positive

	onComplete, err := memoEIBC.GetCompletionHook()
	if err != nil {
		return nil, fmt.Errorf("get on complete hook: %w", err)
	}
	if onComplete != nil {
		if err := k.dack.ValidateCompletionHook(*onComplete); err != nil {
			return nil, fmt.Errorf("validate on complete hook: %w", err)
		}
	}

	order := types.NewDemandOrder(*rollappPacket, demandOrderPrice, fee, demandOrderDenom, demandOrderRecipient, creationHeight, onComplete)
	return order, nil
}

func GetEIBCMemo(memoS string) (dacktypes.EIBCMemo, error) {
	if memoS == "" {
		return dacktypes.DefaultEIBCMemo(), nil
	}
	m, err := dacktypes.ParseMemo(memoS)
	if err != nil {
		if errorsmod.IsOf(err, dacktypes.ErrEIBCMemoEmpty) {
			return dacktypes.DefaultEIBCMemo(), nil
		}
		return dacktypes.EIBCMemo{}, fmt.Errorf("parse packet metadata: %w", err)
	}
	return *m.EIBC, m.EIBC.ValidateBasic()
}

// CreateDemandOrderOnErrAckOrTimeout creates a demand order for a timeout or errack packet.
// The fee multiplier is read from params and used to calculate the fee.
func (k Keeper) CreateDemandOrderOnErrAckOrTimeout(ctx sdk.Context, fungibleTokenPacketData transfertypes.FungibleTokenPacketData,
	rollappPacket *commontypes.RollappPacket,
) (*types.DemandOrder, error) {
	// Calculate the demand order price and validate it,
	amt, _ := math.NewIntFromString(fungibleTokenPacketData.Amount) // guaranteed ok and positive by above validation

	// Calculate the fee by multiplying the fee by the price
	var feeMultiplier math.LegacyDec
	switch rollappPacket.Type {
	case commontypes.RollappPacket_ON_TIMEOUT:
		feeMultiplier = k.TimeoutFee(ctx)
	case commontypes.RollappPacket_ON_ACK:
		feeMultiplier = k.ErrAckFee(ctx)
	}
	fee := feeMultiplier.MulInt(amt).TruncateInt()
	if !fee.IsPositive() {
		ctx.Logger().Debug("fee is not positive, skipping demand order creation", "packet", rollappPacket.LogString())
		return nil, nil
	}
	demandOrderPrice := amt.Sub(fee)

	trace := transfertypes.ParseDenomTrace(fungibleTokenPacketData.Denom)
	demandOrderDenom := trace.IBCDenom()
	demandOrderRecipient := fungibleTokenPacketData.Sender // and who tried to send it (refund because it failed)
	creationHeight := uint64(ctx.BlockHeight())            //nolint:gosec // block height is always positive

	order := types.NewDemandOrder(*rollappPacket, demandOrderPrice, fee, demandOrderDenom, demandOrderRecipient, creationHeight, nil)
	return order, nil
}

func (k Keeper) BlockedAddr(addr string) bool {
	account, err := sdk.AccAddressFromBech32(addr)
	if err != nil {
		return false
	}
	return k.bk.BlockedAddr(account)
}
