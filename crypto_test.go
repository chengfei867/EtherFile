package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestDefaultEncrypter_Encrypt_(t *testing.T) {
	e := DefaultEncrypter{}
	key := e.KeyGeneration()
	fmt.Printf("%x\n", key)
	src := bytes.NewReader([]byte("test aes encrypt!"))
	dst := new(bytes.Buffer)
	_, err := e.Encrypt(key, src, dst)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("the plaintext data :%+v \n", src)
	t.Logf("the ciphertext data : %s \n", dst.String())

	res := new(bytes.Buffer)
	totalSize, err := e.Decrypt(key, dst, res)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("the plaintext data :%s size : %d \n", res.String(), totalSize)
}
