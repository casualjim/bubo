package registry

import "github.com/alphadose/haxmap"

type Registry[T any] interface {
	Get(name string) (T, bool)
	Add(name string, value T)
	GetOrAdd(name string, value func() T) (T, bool)
	Del(name string)
}

type registry[T any] struct {
	values *haxmap.Map[string, T]
}

func New[T any]() Registry[T] {
	return &registry[T]{
		values: haxmap.New[string, T](),
	}
}

func (r *registry[T]) Get(name string) (T, bool) {
	return r.values.Get(name)
}

func (r *registry[T]) Add(name string, value T) {
	r.values.Set(name, value)
}

func (r *registry[T]) GetOrAdd(name string, valueFn func() T) (T, bool) {
	return r.values.GetOrCompute(name, valueFn)
}

func (r *registry[T]) Del(name string) {
	r.values.Del(name)
}
