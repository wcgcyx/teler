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
	"reflect"
	"time"
	"unicode"

	"github.com/filecoin-project/go-jsonrpc"
	logging "github.com/ipfs/go-log"
	"github.com/wcgcyx/teler/node"
)

// Logger
var log = logging.Logger("rpc-server")

type Server struct {
	s *http.Server
}

// NewServer creates a new rpc server.
func NewServer(opts Opts, node *node.Node) (*Server, error) {
	log.Infof("Start API server...")
	// New jsonrpc server
	rpc := jsonrpc.NewServer()
	adminHandle := &adminAPIHandler{
		opts: opts,
		node: node,
		be:   node.Backend,
	}
	ethHandle := &ethAPIHandler{
		opts: opts,
		be:   node.Backend,
	}
	registerAndSetAlias(rpc, "eth", ethHandle)
	registerAndSetAlias(rpc, "admin", adminHandle)
	s := &http.Server{
		Addr:           fmt.Sprintf("%v:%v", opts.Host, opts.Port),
		Handler:        rpc,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	errChan := make(chan error, 1)
	go func() {
		// Start server.
		errChan <- s.ListenAndServe()
	}()
	// Wait for 3 seconds for the server to start
	tc := time.After(3 * time.Second)
	select {
	case <-tc:
		log.Infof("API Server started.")
		return &Server{s: s}, nil
	case err := <-errChan:
		return nil, err
	}
}

// register handlers and set alias.
func registerAndSetAlias(rpc *jsonrpc.RPCServer, namespace string, handler interface{}) {
	rpc.Register(namespace, handler)
	val := reflect.ValueOf(handler)
	lowerFirstLetter := func(s string) string {
		if len(s) == 0 {
			return s
		}
		r := []rune(s)
		r[0] = unicode.ToLower(r[0])
		return string(r)
	}
	for i := 0; i < val.NumMethod(); i++ {
		method := val.Type().Method(i)
		rpc.AliasMethod(namespace+"_"+lowerFirstLetter(method.Name), namespace+"."+method.Name)
	}
}

// Close graceful close the component.
func (s *Server) Shutdown() {
	log.Infof("Close API Server...")
	err := s.s.Shutdown(context.Background())
	if err != nil {
		log.Errorf("Fail to close API Server: %v", err.Error())
		return
	}
	log.Infof("API Server closed successfully.")
}
