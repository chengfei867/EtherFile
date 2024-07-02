package main

import (
	"crypto/sha1"
	"fmt"
	"path/filepath"
	"strings"
)

type PathKey struct {
	PathName string
	FileName string
}

func (p PathKey) RootPath() string {
	return strings.Split(p.PathName, "/")[0]
}

func (p PathKey) FullPath() string {
	return filepath.Join(p.PathName, p.FileName)
}

type PathTransformFunc func(string) PathKey

func DefaultPathTransformFunc(key string) PathKey {
	return PathKey{
		FileName: key,
		PathName: key,
	}
}

func SHA1PathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := fmt.Sprintf("%x", hash)
	blocksize := 5
	pathLen := len(hashStr) / blocksize
	paths := make([]string, pathLen)
	for i := range pathLen {
		paths[i] = hashStr[i*blocksize : (i+1)*blocksize]
	}
	return PathKey{
		PathName: filepath.Join(paths...),
		FileName: hashStr,
	}
}
