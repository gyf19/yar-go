package yar

import (
	"applib/pool"
	"encoding/binary"
	"errors"

	"io"

	"strings"
	"sync"
)

const YAR_PROTOCOL_MAGIC_NUM = 0x80DFEC60
const YAR_PROTOCOL_VERSION = 0
const YAR_PROTOCOL_RESERVED = 0

var YAR_PROTOCOL_TOKEN = [32]byte{}
var YAR_PROVIDER = [32]byte{'Y', 'a', 'r', ' ', 'G', 'o', ' ', 'C', 'l', 'i', 'e', 'n', 't'}

var headerPool *sync.Pool
var bytePool *pool.BytePool
var packagers map[string]Packager

func init() {
	packagers = make(map[string]Packager)
	msgpack := newMsgpackPack()
	packagers["msgp"] = msgpack
	packagers["msgpack"] = msgpack
	json := newJsonPack()
	packagers["json"] = json

	bytePool = pool.NewBytePool()

	headerPool = &sync.Pool{
		New: func() interface{} {
			return &YarHeader{}
		},
	}
}

type YarHeader struct {
	id        int32  // transaction id
	version   uint16 // jsoncl version
	magic_num uint32 // default is: 0x80DFEC60
	reserved  uint32
	provider  [32]byte // reqeust from who
	token     [32]byte // request token, used for authentication
	body_len  uint32   // request body len
	packager  [8]byte  // packager
}

func (h *YarHeader) Reset() {
	h.body_len = 0
	h.id = 0
	h.reserved = 0
	h.magic_num = 0
	h.version = 0
}

type RawMessage []byte

// MarshalJSON returns *m as the JSON encoding of m.
func (m *RawMessage) MarshalJSON() ([]byte, error) {
	return *m, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (m *RawMessage) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}

// MarshalMsgpack returns *m as the msgpack encoding of m.
func (m *RawMessage) MarshalMsgpack() ([]byte, error) {
	return *m, nil
}

// UnmarshalMsgpack sets *m to a copy of data.
func (m *RawMessage) UnmarshalMsgpack(data []byte) error {
	if m == nil {
		return errors.New("msgpack.RawMessage: UnmarshalMsgpack on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}

type serverRequest struct {
	Header *YarHeader  `json:"_" msgpack:"_"`
	Id     int64       `json:"i" msgpack:"i"`
	Method string      `json:"m" msgpack:"m"`
	Params *RawMessage `json:"p" msgpack:"p"`
}

func (m *serverRequest) Reset() {
	m.Method = ""
	m.Params = nil
	m.Id = 0
	m.Header = nil
}

func (m *serverRequest) GetId() int64 {
	if m != nil && m.Id > 0 {
		return m.Id
	}
	return 0
}

func (m *serverRequest) GetMethod() string {
	if m != nil && m.Method != "" {
		return m.Method
	}
	return ""
}

type serverResponse struct {
	Id     int64        `json:"i" msgpack:"i"`
	Error  string       `json:"e" msgpack:"e"`
	Output string       `json:"o" msgpack:"o"`
	Status int          `json:"s" msgpack:"s"`
	Result *interface{} `json:"r" msgpack:"r"`
}

func (m *serverResponse) Reset() { *m = serverResponse{} }

func (m *serverResponse) GetId() int64 {
	if m != nil && m.Id > 0 {
		return m.Id
	}
	return 0
}

func (m *serverResponse) GetError() string {
	if m != nil && m.Error != "" {
		return m.Error
	}
	return ""
}

type clientRequest struct {
	Id     int64       `json:"i" msgpack:"i"`
	Method string      `json:"m" msgpack:"m"`
	Params interface{} `json:"p" msgpack:"p"`
}

type clientResponse struct {
	Header *YarHeader  `json:"_" msgpack:"_"`
	Id     int64       `json:"i" msgpack:"i"`
	Error  string      `json:"e" msgpack:"e"`
	Output string      `json:"o" msgpack:"o"`
	Status int         `json:"s" msgpack:"s"`
	Result *RawMessage `json:"r" msgpack:"r"`
}

func (r *clientResponse) reset() {
	r.Id = 0
	r.Result = nil
	r.Error = ""
	r.Header = nil
}

type Packager interface {
	Unmarshal(data []byte, x interface{}) error
	Marshal(interface{}) ([]byte, error)
	GetName() [8]byte
}

func getPackager(name string) (Packager, error) {
	packer, ok := packagers[name]
	if !ok {
		return nil, errors.New("not packager")
	}
	return packer, nil
}

func getPackagerBybyte(buf [8]byte) (Packager, error) {
	name := strings.ToLower(string(buf[:4]))
	return getPackager(name)
}

func readPack(r io.Reader, x interface{}) (Packager, error) {
	header := headerPool.Get().(*YarHeader)
	defer headerPool.Put(header)
	header.Reset()

	binary.Read(r, binary.BigEndian, &header.id)
	binary.Read(r, binary.BigEndian, &header.version)
	binary.Read(r, binary.BigEndian, &header.magic_num)
	binary.Read(r, binary.BigEndian, &header.reserved)
	binary.Read(r, binary.BigEndian, &header.provider)
	binary.Read(r, binary.BigEndian, &header.token)
	binary.Read(r, binary.BigEndian, &header.body_len)
	binary.Read(r, binary.BigEndian, &header.packager)

	if header.body_len < 8 || header.body_len > 2*1024*1024 {
		return nil, errors.New("yar: Response header missing params")
	}
	body_len := (int)(header.body_len - 8)
	//data := make([]byte, body_len)
	data := bytePool.Get(body_len)
	defer bytePool.Put(data)
	n, err := io.ReadFull(r, data)
	if n != body_len {
		return nil, errors.New("yar: readPack body len error")
	}
	packer, err := getPackagerBybyte(header.packager)
	if err != nil {
		return nil, err
	}
	err = packer.Unmarshal(data, x)
	if err != nil {
		return nil, err
	}
	return packer, nil
}

func writePack(w io.Writer, packer Packager, Id int32, x interface{}) error {
	data, err := packer.Marshal(x)
	if err != nil {
		return err
	}
	header := headerPool.Get().(*YarHeader)
	defer headerPool.Put(header)

	header.id = Id
	header.version = YAR_PROTOCOL_VERSION //uint16(0)
	header.magic_num = YAR_PROTOCOL_MAGIC_NUM
	header.reserved = YAR_PROTOCOL_RESERVED
	header.provider = YAR_PROVIDER
	header.token = YAR_PROTOCOL_TOKEN
	header.body_len = uint32(len(data) + 8)
	header.packager = packer.GetName()

	binary.Write(w, binary.BigEndian, header)

	_, err = w.Write(data)
	if err != nil {
		return err
	}
	return nil
}
