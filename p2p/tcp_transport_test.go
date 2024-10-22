package p2p

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewTCPTransport(t *testing.T) {
	tcpT := NewTCPTransport(TCPTransportOpts{
		ListenAddr: ":8080",
	})
	assert.Nil(t, tcpT.ListenAndAccept())
}
