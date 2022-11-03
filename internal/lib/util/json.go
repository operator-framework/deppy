package util

import (
	"bytes"
	"encoding/json"
)

func JSONMarshal(p interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}
	dec := json.NewEncoder(buf)
	dec.SetEscapeHTML(false)
	err := dec.Encode(p)
	if err != nil {
		return nil, err
	}
	out := &bytes.Buffer{}
	if err := json.Compact(out, buf.Bytes()); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func JSONUnmarshal(p []byte, out interface{}) error {
	buf := bytes.NewReader(p)
	dec := json.NewDecoder(buf)
	return dec.Decode(out)
}
