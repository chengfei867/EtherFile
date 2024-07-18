package main

import (
	"Etherfile/p2p"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"
)

type FileServerOpts struct {
	Encrypter         Encrypter
	ListenAddr        string
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	sync.Mutex
	peers map[string]p2p.Peer

	store *Store
	quit  chan struct{}
}

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	Key  string
	Size int64
}

type MessageGetFile struct {
	Key string
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}
	return &FileServer{
		FileServerOpts: opts,
		peers:          make(map[string]p2p.Peer),
		store:          NewStore(storeOpts),
		quit:           make(chan struct{}),
	}
}

func (fs *FileServer) Start() error {
	err := fs.Transport.ListenAndAccept()
	if err != nil {
		return fmt.Errorf("Error listening on %s: %s\n", fs.ListenAddr, err)
	}
	fs.bootstrapNetwork()
	fs.loop()
	return nil
}

func (fs *FileServer) bootstrapNetwork() {
	for _, addr := range fs.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}
		go func(addr string) {
			if err := fs.Transport.Dial(addr); err != nil {
				log.Printf("Error connecting to %s: %s\n", addr, err)
			}
		}(addr)
	}
}

// Store 存储函数 将文件存在本地 并且广播到整个网络进行备份存储
func (fs *FileServer) Store(key string, r io.Reader) error {
	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)

	//加密存储到本地
	err := fs.store.writeEncrypt(key, fs.Encrypter, tee)
	if err != nil {
		return err
	}

	// 广播发送存储文件命令到网络中其他节点进行分布式存储备份
	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: int64(fileBuffer.Len() + DefaultIVSize),
		},
	}
	fs.broadcast(&msg)

	// 发送待存储文件至所有peer
	time.Sleep(10 * time.Millisecond)
	fs.stream(fileBuffer.Bytes())
	return nil
}

// 广播消息到所有对等点
func (fs *FileServer) broadcast(msg *Message) {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		log.Printf("Error encoding message: %v\n", err)
	}
	for _, peer := range fs.peers {
		go func(p p2p.Peer) {
			err := p.Send([]byte{p2p.IncomingMessage})
			if err = p.Send(buf.Bytes()); err != nil {
				log.Fatalf("Error sending message to %s: %s\n", p, err)
			}
			log.Printf("[%s] send msg to %s\n", fs.ListenAddr, peer.RemoteAddr())
		}(peer)
	}
}

// 向所有peer传输文件
func (fs *FileServer) stream(fileDataStream []byte) {
	for _, peer := range fs.peers {
		go func(p p2p.Peer) {
			err := peer.Send([]byte{p2p.IncomingStream})
			//_, err = io.Copy(peer, bytes.NewReader(fileDataStream))
			//加密传输
			_, err = fs.Encrypter.Encrypt(fs.Encrypter.Key(), bytes.NewReader(fileDataStream), peer)
			if err != nil {
				return
			}
			if err != nil {
				log.Fatalf("Error streaming data to %s: %s\n", peer, err)
			}
			log.Printf("[%s] send file to %s\n", fs.ListenAddr, peer.RemoteAddr())
		}(peer)
	}
}

func (fs *FileServer) Get(key string) (io.Reader, error) {
head:
	if fs.store.Exists(key) {
		log.Printf("[%s] file : %s exists\n", fs.ListenAddr, key)
		dst := new(bytes.Buffer)
		err := fs.store.ReadDecrypt(key, fs.Encrypter, dst)
		if err != nil {
			return nil, err
		}
		return dst, nil
	}
	log.Printf("[%s] file not found,will search on network..", fs.ListenAddr)
	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}
	fs.broadcast(&msg)
	time.Sleep(1 * time.Second)
	fileBuffer := new(bytes.Buffer)
	fileBufferCh := make(chan struct{})
	for _, peer := range fs.peers {
		go func(p p2p.Peer) {
			defer p.CloseStream()
			fileBuffer = new(bytes.Buffer)
			fileSize := int64(0)
			err := binary.Read(p, binary.LittleEndian, &fileSize)
			if err != nil {
				return
			}
			_, err = io.CopyN(fileBuffer, p, fileSize)
			log.Printf("[%s] get file from peer %s\n", fs.ListenAddr, p.RemoteAddr())
			if err != nil {
				log.Fatalf("Error reading data from %s: %s\n", p, err)
			}
			if fileBuffer.Len() > 0 {
				fileBufferCh <- struct{}{}
			}
		}(peer)
	}
	for {
		select {
		case <-fileBufferCh:
			// 将文件写入到本地
			err := fs.store.Write(key, fileBuffer)
			if err != nil {
				return nil, err
			}
			goto head
		case <-time.After(5 * time.Second):
			return nil, fmt.Errorf("timeout waiting for file to exist")
		}
	}
}

func (fs *FileServer) loop() {
	for {
		select {
		case msg := <-fs.Transport.Consume():
			var m Message
			if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&m); err != nil {
				log.Printf("Error decoding message: %s", err)
			}
			if err := fs.handlerMsg(msg.From.String(), &m); err != nil {
				log.Println("Error handling message:", err)
			}
		case <-fs.quit:
			fs.Stop()
			return
		}
	}
}

func (fs *FileServer) handlerMsg(from string, msg *Message) error {
	switch m := msg.Payload.(type) {
	case MessageStoreFile:
		return fs.handleMsgStoreFile(from, m)
	case MessageGetFile:
		return fs.handleMsgGetFile(from, m)
	default:
		log.Printf("Unrecognized message from %s", m)
	}
	return nil
}

// 处理文件存储的请求
func (fs *FileServer) handleMsgStoreFile(from string, msg MessageStoreFile) error {
	//log.Printf("Handling store file message from %s cmd : %s\n", from, msg.Key)
	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer %s not found", from)
	}
	defer func() {
		log.Printf("[%s] Compting store file from %s", fs.ListenAddr, from)
		peer.CloseStream()
	}()
	return fs.store.Write(msg.Key, io.LimitReader(peer, msg.Size))
}

// 处理获取文件的请求
func (fs *FileServer) handleMsgGetFile(from string, msg MessageGetFile) error {
	if !fs.store.Exists(msg.Key) {
		return fmt.Errorf("file not found on %s\n", from)
	}
	n, r, err := fs.store.Read(msg.Key)
	defer func(r io.ReadCloser) {
		_ = r.Close()
	}(r)
	if err != nil {
		return err
	}
	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer %s not found", from)
	}
	// copy文件给广播节点
	err = peer.Send([]byte{p2p.IncomingStream})
	fileSize := n
	err = binary.Write(peer, binary.LittleEndian, &fileSize)
	if err != nil {
		return err
	}
	if _, err = io.Copy(peer, r); err != nil {
		return err
	}
	log.Printf("[%s] find file %s,sending to %s\n", fs.ListenAddr, msg.Key, peer.RemoteAddr())
	return nil
}

func (fs *FileServer) Stop() {
	close(fs.quit)
	err := fs.Transport.Close()
	if err != nil {
		log.Printf("Error closing transport %s\n", err)
	}
	log.Printf("Quiting file server on : %s\n", fs.ListenAddr)
}

// OnPeer 连接建立成功的回调函数
func (fs *FileServer) OnPeer(peer p2p.Peer) error {
	fs.Lock()
	defer fs.Unlock()
	fs.peers[peer.RemoteAddr().String()] = peer
	log.Printf("[%s] connected to peer %s\n", fs.ListenAddr, peer.RemoteAddr())
	return nil
}

func init() {
	gob.Register(MessageGetFile{})
	gob.Register(MessageStoreFile{})
}
