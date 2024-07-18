package p2p

import "net"

const (
	IncomingMessage = 0x1
	IncomingStream  = 0x2
)

type Msg struct {
	From    net.Addr
	Payload []byte
	Stream  bool
}
