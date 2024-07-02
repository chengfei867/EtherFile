package p2p

// Peer 代表网络中的对等节点
type Peer interface {
	Close() error
}

// Transport 处理网络中节点之间的传输，多种新式(TCP,UDP,Websockets...)
type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
}
