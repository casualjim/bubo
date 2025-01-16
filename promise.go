// Package bubo provides a framework for building conversational AI agents that can interact
// in a structured manner. It supports multi-agent conversations, structured output,
// and flexible execution contexts.
package bubo

import (
	"context"
	"sync"

	"github.com/casualjim/bubo/internal/executor"
)

// Future represents a value of type T that will be available in the future.
// It provides a way to retrieve the value once it's ready.
// Note: This interface cannot be type aliased yet due to the type parameter T.
type Future[T any] interface {
	// Get retrieves the value once it's available.
	// Returns the value of type T and any error that occurred during computation.
	Get() (T, error)
}

// deferredPromise implements a promise pattern for handling asynchronous results
// of type T. It coordinates between the executor's CompletableFuture and the
// conversation hook, ensuring thread-safe access to results and proper error handling.
type deferredPromise[T any] struct {
	promise executor.CompletableFuture[T] // The underlying future that will hold the final result
	hook    Hook[T]                       // Hook for handling results and errors
	mu      sync.Mutex                    // Mutex for thread-safe access to value and error
	value   string                        // The raw result value
	err     error                         // Any error that occurred during execution
	once    sync.Once                     // Ensures one-time completion/error setting
}

// Forward processes the promise's result or error, propagating it to both the
// CompletableFuture and the hook. This method ensures proper synchronization
// and handles both successful results and errors appropriately.
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

// Complete marks the promise as successfully completed with the given result.
// This method is thread-safe and ensures the result is set only once.
func (d *deferredPromise[T]) Complete(result string) {
	d.once.Do(func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.value = result
	})
}

// Error marks the promise as failed with the given error.
// This method is thread-safe and ensures the error is set only once.
func (d *deferredPromise[T]) Error(err error) {
	d.once.Do(func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.err = err
	})
}

// noopPromise implements a no-operation promise that discards all results and errors.
// It's used for intermediate steps in a conversation where the results don't need
// to be captured or processed.
type noopPromise struct{}

func (noopPromise) Complete(string) {}
func (noopPromise) Error(error)     {}
