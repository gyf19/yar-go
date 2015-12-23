package yar

import (
	"encoding/json"
	"errors"
)

type jsonPack struct {
	PackagerName [8]byte
}

func newJsonPack() *jsonPack {
	name := [8]byte{'J', 'S', 'O', 'N'}
	return &jsonPack{
		PackagerName: name,
	}
}

func (m *jsonPack) Marshal(x interface{}) (data []byte, err error) {
	if x == nil {
		return nil, errors.New("yar: serverResponse null")
	}
	return json.Marshal(x)
}

func (m *jsonPack) GetName() [8]byte {
	return m.PackagerName
}

func (m *jsonPack) Unmarshal(data []byte, x interface{}) error {
	return json.Unmarshal(data, &x)
}
