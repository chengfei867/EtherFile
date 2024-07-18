package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

/**
密钥生成: KeyGeneration() []byte 生成随机的密钥
加密函数: Encrypt(key []byte,src io.Reader,dst io.Writer)
解密函数: Decrypt(key []byte,src io.Reader,dst io.Writer)
*/

const (
	DefaultKeyLength = 32
	DefaultIVSize    = 16
	BufferSize       = 32 * 1024
)

type Encrypter interface {
	Key() []byte
	KeyGeneration() []byte
	Encrypt([]byte, io.Reader, io.Writer) (int64, error)
	Decrypt([]byte, io.Reader, io.Writer) (int64, error)
}

// DefaultEncrypter 默认的加密类：使用AES加密算法
type DefaultEncrypter struct {
	key []byte
}

func NewDefaultEncrypter(key ...[]byte) *DefaultEncrypter {
	e := &DefaultEncrypter{}
	if len(key) > 0 && len(key[0]) == DefaultKeyLength {
		e.key = key[0]
	} else {
		e.key = e.KeyGeneration()
	}
	return e
}

func (e *DefaultEncrypter) Key() []byte {
	return e.key
}

// KeyGeneration 随机的密钥生成函数
func (e *DefaultEncrypter) KeyGeneration() []byte {
	key := make([]byte, DefaultKeyLength)
	_, err := io.ReadFull(rand.Reader, key)
	if err != nil {
		return nil
	}
	return key
}

func (e *DefaultEncrypter) Encrypt(key []byte, src io.Reader, dst io.Writer) (int64, error) {
	// 使用给定的密钥创建AES加密块
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	var (
		buf       = make([]byte, BufferSize)
		counter   = make([]byte, block.BlockSize()) // 用于CTR模式的计数器
		totalSize = int64(0)
	)

	// 初始化一个CTR加密流
	stream := cipher.NewCTR(block, counter)

	// 将计数器初始值写入到目标写入器
	if _, err := dst.Write(counter); err != nil {
		return 0, err
	}

	// 循环读取源数据并进行加密
	for {
		n, err := src.Read(buf) // 从源读取数据到缓冲区
		if n > 0 {
			// 使用CTR流加密缓冲区中的数据
			stream.XORKeyStream(buf, buf[:n])
			// 将加密后的数据写入目标写入器
			n, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			totalSize += int64(n)
		}
		// 处理读取错误
		if err != nil {
			if err == io.EOF {
				break // 读取结束
			}
			return 0, err
		}
	}
	return totalSize, nil
}

func (e *DefaultEncrypter) Decrypt(key []byte, src io.Reader, dst io.Writer) (int64, error) {
	// 创建 AES 密码块
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	// 从源读取计数器
	counter := make([]byte, block.BlockSize())
	if _, err := src.Read(counter); err != nil {
		return 0, err
	}

	// 初始化一个缓冲区和加密流
	var (
		buf       = make([]byte, 32*1024)         // 创建一个大小为 32KB 的缓冲区
		stream    = cipher.NewCTR(block, counter) // 创建一个新的 CTR 加密流
		totalSize = int64(0)
	)

	// 逐块读取、解密和写入数据
	for {
		n, err := src.Read(buf) // 从源读取数据到缓冲区
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n]) // 使用密钥流解密数据
			_, err := dst.Write(buf[:n])      // 将解密后的数据写入目标
			if err != nil {
				return 0, err
			}
			totalSize += int64(n)
		}
		if err != nil {
			if err == io.EOF {
				break // 读取完所有数据，退出循环
			}
			return 0, err
		}
	}
	return totalSize, nil
}
