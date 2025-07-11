package types_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/dymensionxyz/dymension/v3/app/apptesting"
	incentivestypes "github.com/dymensionxyz/dymension/v3/x/incentives/types"
	lockuptypes "github.com/dymensionxyz/dymension/v3/x/lockup/types"
)

// TestMsgCreateGauge tests if valid/invalid create gauge messages are properly validated/invalidated
func TestMsgCreateGauge(t *testing.T) {
	// generate a private/public key pair and get the respective address
	pk1 := ed25519.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pk1.Address())

	// make a proper createPool message
	createMsg := func(after func(msg incentivestypes.MsgCreateGauge) incentivestypes.MsgCreateGauge) incentivestypes.MsgCreateGauge {
		distributeTo := lockuptypes.QueryCondition{
			Denom:    "lptoken",
			Duration: time.Second,
		}

		properMsg := incentivestypes.MsgCreateGauge{
			IsPerpetual:       false,
			Owner:             addr1.String(),
			GaugeType:         incentivestypes.GaugeType_GAUGE_TYPE_ASSET,
			Asset:             &distributeTo,
			Coins:             sdk.Coins{sdk.NewInt64Coin("stake", 10)},
			StartTime:         time.Now(),
			NumEpochsPaidOver: 2,
		}

		return after(properMsg)
	}

	tests := []struct {
		name       string
		msg        incentivestypes.MsgCreateGauge
		expectPass bool
	}{
		{
			name: "proper msg",
			msg: createMsg(func(msg incentivestypes.MsgCreateGauge) incentivestypes.MsgCreateGauge {
				return msg
			}),
			expectPass: true,
		},
		{
			name: "empty owner",
			msg: createMsg(func(msg incentivestypes.MsgCreateGauge) incentivestypes.MsgCreateGauge {
				msg.Owner = ""
				return msg
			}),
			expectPass: false,
		},
		{
			name: "empty distribution denom",
			msg: createMsg(func(msg incentivestypes.MsgCreateGauge) incentivestypes.MsgCreateGauge {
				msg.Asset.Denom = ""
				return msg
			}),
			expectPass: false,
		},
		{
			name: "invalid distribution denom",
			msg: createMsg(func(msg incentivestypes.MsgCreateGauge) incentivestypes.MsgCreateGauge {
				msg.Asset.Denom = "111"
				return msg
			}),
			expectPass: false,
		},
		{
			name: "invalid distribution start time",
			msg: createMsg(func(msg incentivestypes.MsgCreateGauge) incentivestypes.MsgCreateGauge {
				msg.StartTime = time.Time{}
				return msg
			}),
			expectPass: false,
		},
		{
			name: "invalid num epochs paid over",
			msg: createMsg(func(msg incentivestypes.MsgCreateGauge) incentivestypes.MsgCreateGauge {
				msg.NumEpochsPaidOver = 0
				return msg
			}),
			expectPass: false,
		},
		{
			name: "invalid num epochs paid over for perpetual gauge",
			msg: createMsg(func(msg incentivestypes.MsgCreateGauge) incentivestypes.MsgCreateGauge {
				msg.NumEpochsPaidOver = 2
				msg.IsPerpetual = true
				return msg
			}),
			expectPass: false,
		},
		{
			name: "valid num epochs paid over for perpetual gauge",
			msg: createMsg(func(msg incentivestypes.MsgCreateGauge) incentivestypes.MsgCreateGauge {
				msg.NumEpochsPaidOver = 1
				msg.IsPerpetual = true
				return msg
			}),
			expectPass: true,
		},
	}

	for _, test := range tests {
		if test.expectPass {
			require.NoError(t, test.msg.ValidateBasic(), "test: %v", test.name)
		} else {
			require.Error(t, test.msg.ValidateBasic(), "test: %v", test.name)
		}
	}
}

// TestMsgAddToGauge tests if valid/invalid add to gauge messages are properly validated/invalidated
func TestMsgAddToGauge(t *testing.T) {
	// generate a private/public key pair and get the respective address
	pk1 := ed25519.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pk1.Address())

	// make a proper addToGauge message
	createMsg := func(after func(msg incentivestypes.MsgAddToGauge) incentivestypes.MsgAddToGauge) incentivestypes.MsgAddToGauge {
		properMsg := *incentivestypes.NewMsgAddToGauge(
			addr1,
			1,
			sdk.Coins{sdk.NewInt64Coin("stake", 10)},
		)

		return after(properMsg)
	}

	tests := []struct {
		name       string
		msg        incentivestypes.MsgAddToGauge
		expectPass bool
	}{
		{
			name: "proper msg",
			msg: createMsg(func(msg incentivestypes.MsgAddToGauge) incentivestypes.MsgAddToGauge {
				return msg
			}),
			expectPass: true,
		},
		{
			name: "empty owner",
			msg: createMsg(func(msg incentivestypes.MsgAddToGauge) incentivestypes.MsgAddToGauge {
				msg.Owner = ""
				return msg
			}),
			expectPass: false,
		},
		{
			name: "empty rewards",
			msg: createMsg(func(msg incentivestypes.MsgAddToGauge) incentivestypes.MsgAddToGauge {
				msg.Rewards = sdk.Coins{}
				return msg
			}),
			expectPass: false,
		},
	}

	for _, test := range tests {
		if test.expectPass {
			require.NoError(t, test.msg.ValidateBasic(), "test: %v", test.name)
		} else {
			require.Error(t, test.msg.ValidateBasic(), "test: %v", test.name)
		}
	}
}

// // Test authz serialize and de-serializes for incentives msg.
func TestAuthzMsg(t *testing.T) {
	app := apptesting.Setup(t)
	pk1 := ed25519.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pk1.Address()).String()
	coin := sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(1))
	someDate := time.Date(1, 1, 1, 1, 1, 1, 1, time.UTC)

	testCases := []struct {
		name          string
		incentivesMsg sdk.Msg
	}{
		{
			name: "MsgAddToGauge",
			incentivesMsg: &incentivestypes.MsgAddToGauge{
				Owner:   addr1,
				GaugeId: 1,
				Rewards: sdk.NewCoins(coin),
			},
		},
		{
			name: "MsgCreateGauge",
			incentivesMsg: &incentivestypes.MsgCreateGauge{
				IsPerpetual: false,
				Owner:       addr1,
				GaugeType:   incentivestypes.GaugeType_GAUGE_TYPE_ASSET,
				Asset: &lockuptypes.QueryCondition{
					Denom:    "lptoken",
					Duration: time.Second,
				},
				Coins:             sdk.NewCoins(coin),
				StartTime:         someDate,
				NumEpochsPaidOver: 1,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			apptesting.TestMessageAuthzSerialization(t, app.AppCodec(), tc.incentivesMsg)
		})
	}
}
