package p2p

// Transport 处理网络中节点之间的传输，多种新式(TCP,UDP,Websockets...)
type Transport interface {
	ListenAndAccept() error
	Dial(string) error
	Consume() <-chan Msg
	Close() error
	ListenAddr() string
}
