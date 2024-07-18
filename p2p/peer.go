package p2p

import (
	"net"
	"sync"
)

// Peer 代表网络中的对等节点
type Peer interface {
	net.Conn
	Send([]byte) error
	Close() error
	CloseStream()
}

// TCPPeer 代表一个TCP客户端节点
type TCPPeer struct {
	net.Conn
	// 当前主动发起连接 则为一个出站节点 该值为true
	// 被动接收其他节点连接 则为一个入站节点 该值为false
	outbound bool
	wg       sync.WaitGroup
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
	}
}

func (p *TCPPeer) CloseStream() {
	p.wg.Done()
}

func (p *TCPPeer) Send(msg []byte) error {
	_, err := p.Write(msg)
	return err
}

func (p *TCPPeer) Close() error {
	return p.Close()
}
