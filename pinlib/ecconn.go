package pinlib

import (
	"crypto/cipher"
	"io"
	"net"

	"github.com/golang/snappy"
	"golang.org/x/crypto/chacha20poly1305"
)

// CompressorConn is used as a wrapper for net.Conn for compressing data while writing and
// decompressing while reading.
type CompressorConn struct {
	net.Conn
	rd io.Reader
	wr io.Writer
}

// NewCompressorConn is used to create a new CompressorConn struct
func NewCompressorConn(conn net.Conn) *CompressorConn {
	cc := CompressorConn{}
	cc.rd = snappy.NewReader(conn)
	cc.wr = snappy.NewWriter(conn)
	cc.Conn = conn
	return &cc
}

// Read method implements io.Reader interface for the CompressorConn
// Data read is decompressed.
func (cc *CompressorConn) Read(p []byte) (int, error) {
	return cc.rd.Read(p)
}

// Write method implements io.Writer interface for the CompressorConn
// Data written is compressed.
func (cc *CompressorConn) Write(p []byte) (int, error) {
	return cc.wr.Write(p)
}

// CryptoConn is a net.Conn wrapper that decrypts the read data
// and encrypts the written data. Here ChaCha20+Poly1305 cipher/authentication is used.
type CryptoConn struct {
	key      [32]byte
	nonce    [12]byte
	nonceGen *Rng
	crypter  cipher.AEAD
	net.Conn
}

// NewCryptoConn creates a new CryptoConn struct
func NewCryptoConn(conn net.Conn, secret [32]byte, initSeed int64) *CryptoConn {
	c := &CryptoConn{Conn: NewCompressorConn(conn)}
	copy(c.key[:], secret[:32])
	c.nonceGen = NewRng(initSeed)
	c.crypter, _ = chacha20poly1305.New(c.key[:])
	return c
}

// Read method implements io.Reader interface for the CryptoConn
// Data read is decrypted.
func (ac *CryptoConn) Read(b []byte) (int, error) {
	out := make([]byte, len(b)+16+12)
	rd, err := ac.Conn.Read(out)
	if err != nil {
		return rd, err
	}

	x, err := ac.crypter.Open(nil, out[:12], out[12:rd], nil)
	copy(b[:], x[:])
	return len(x), err
}

// Write method implements io.Writer interface for the CryptoConn
// Data written is encrypted.
func (ac *CryptoConn) Write(b []byte) (int, error) {
	ac.nonce = ac.nonceGen.RandomNonceGenerator()
	out := ac.crypter.Seal(ac.nonce[:], ac.nonce[:], b, nil)
	return ac.Conn.Write(out)
}
