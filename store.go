package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"os"
)

const (
	DefaultRootName = "etherPath"
)

type StoreOpts struct {
	Root              string
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = DefaultRootName
	}
	return &Store{opts}
}

func (s *Store) writeEncrypt(key string, encrypter Encrypter, src io.Reader) error {
	// 加密当前文件 并且加入缓冲区
	encryptedBuffer := new(bytes.Buffer)
	if _, err := encrypter.Encrypt(encrypter.Key(), src, encryptedBuffer); err != nil {
		return err
	}

	// 将加密后的数据写入存储
	if err := s.writeStream(key, encryptedBuffer); err != nil {
		return err
	}
	return nil
}

func (s *Store) writeStream(key string, r io.Reader) error {
	// 路径名转换
	pathKey := s.PathTransformFunc(key)

	//log.Printf("row key : %s,  pathKey : %s ", key, pathKey)
	if err := os.MkdirAll(s.Root+"/"+pathKey.PathName, os.ModePerm); err != nil {
		return err
	}

	// 拼接路径和文件名
	fullPathWithRoot := s.Root + "/" + pathKey.FullPath()
	f, err := os.OpenFile(fullPathWithRoot, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	// 将内容写入文件
	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}
	log.Printf("wrote %d bytes to %s", n, fullPathWithRoot)
	return nil
}

func (s *Store) WriteEncrypt(key string, encrypter Encrypter, src io.Reader) error {
	return s.writeEncrypt(key, encrypter, src)
}

func (s *Store) Write(key string, r io.Reader) error {
	return s.writeStream(key, r)
}

func (s *Store) readDecrypt(key string, encrypter Encrypter, dst io.Writer) error {
	keyPath := s.PathTransformFunc(key)
	f, err := os.Open(s.Root + "/" + keyPath.FullPath())
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = encrypter.Decrypt(encrypter.Key(), f, dst)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) readStream(key string) (int64, io.ReadCloser, error) {
	keyPath := s.PathTransformFunc(key)
	f, err := os.Open(s.Root + "/" + keyPath.FullPath())
	if err != nil {
		return 0, nil, err
	}
	info, err := f.Stat()
	if err != nil {
		return 0, nil, err
	}
	return info.Size(), f, nil
}

func (s *Store) ReadDecrypt(key string, encrypter Encrypter, dst io.Writer) error {
	return s.readDecrypt(key, encrypter, dst)
}

func (s *Store) Read(key string) (int64, io.ReadCloser, error) {
	return s.readStream(key)
}

func (s *Store) Delete(key string) error {
	keyPath := s.PathTransformFunc(key)
	defer func() {
		log.Println("Deleted file:", keyPath.FileName)
	}()
	root := keyPath.RootPath()
	return os.RemoveAll(s.Root + "/" + root)
}

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) Exists(key string) bool {
	keyPath := s.PathTransformFunc(key)
	//log.Printf("Get full path: %s", s.Root+"/"+keyPath.FullPath())
	_, err := os.Stat(s.Root + "/" + keyPath.FullPath())
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}
