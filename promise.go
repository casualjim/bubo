package bubo

import (
	"context"
	"sync"

	"github.com/casualjim/bubo/internal/executor"
)

type Future[T any] interface {
	// can't type alias this (yet) because of the type parameter
	Get() (T, error)
}

type deferredPromise[T any] struct {
	promise executor.CompletableFuture[T]
	hook    Hook[T]
	mu      sync.Mutex
	value   string
	err     error
	once    sync.Once
}

func (d *deferredPromise[T]) Forward(ctx context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.err != nil {
		d.promise.Error(d.err)
		return
	}

	d.promise.Complete(d.value)
	res, err := d.promise.Get()
	if err != nil {
		d.hook.OnError(ctx, err)
		return
	}
	d.hook.OnResult(ctx, res)
}

func (d *deferredPromise[T]) Complete(result string) {
	d.once.Do(func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.value = result
	})
}

func (d *deferredPromise[T]) Error(err error) {
	d.once.Do(func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.err = err
	})
}

type noopPromise struct{}

func (noopPromise) Complete(string) {}
func (noopPromise) Error(error)     {}
