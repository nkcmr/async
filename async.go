package async // import "code.nkcmr.net/async"

import (
	"context"
	"sync/atomic"
)

// Promise is an abstract representation of a value that might eventually be
// delivered.
type Promise[T any] interface {
	// Settled indicates if a call to Await will cause a blocking behavior, or
	// if the result will be immediately returned.
	Settled() bool

	// Await will cause the calling code to block and wait for the promise to
	// settle. Await MUST be able to be called by multiple goroutines and safely
	// deliver the same value/error to all waiting goroutines. Successive calls
	// to Await should continue to respond with the result even once the promise
	// is settled.
	Await(context.Context) (T, error)
}

type result[T any] struct {
	value T
	err   error
}

type syncPromise[T any] struct {
	settled int32 // atomic
	done    chan struct{}
	result  result[T]
}

func (s *syncPromise[T]) Await(ctx context.Context) (T, error) {
	select {
	case <-ctx.Done():
		var zerov T
		return zerov, ctx.Err()
	case <-s.done:
		return s.result.value, s.result.err
	}
}

func (s *syncPromise[T]) Settled() bool {
	return atomic.LoadInt32(&s.settled) == 1
}

func (s *syncPromise[T]) resolve(v T) {
	if atomic.CompareAndSwapInt32(&s.settled, 0, 1) {
		s.result = result[T]{value: v}
		close(s.done)
	}
}

func (s *syncPromise[T]) reject(err error) {
	if atomic.CompareAndSwapInt32(&s.settled, 0, 1) {
		s.result = result[T]{err: err}
		close(s.done)
	}
}

// NewPromise wraps a function in a goroutine that will make the result of that
// function deliver its result to the holder of the promise.
func NewPromise[T any](fn func() (T, error)) Promise[T] {
	c := &syncPromise[T]{
		done: make(chan struct{}),
	}
	go func() {
		v, err := fn()
		if err != nil {
			c.reject(err)
		} else {
			c.resolve(v)
		}
	}()
	return c
}

type rp[T any] struct {
	result result[T]
}

func (r *rp[T]) Settled() bool { return true }

func (r *rp[T]) Await(context.Context) (T, error) {
	return r.result.value, r.result.err
}

// Resolve wraps a value in a promise that will always be immediately settled
// and return the provided value.
func Resolve[T any](v T) Promise[T] {
	return &rp[T]{result: result[T]{value: v}}
}

// Reject wraps an error in a promise that will always be immediately settled
// and return an error.
func Reject[T any](err error) Promise[T] {
	return &rp[T]{result: result[T]{err: err}}
}

// All takes a slice of promises and will await the result of all of the
// specified promises. If any promise should return an error, the wh
func All[T any](ctx context.Context, promises []Promise[T]) ([]T, error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()
	out := make([]T, len(promises))
	errc := make(chan error, len(out))
	waiter := func(i int, p Promise[T]) {
		var err error
		out[i], err = p.Await(ctx)
		errc <- err
	}
	for i := range out {
		go waiter(i, promises[i])
	}
	for i := 0; i < len(out); i++ {
		if err := <-errc; err != nil {
			cancel()
			return nil, err
		}
	}
	return out, nil
}
