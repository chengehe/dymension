package types

import (
	"math"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgUpdateState{}

func NewMsgUpdateState(creator, rollappId, dAPath string, startHeight, numBlocks, revision uint64, bDs *BlockDescriptors) *MsgUpdateState {
	return &MsgUpdateState{
		Creator:         creator,
		RollappId:       rollappId,
		StartHeight:     startHeight,
		NumBlocks:       numBlocks,
		DAPath:          dAPath,
		BDs:             *bDs,
		RollappRevision: revision,
	}
}

func (msg *MsgUpdateState) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	// an update can't be with no BDs
	if msg.NumBlocks == uint64(0) {
		return errorsmod.Wrap(ErrInvalidNumBlocks, "number of blocks can not be zero")
	}

	if msg.NumBlocks > math.MaxUint64-msg.StartHeight {
		return errorsmod.Wrapf(ErrInvalidNumBlocks, "numBlocks(%d) + startHeight(%d) exceeds max uint64", msg.NumBlocks, msg.StartHeight)
	}

	// check to see that update contains all BDs
	if len(msg.BDs.BD) != int(msg.NumBlocks) { //nolint:gosec
		return errorsmod.Wrapf(ErrInvalidNumBlocks, "number of blocks (%d) != number of block descriptors(%d)", msg.NumBlocks, len(msg.BDs.BD))
	}

	// check to see that startHeight is not zaro
	if msg.StartHeight == 0 {
		return errorsmod.Wrapf(ErrWrongBlockHeight, "StartHeight must be greater than zero")
	}

	// check that the blocks are sequential by height
	for bdIndex := uint64(0); bdIndex < msg.NumBlocks; bdIndex += 1 {

		// Pre 3D rollapps will use zero DRS until they upgrade. Post 3D rollapps
		// should use a non-zero version. We rely on other fraud mechanisms
		// to catch that if it's wrong. So we don't check DRS.

		if msg.BDs.BD[bdIndex].Height != msg.StartHeight+bdIndex {
			return ErrInvalidBlockSequence
		}
		// check to see stateRoot is a 32 byte array
		if len(msg.BDs.BD[bdIndex].StateRoot) != 32 {
			return errorsmod.Wrapf(ErrInvalidStateRoot, "StateRoot of block high (%d) must be 32 byte array. But received (%d) bytes",
				msg.BDs.BD[bdIndex].Height, len(msg.BDs.BD[bdIndex].StateRoot))
		}
	}

	return nil
}
