package main

import (
	"bytes"
	"errors"
	"fmt"
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

func (s *Store) writeStream(key string, r io.Reader) error {
	// 路径名转换
	pathKey := s.PathTransformFunc(key)

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

func (s *Store) readStream(key string) (io.ReadCloser, error) {
	keyPath := s.PathTransformFunc(key)
	return os.Open(s.Root + "/" + keyPath.FullPath())
}

func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.readStream(key)
	if err != nil {
		return nil, err
	}
	defer func(f io.ReadCloser) {
		err = f.Close()
		if err != nil {
			fmt.Println("Error closing file:", err)
		}
	}(f)
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)
	return buf, nil
}

func (s *Store) Delete(key string) error {
	keyPath := s.PathTransformFunc(key)
	defer func() {
		log.Println("Deleted file:", keyPath.FileName)
	}()
	root := keyPath.RootPath()
	return os.RemoveAll(s.Root + "/" + root)
}

func (s *Store) Exists(key string) bool {
	keyPath := s.PathTransformFunc(key)
	_, err := os.Stat(s.Root + "/" + keyPath.FullPath())
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}
