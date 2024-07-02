package main

import (
	"Etherfile/p2p"
	"fmt"
	"log"
)

func OnPeer(peer p2p.Peer) error {
	//fmt.Printf("TCP: new peer %v\n", peer)
	peer.Close()
	return nil
}

func main() {
	tcpT := p2p.NewTCPTransport(p2p.TCPTransportOpts{
		ListenAddr:    ":3030",
		HandshakeFunc: p2p.DefaultHandShakeFunc,
		Decoder:       new(p2p.DefaultDecoder),
		OnPeer:        OnPeer,
	})
	if err := tcpT.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}
	for {
		select {
		case msg := <-tcpT.Consume():
			fmt.Printf("%s\n", msg)
		}
	}
}
