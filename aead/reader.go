package aead

import (
	"bytes"
	"crypto/cipher"
	"encoding/binary"
	"io"

	"github.com/isayme/go-bufferpool"
	"github.com/isayme/go-toh2/util"
	"github.com/pkg/errors"
)

type AeadReader struct {
	password string
	keySize  int

	reader     io.Reader
	newCipher  func([]byte) (cipher.AEAD, error)
	aead       cipher.AEAD
	nonce      []byte
	readBuffer *bytes.Buffer
}

func NewAeadReader(reader io.Reader, password string, keySize int, newCipher func([]byte) (cipher.AEAD, error)) *AeadReader {
	return &AeadReader{
		password:   password,
		keySize:    keySize,
		reader:     reader,
		readBuffer: bytes.NewBuffer(nil),
		newCipher:  newCipher,
	}
}

func (r *AeadReader) getAeadCipher() (cipher.AEAD, error) {
	if r.aead != nil {
		return r.aead, nil
	}

	salt := bufferpool.Get(r.keySize)
	defer bufferpool.Put(salt)

	if _, err := io.ReadFull(r.reader, salt); err != nil {
		return nil, errors.Wrap(err, "read aead salt")
	}

	key := util.KDF(r.password, salt, r.keySize)
	c, err := r.newCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, "new cipher")
	}

	r.aead = c
	r.nonce = make([]byte, c.NonceSize())

	return r.aead, nil
}

func (r *AeadReader) doRead() error {
	c, err := r.getAeadCipher()
	if err != nil {
		return err
	}

	// read size
	sizeBuf := bufferpool.Get(2 + c.Overhead())
	defer bufferpool.Put(sizeBuf)
	_, err = io.ReadFull(r.reader, sizeBuf)
	if err != nil {
		return errors.Wrap(err, "aead read size")
	}

	ret, err := c.Open(sizeBuf[:0], r.nonce, sizeBuf, nil)
	if err != nil {
		return errors.Wrap(err, "aead decrypt size")
	}
	util.NextNonce(r.nonce)

	payloadSize := int(binary.BigEndian.Uint16(ret) & 0x3FFF)

	// read payload
	payloadBuf := bufferpool.Get(payloadSize + c.Overhead())
	defer bufferpool.Put(payloadBuf)

	_, err = io.ReadFull(r.reader, payloadBuf)
	if err != nil {
		return errors.Wrap(err, "aead read payload")
	}

	ret, err = c.Open(payloadBuf[:0], r.nonce, payloadBuf, nil)
	if err != nil {
		return errors.Wrap(err, "aead decrypt payload")
	}
	util.NextNonce(r.nonce)

	r.readBuffer.Write(ret)

	return nil
}

func (r *AeadReader) Read(p []byte) (n int, err error) {
	if r.readBuffer.Len() > 0 {
		return r.readBuffer.Read(p)
	}

	if err := r.doRead(); err != nil {
		return 0, err
	}

	return r.readBuffer.Read(p)
}