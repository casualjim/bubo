package api

type RunResult[T any] struct {
	Success T
	Err     error
}

func (r RunResult[T]) IsSuccess() bool {
	return r.Err == nil
}

func (r RunResult[T]) IsError() bool {
	return r.Err != nil
}
