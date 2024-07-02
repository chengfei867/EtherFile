package p2p

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

type TCPTransportOpts struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

// TCPTransport 实现Transport接口 需要维护对等点信息
type TCPTransport struct {
	TCPTransportOpts
	listerner net.Listener
	rc        chan RPC

	sync.RWMutex
	peers map[net.Addr]*Peer
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rc:               make(chan RPC),
		peers:            make(map[net.Addr]*Peer),
	}
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listerner, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}
	go t.startAcceptLoop()
	return nil
}

// Consume 返回一个只读channel，消费其中的来自网络中另外的peer的消息
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rc
}

// 轮询监听请求
func (t *TCPTransport) startAcceptLoop() net.Conn {
	for {
		conn, err := t.listerner.Accept()
		if err != nil {
			fmt.Println("TCP: Accept error:", err)
		}
		// 每有一个请求到来创建一个协程处理
		go t.handleConn(conn)
	}
}

// 处理请求
func (t *TCPTransport) handleConn(conn net.Conn) {
	var err error
	defer func() {
		_ = conn.Close()
	}()
	peer := NewTCPPeer(conn, true)
	// 握手
	if err = t.HandshakeFunc(peer); err != nil {
		fmt.Printf("TCP: handshake error: %v\n", err)
		return
	}

	// 握手成功后进行OnPeer(回调函数 允许一些自定义逻辑)
	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			fmt.Printf("TCP: OnPeer error: %v\n", err)
			return
		}
	}
	rpc := new(RPC)
	for {
		if errors.Is(err, net.ErrClosed) {
			fmt.Println("TCP: connection closed")
			return
		}
		if err = t.Decoder.Decode(conn, rpc); err != nil {
			fmt.Println("TCP: decoder error:", err)
		}
		rpc.From = conn.RemoteAddr()
		t.rc <- *rpc
		//fmt.Printf("TCP: from: %s message: %v\n", conn.RemoteAddr(), string(rpc.Payload))
	}
}
