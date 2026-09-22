// Package input adapts potentially blocking local input to invocation cancellation.
package input

import (
	"context"
	"io"
)

// Reader lets the caller return on cancellation even if Source is an arbitrary
// blocking Reader. The pending read owns its buffer; it cannot mutate the caller's
// buffer after return. At most one read is pending when an invocation stops. The
// owner closes its source, or process exit releases stdin, to finish that read.
type Reader struct {
	Context context.Context
	Source  io.Reader
}

func (r Reader) Read(p []byte) (int, error) {
	if err := r.Context.Err(); err != nil {
		return 0, err
	}
	if r.Context.Done() == nil {
		return r.Source.Read(p)
	}
	type result struct {
		n    int
		err  error
		data []byte
	}
	done := make(chan result, 1)
	go func() { b := make([]byte, len(p)); n, e := r.Source.Read(b); done <- result{n, e, b} }()
	select {
	case <-r.Context.Done():
		return 0, r.Context.Err()
	case v := <-done:
		copy(p, v.data[:v.n])
		return v.n, v.err
	}
}
