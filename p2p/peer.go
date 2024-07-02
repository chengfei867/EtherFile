package p2p

import "net"

// TCPPeer 代表一个TCP客户端节点
type TCPPeer struct {
	conn net.Conn

	// 当前主动发起连接 则为一个出站节点 该值为true
	// 被动接收其他节点连接 则为一个入站节点 该值为false
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{conn: conn, outbound: outbound}
}

func (p *TCPPeer) Close() error {
	return p.conn.Close()
}
