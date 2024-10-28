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
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/filecoin-project/go-jsonrpc"
)

type AdminAPI struct {
	Pause              func() error
	Unpause            func() error
	DestructStateChild func(height uint64, root common.Hash, child common.Hash) error
	ForceProcessBlock  func(ctx context.Context, hash common.Hash) error
}

func NewClient(ctx context.Context, port int) (AdminAPI, jsonrpc.ClientCloser, error) {
	var client AdminAPI
	closer, err := jsonrpc.NewClient(ctx, fmt.Sprintf("http://localhost:%v", port), "admin", &client, http.Header{})
	return client, closer, err
}
