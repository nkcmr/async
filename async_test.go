package async

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestNewPromise(t *testing.T) {
	promise := NewPromise(func() (string, error) {
		time.Sleep(time.Millisecond * 500)
		return "foo", nil
	})
	ctx := context.Background()
	v, err := promise.Await(ctx)
	requireEqual(t, true, promise.Settled())
	requireNoError(t, err)
	requireEqual(t, "foo", v)

	promise = NewPromise(func() (string, error) {
		time.Sleep(time.Millisecond)
		return "", errors.New("darn")
	})
	_, err = promise.Await(ctx)
	requireEqual(t, true, promise.Settled())
	requireError(t, err)
	requireEqual(t, "darn", err.Error())

	ctxlowtimeout, cancel := context.WithTimeout(ctx, time.Millisecond*50)
	defer cancel()
	promise = NewPromise(func() (string, error) {
		time.Sleep(time.Millisecond * 100) // should blow past the timeout
		return "too slow", nil
	})
	_, err = promise.Await(ctxlowtimeout)
	requireEqual(t, false, promise.Settled()) // since it is probably still pending once we've timed out
	requireError(t, err)
	requireEqual(t, context.DeadlineExceeded, err)
}

func TestAll(t *testing.T) {
	promises := []Promise[int]{
		NewPromise(func() (int, error) {
			return 42, nil
		}),
		NewPromise(func() (int, error) {
			time.Sleep(time.Millisecond * 50)
			return 43, nil
		}),
		NewPromise(func() (int, error) {
			time.Sleep(time.Millisecond * 100)
			return 44, nil
		}),
	}
	ctx := context.Background()
	ints, err := All(ctx, promises)
	requireNoError(t, err)
	requireEqual(t, []int{42, 43, 44}, ints)

	promises = []Promise[int]{
		NewPromise(func() (int, error) {
			return 42, nil
		}),
		NewPromise(func() (int, error) {
			time.Sleep(time.Millisecond * 50)
			return 0, errors.New("doh!")
		}),
	}
	ints, err = All(ctx, promises)
	requireError(t, err)
	requireEqual(t, ints, nil)
}

func TestResolve(t *testing.T) {
	promise := Resolve("dff73ab5-5ff6-44f6-ba1e-7447ebf38675")
	if !promise.Settled() {
		t.Fatal("expected promise to be immediately resolved")
		return
	}
	ctx := context.Background()
	v, err := promise.Await(ctx)
	requireNoError(t, err)
	requireEqual(t, "dff73ab5-5ff6-44f6-ba1e-7447ebf38675", v)
}

func TestReject(t *testing.T) {
	promise := Reject[struct{}](errors.New("it failed. what can i say?"))
	if !promise.Settled() {
		t.Fatal("expected promise to be immediately rejected")
		return
	}
	ctx := context.Background()
	_, err := promise.Await(ctx)
	requireError(t, err)
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
}

func requireError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func requireEqual[T any](t *testing.T, expected, actual T) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf(`expected "%v" got "%v"`, expected, actual)
	}
}
