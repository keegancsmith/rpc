// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
)

type shutdownCodec struct {
	responded chan int
	closed    bool
}

func (c *shutdownCodec) WriteRequest(*Request, any) error { return nil }
func (c *shutdownCodec) ReadResponseBody(any) error       { return nil }
func (c *shutdownCodec) ReadResponseHeader(*Response) error {
	c.responded <- 1
	return errors.New("shutdownCodec ReadResponseHeader")
}
func (c *shutdownCodec) Close() error {
	c.closed = true
	return nil
}

func TestCloseCodec(t *testing.T) {
	codec := &shutdownCodec{responded: make(chan int)}
	client := NewClientWithCodec(codec)
	<-codec.responded
	client.Close()
	if !codec.closed {
		t.Error("client.Close did not close codec")
	}
}

// Test that errors in gob shut down the connection. Issue 7689.

type R struct {
	msg []byte // Not exported, so R does not work with gob.
}

type S struct{}

func (s *S) Recv(ctx context.Context, nul *struct{}, reply *R) error {
	*reply = R{[]byte("foo")}
	return nil
}

func TestGobError(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Fatal("no error")
		}
		if !strings.Contains(err.(error).Error(), "reading body unexpected EOF") {
			t.Fatal("expected `reading body unexpected EOF', got", err)
		}
	}()
	Register(new(S))

	listen, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	go Accept(listen)

	client, err := Dial("tcp", listen.Addr().String())
	if err != nil {
		panic(err)
	}

	var reply Reply
	err = client.Call(context.Background(), "S.Recv", &struct{}{}, &reply)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%#v\n", reply)
	client.Close()

	listen.Close()
}

type ClientCodecError struct {
	WriteRequestError error
}

func (c *ClientCodecError) WriteRequest(*Request, any) error {
	return c.WriteRequestError
}
func (c *ClientCodecError) ReadResponseHeader(*Response) error {
	return nil
}
func (c *ClientCodecError) ReadResponseBody(any) error {
	return nil
}
func (c *ClientCodecError) Close() error {
	return nil
}

func TestClientTrace(t *testing.T) {
	wantErr := errors.New("test")
	client := NewClientWithCodec(&ClientCodecError{WriteRequestError: wantErr})
	defer client.Close()

	startCalled := false
	var gotErr error
	ctx := WithClientTrace(context.Background(), &ClientTrace{
		WriteRequestStart: func() { startCalled = true },
		WriteRequestDone:  func(err error) { gotErr = err },
	})

	var reply Reply
	err := client.Call(ctx, "S.Recv", &struct{}{}, &reply)
	if err != wantErr {
		t.Fatalf("expected Call to return the same error sent to ClientTrace.WriteRequestDone: want %v, got %v", wantErr, err)
	}
	if gotErr != wantErr {
		t.Fatalf("expected ClientTrace.WriteRequestDone to be called with error %v, got %v", wantErr, gotErr)
	}
	if !startCalled {
		t.Fatal("expected ClientTrace.WriteRequestStart to be called")
	}
}
