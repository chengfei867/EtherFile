package main

import (
	"Etherfile/p2p"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"time"
)

func makeServer(addr string, nodes ...string) *FileServer {
	trOpts := p2p.TCPTransportOpts{
		ListenAddr:    addr,
		HandshakeFunc: p2p.DefaultHandShakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		// ToDo OnPeer func
	}
	key, _ := hex.DecodeString("984eb1fdd6e12dfcf5bf0a8c71c3cb65d7d4506b392bf2f56051cc025ad37a6d")
	transport := p2p.NewTCPTransport(trOpts)
	fileServerOpts := FileServerOpts{
		Encrypter:         NewDefaultEncrypter(key),
		ListenAddr:        addr,
		StorageRoot:       addr + "_path",
		PathTransformFunc: SHA1PathTransformFunc,
		Transport:         transport,
		BootstrapNodes:    nodes,
	}
	fs := NewFileServer(fileServerOpts)
	transport.OnPeer = fs.OnPeer
	return fs

}

func main() {
	fs1 := makeServer(":3000")
	fs2 := makeServer(":3001", ":3000")
	fs3 := makeServer(":3002", ":3000", ":3001")
	//go func() {
	//	time.Sleep(3 * time.Second)
	//	fs.quit <- struct{}{}
	//}()
	go func() {
		if err := fs1.Start(); err != nil {
			log.Printf("Error starting server: %v", err)
		}
	}()
	time.Sleep(1000 * time.Millisecond)
	go func() {
		if err := fs2.Start(); err != nil {
			log.Printf("Error starting server: %v", err)
		}
	}()
	time.Sleep(1000 * time.Millisecond)
	go func() {
		if err := fs3.Start(); err != nil {
			log.Printf("Error starting server: %v", err)
		}
	}()
	//time.Sleep(1000 * time.Millisecond)
	//data := []byte("This is a large file")
	////for i := 0; i < 1; i++ {
	//reader := bytes.NewReader(data)
	//if err := fs3.Store("my_file", reader); err != nil {
	//	log.Printf("Error storing data: %v", err)
	//}

	time.Sleep(100 * time.Millisecond)
	f, err := fs1.Get("my_file")
	if err != nil {
		log.Fatalf("Error getting file: %v", err)
	}
	fileData, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Error reading file: %v\n", err)
	}
	fmt.Printf("filedata : %s\n", string(fileData))
	select {}
}
