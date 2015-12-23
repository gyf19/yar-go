// Copyright 2013 <chaishushan{AT}gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package yar

import (
	"fmt"
	"io"
	"net"
	"net/rpc"
	"sync"
	"time"
)

type clientCodec struct {
	r io.Reader
	w io.Writer
	c io.Closer

	// temporary work space
	req  clientRequest
	resp clientResponse

	packer Packager

	mutex   sync.Mutex        // protects pending
	pending map[uint64]string // map request id to method name
}

// NewClientCodec returns a new rpc.ClientCodec using Yar-RPC on conn.
func NewClientCodec(conn io.ReadWriteCloser, packagerName string) rpc.ClientCodec {
	packer, _ := getPackager(packagerName)
	return &clientCodec{
		r:       conn,
		w:       conn,
		c:       conn,
		pending: make(map[uint64]string),
		packer:  packer,
	}
}

func (c *clientCodec) WriteRequest(r *rpc.Request, param interface{}) error {
	c.mutex.Lock()
	c.pending[r.Seq] = r.ServiceMethod
	c.mutex.Unlock()
	c.req.Method = r.ServiceMethod
	c.req.Params = &param
	c.req.Id = (int64)(r.Seq)
	Id := (int32)(c.req.Id)
	err := writePack(c.w, c.packer, Id, &c.req)
	if err != nil {
		return err
	}
	return nil
}

func (c *clientCodec) ReadResponseHeader(r *rpc.Response) error {
	c.resp.reset()

	_, err := readPack(c.r, &c.resp)
	if err != nil {
		return err
	}

	c.mutex.Lock()
	Id := (uint64)(c.resp.Id)
	r.ServiceMethod = c.pending[Id]
	delete(c.pending, Id)
	c.mutex.Unlock()

	r.Error = ""
	r.Seq = Id
	if c.resp.Error != "" || c.resp.Result == nil {
		fmt.Errorf("invalid error %v", c.resp.Error)
		r.Error = c.resp.Error
	}
	return nil
}

func (c *clientCodec) ReadResponseBody(x interface{}) error {
	if x == nil {
		return nil
	}
	return c.packer.Unmarshal(*c.resp.Result, x)
}

// Close closes the underlying connection.
func (c *clientCodec) Close() error {
	return c.c.Close()
}

// NewClient returns a new rpc.Client to handle requests to the
// set of services at the other end of the connection.
func NewClient(conn io.ReadWriteCloser, packagerName string) *rpc.Client {
	return rpc.NewClientWithCodec(NewClientCodec(conn, packagerName))
}

// Dial connects to a Yar-RPC server at the specified network address.
func Dial(network, address string, packagerName string) (*rpc.Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return NewClient(conn, packagerName), err
}

// DialTimeout connects to a Yar-RPC server at the specified network address.
func DialTimeout(network, address string, timeout time.Duration, packagerName string) (*rpc.Client, error) {
	conn, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return nil, err
	}
	return NewClient(conn, packagerName), err
}
