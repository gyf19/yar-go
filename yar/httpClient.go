package yar

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

type clientHttpCodec struct {
	r *bufio.Reader
	w io.Writer
	c io.Closer

	host string
	path string

	// temporary work space
	req  clientRequest
	resp clientResponse

	packer Packager

	mutex   sync.Mutex        // protects pending
	pending map[uint64]string // map request id to method name
}

// NewClientCodec returns a new rpc.ClientCodec using Yar-RPC on conn.
func NewClientHtppCodec(conn io.ReadWriteCloser, u *url.URL, packagerName string) rpc.ClientCodec {
	packer, _ := getPackager(packagerName)
	return &clientHttpCodec{
		r:       bufio.NewReader(conn),
		w:       conn,
		c:       conn,
		host:    u.Host,
		path:    u.RequestURI(),
		pending: make(map[uint64]string),
		packer:  packer,
	}
}

func (c *clientHttpCodec) WriteRequest(r *rpc.Request, param interface{}) error {
	c.mutex.Lock()
	c.pending[r.Seq] = r.ServiceMethod
	c.mutex.Unlock()
	c.req.Method = r.ServiceMethod
	c.req.Params = []interface{}{&param}
	c.req.Id = (int64)(r.Seq)

	data, err := c.packer.Marshal(c.req)
	if err != nil {
		return err
	}
	//write http header
	httpLenth := strconv.Itoa(len(data) + 90)
	io.WriteString(c.w, "POST "+c.path+" HTTP/1.1\r\nHost: "+c.host+"\r\nContent-Type: application/x-www-form-urle\r\nUser-Agent: Go Yar Rpc-1.2.4\r\nConnection: keep-alive\r\nContent-Length: "+httpLenth+"\r\n\r\n")
	header := headerPool.Get().(*YarHeader)
	defer headerPool.Put(header)

	header.id = int32(c.req.Id)
	header.version = YAR_PROTOCOL_VERSION //uint16(0)
	header.magic_num = YAR_PROTOCOL_MAGIC_NUM
	header.reserved = YAR_PROTOCOL_RESERVED
	header.provider = YAR_PROVIDER
	header.token = YAR_PROTOCOL_TOKEN
	header.body_len = uint32(len(data) + 8)
	header.packager = c.packer.GetName()

	binary.Write(c.w, binary.BigEndian, header)

	_, err = c.w.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (c *clientHttpCodec) ReadResponseHeader(r *rpc.Response) error {
	c.resp.reset()

	//read http header
	tp := textproto.NewReader(c.r)
	line, err := tp.ReadLine()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return err
	}
	f := strings.SplitN(line, " ", 3)
	if len(f) < 2 {
		return errors.New("malformed HTTP response")
	}
	_, err = tp.ReadMIMEHeader()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return err
	}
	_, err = readPack(c.r, &c.resp)
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

func (c *clientHttpCodec) ReadResponseBody(x interface{}) error {
	if x == nil {
		return nil
	}
	return c.packer.Unmarshal(*c.resp.Result, x)
}

// Close closes the underlying connection.
func (c *clientHttpCodec) Close() error {
	return c.c.Close()
}

func NewHttpClient(conn io.ReadWriteCloser, u *url.URL, packagerName string) *rpc.Client {
	return rpc.NewClientWithCodec(NewClientHtppCodec(conn, u, packagerName))
}

func DialHTTP(urlStr string, packagerName string) (*rpc.Client, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	conn, err := net.Dial("tcp", u.Host)
	if err != nil {
		return nil, err
	}
	return NewHttpClient(conn, u, packagerName), nil
}
