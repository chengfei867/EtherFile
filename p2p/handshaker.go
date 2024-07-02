package p2p

import "errors"

var ErrInvalidHandshake = errors.New("invalid handshake")

type HandshakeFunc func(Peer) error

func DefaultHandShakeFunc(Peer) error {
	return nil
}
