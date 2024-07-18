package p2p

import (
	"errors"
	"fmt"
	"log"
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
	rc        chan Msg

	sync.RWMutex
	peers map[net.Addr]*Peer
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rc:               make(chan Msg),
		peers:            make(map[net.Addr]*Peer),
	}
}

func (t *TCPTransport) ListenAddr() string {
	return t.TCPTransportOpts.ListenAddr
}

// Dial 向其他节点发起建立连接
func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	go t.handleConn(conn, true)
	return nil
}

// 轮询监听请求
func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listerner.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Println("TCP transport closed")
				return
			}
			fmt.Println("TCP: Accept error:", err)
		}
		// 每有一个请求到来创建一个协程处理
		go func() {
			//log.Println("TCP: new connection from", conn.RemoteAddr())
			t.handleConn(conn, false)
		}()
	}
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listerner, err = net.Listen("tcp", t.ListenAddr())
	if err != nil {
		return err
	}
	go t.startAcceptLoop()
	log.Printf("TCP transport listening on %s\n", t.ListenAddr())
	return nil
}

// Consume 返回一个只读channel，消费其中的来自网络中另外的peer的消息
func (t *TCPTransport) Consume() <-chan Msg {
	return t.rc
}

// 处理请求
func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	var err error
	defer func() {
		_ = conn.Close()
	}()

	// peer的conn和其transport的conn是同一个
	peer := NewTCPPeer(conn, outbound)
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
	// 阻塞读
	for {
		msg := Msg{}
		if errors.Is(err, net.ErrClosed) {
			fmt.Println("TCP: connection closed")
			return
		}
		if err = t.Decoder.Decode(conn, &msg); err != nil {
			fmt.Println("TCP: decoder error:", err)
		}
		msg.From = conn.RemoteAddr()
		if msg.Stream {
			peer.wg.Add(1)
			log.Println("TCP: incoming stream...")
			peer.wg.Wait()
			log.Println("TCP: completed received stream...")
			continue
		}
		t.rc <- msg
		//log.Println("TCP: message delivered.")
		//fmt.Printf("TCP: from: %s message: %v\n", conn.RemoteAddr(), string(rpc.Payload))
	}
}

// Close 实现transport结构
func (t *TCPTransport) Close() error {
	close(t.rc)
	return t.listerner.Close()
}
