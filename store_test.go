package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_PathTransformFunc(t *testing.T) {
	path := SHA1PathTransformFunc("test_key")
	t.Log(path)
}

func Test_store(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: SHA1PathTransformFunc,
	}
	key := "test_file_path"
	s := NewStore(opts)
	data := bytes.NewReader([]byte("some bytes"))
	err := s.writeStream(key, data)
	if err != nil {
		t.Error(err)
	}
	read, err := s.Read(key)
	if err != nil {
		t.Error(err)
	}
	t.Logf("%s", read)
	err = s.Delete(key)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, s.Exists(key), false)
}
