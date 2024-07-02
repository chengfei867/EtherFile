package p2p

import (
	"io"
)

type Decoder interface {
	Decode(io.Reader, *RPC) error
}

type DefaultDecoder struct{}

func (d DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
	msg.Payload = make([]byte, 1024)
	_, err := r.Read(msg.Payload)
	if err != nil {
		return err
	}
	return nil
}
