package rpc

/*
 * Licensed under LGPL-3.0.
 *
 * You can get a copy of the LGPL-3.0 License at
 *
 * https://www.gnu.org/licenses/lgpl-3.0.en.html
 *
 * @wcgcyx - https://github.com/wcgcyx
 */

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/wcgcyx/teler/backend"
	"github.com/wcgcyx/teler/node"
)

// adminAPIHandler is used to handle admin API.
type adminAPIHandler struct {
	opts Opts

	node *node.Node
	be   backend.Backend
}

func (h *adminAPIHandler) Pause() error {
	h.node.Pause()
	return nil
}

func (h *adminAPIHandler) Unpause() error {
	h.node.Unpause()
	return nil
}

func (h *adminAPIHandler) DestructStateChild(height uint64, root common.Hash, child common.Hash) error {
	return h.be.DebugDestructStateChild(height, root, child)
}

func (h *adminAPIHandler) ForceProcessBlock(ctx context.Context, hash common.Hash) error {
	exists, err := h.be.Blockchain().HasBlock(ctx, hash)
	if err != nil {
		return err
	}
	var blk *types.Block
	if exists {
		blk, err = h.be.Blockchain().GetBlockByHash(ctx, hash)
	} else {
		blk, err = h.node.BlkSrc.BlockByHash(ctx, hash)
	}
	if err != nil {
		return err
	}
	return h.be.DebugForceProcessBlock(ctx, blk)
}
