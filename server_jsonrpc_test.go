package rpc

import (
	"io"
)

type HTTPReadWriteCloser struct {
	In  io.Reader
	Out io.Writer
}

func (c *HTTPReadWriteCloser) Read(p []byte) (n int, err error)  { return c.In.Read(p) }
func (c *HTTPReadWriteCloser) Write(d []byte) (n int, err error) { return c.Out.Write(d) }
func (c *HTTPReadWriteCloser) Close() error                      { return nil }
