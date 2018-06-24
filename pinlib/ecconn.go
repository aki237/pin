package pinlib

import (
	"io"
	"net"

	"github.com/golang/snappy"
	"golang.org/x/crypto/salsa20"
)

type CompressorConn struct {
	net.Conn
	rd io.Reader
	wr io.Writer
}

func NewCompressorConn(conn net.Conn) *CompressorConn {
	cc := CompressorConn{}
	cc.rd = snappy.NewReader(conn)
	cc.wr = snappy.NewWriter(conn)
	cc.Conn = conn
	return &cc
}

func (cc *CompressorConn) Read(p []byte) (int, error) {
	return cc.rd.Read(p)
}

func (cc *CompressorConn) Write(p []byte) (int, error) {
	return cc.wr.Write(p)
}

type CryptoConn struct {
	key   [32]byte
	nonce [8]byte
	net.Conn
}

func NewCryptoConn(conn net.Conn, secret [40]byte) *CryptoConn {
	c := &CryptoConn{Conn: NewCompressorConn(conn)}
	copy(c.key[:], secret[:32])
	copy(c.nonce[:], secret[32:])

	return c
}

func (ac *CryptoConn) Write(b []byte) (int, error) {
	out := make([]byte, len(b))

	salsa20.XORKeyStream(out, b, ac.nonce[:], &ac.key)
	return ac.Conn.Write(out)
}

func (ac *CryptoConn) Read(b []byte) (int, error) {
	out := make([]byte, len(b))
	rd, err := ac.Conn.Read(out)

	salsa20.XORKeyStream(b, out, ac.nonce[:], &ac.key)
	return rd, err
}
