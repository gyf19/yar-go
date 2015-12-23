// Copyright 2013 <chaishushan{AT}gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package yar

import (
	"errors"
	"io"
	"log"
	"net"
	"net/rpc"
	"sync"
)

var errMissingParams = errors.New("jsonrpc: request body missing params")

type serverCodec struct {
	r io.Reader
	w io.Writer
	c io.Closer

	// temporary work space
	req    serverRequest
	packer Packager

	mutex   sync.Mutex // protects seq, pending
	seq     uint64
	pending map[uint64]int64
}

// NewServerCodec returns a serverCodec that communicates with the ClientCodec
// on the other end of the given conn.
func NewServerCodec(conn io.ReadWriteCloser) rpc.ServerCodec {
	return &serverCodec{
		r:       conn,
		w:       conn,
		c:       conn,
		pending: make(map[uint64]int64),
	}
}

func (c *serverCodec) ReadRequestHeader(r *rpc.Request) error {
	c.req.Reset()
	packer, err := readPack(c.r, &c.req)
	if err != nil {
		return err
	}

	c.packer = packer
	r.ServiceMethod = c.req.Method
	c.mutex.Lock()
	c.seq++
	c.pending[c.seq] = c.req.Id
	c.req.Id = 0
	r.Seq = c.seq
	c.mutex.Unlock()

	return nil
}

func (c *serverCodec) ReadRequestBody(x interface{}) error {
	if x == nil {
		return nil
	}
	if c.req.Params == nil {
		return errMissingParams
	}
	return c.packer.Unmarshal(*c.req.Params, &x)
}

var invalidRequest = struct{}{}

func (c *serverCodec) WriteResponse(r *rpc.Response, x interface{}) error {
	c.mutex.Lock()
	id, ok := c.pending[r.Seq]
	if !ok {
		c.mutex.Unlock()
		return errors.New("invalid sequence number in response")
	}
	delete(c.pending, r.Seq)
	c.mutex.Unlock()

	resp := serverResponse{
		Id:     id,
		Error:  "",
		Result: nil,
		Output: "",
		Status: 0,
	}

	if r.Error == "" {
		resp.Result = &x
	} else {
		resp.Error = r.Error
	}

	Id := (int32)(resp.Id)
	err := writePack(c.w, c.packer, Id, &resp)
	if err != nil {
		return err
	}
	return nil
}

func (s *serverCodec) Close() error {
	return s.c.Close()
}

func ServeConn(conn io.ReadWriteCloser) {
	rpc.ServeCodec(NewServerCodec(conn))
}

var jsonRpcConnected = "200 Connected to JSON RPC"

type Server struct {
	*rpc.Server
}

func NewServer() *Server {
	return &Server{rpc.NewServer()}
}
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Fatal("rpc.Serve: accept:", err.Error()) // TODO(r): exit?
		}
		go server.ServeConn(conn)
	}
}
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	server.ServeCodec(NewServerCodec(conn))
}
