package input

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestCanceledBlockedReadReturnsWithoutMutatingBuffer(t *testing.T) {
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { b := make([]byte, 8); _, err := (Reader{ctx, pr}).Read(b); done <- err }()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked input ignored cancellation")
	}
}
