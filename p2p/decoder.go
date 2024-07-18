package p2p

import (
	"io"
)

type Decoder interface {
	Decode(io.Reader, *Msg) error
}

type DefaultDecoder struct{}

func (d DefaultDecoder) Decode(r io.Reader, msg *Msg) error {
	peekBuf := make([]byte, 1)
	if _, err := r.Read(peekBuf); err != nil {
		return err
	}
	//In case of a stream we are not decoding what is being sent over the network
	stream := peekBuf[0] == IncomingStream
	if stream {
		msg.Stream = true
		return nil
	}
	buf := make([]byte, 1028)
	n, err := r.Read(buf)
	if err != nil {
		return err
	}
	msg.Payload = buf[:n]
	return nil
}
