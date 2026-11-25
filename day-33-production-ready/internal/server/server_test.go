package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// freeAddr finds a port nobody is using right now
func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	return addr
}

func waitUntilUp(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond); err == nil {
			conn.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("the server never came up on %s", addr)
}

func TestRunFinishesRunningRequests(t *testing.T) {
	addr := freeAddr(t)
	started := make(chan struct{})

	srv := &http.Server{Addr: addr, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		time.Sleep(300 * time.Millisecond) // a slow request
		fmt.Fprint(w, "finished")
	})}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, srv, 5*time.Second) }()
	waitUntilUp(t, addr)

	type result struct {
		body string
		err  error
	}
	got := make(chan result, 1)
	go func() {
		resp, err := http.Get("http://" + addr)
		if err != nil {
			got <- result{err: err}
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		got <- result{body: string(body)}
	}()

	<-started
	cancel() // ask the server to stop while that request is still running

	if r := <-got; r.err != nil || r.body != "finished" {
		t.Fatalf("the running request was cut off: body %q, error %v", r.body, r.err)
	}
	if err := <-done; err != nil {
		t.Errorf("Run returned %v, want nil after a clean shutdown", err)
	}

	// And it really stopped listening
	if conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond); err == nil {
		conn.Close()
		t.Error("the server is still accepting connections after shutting down")
	}
}

func TestRunGivesUpAfterTheGracePeriod(t *testing.T) {
	addr := freeAddr(t)
	started := make(chan struct{})

	srv := &http.Server{Addr: addr, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		time.Sleep(3 * time.Second) // far longer than the grace period below
	})}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, srv, 150*time.Millisecond) }()
	waitUntilUp(t, addr)

	go func() {
		resp, err := http.Get("http://" + addr)
		if err == nil {
			resp.Body.Close()
		}
	}()

	<-started
	begin := time.Now()
	cancel()

	err := <-done
	if err == nil {
		t.Error("expected an error when the request outlasts the grace period")
	}
	if took := time.Since(begin); took > 2*time.Second {
		t.Errorf("Run waited %v, it should have given up after about 150ms", took)
	}
}

func TestRunReturnsStartupErrors(t *testing.T) {
	// Take a port, then try to start a second server on it
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	srv := &http.Server{Addr: ln.Addr().String(), Handler: http.NotFoundHandler()}

	err = Run(context.Background(), srv, time.Second)
	if err == nil {
		t.Fatal("expected an error because the port is already taken")
	}
}
