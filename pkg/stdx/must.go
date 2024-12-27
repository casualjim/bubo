package stdx

func Must0(err error) {
	if err != nil {
		panic(err)
	}
}

func Must1[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func Must2[T any, V any](t T, v V, err error) (T, V) {
	if err != nil {
		panic(err)
	}
	return t, v
}

func Must3[T any, V any, U any](t T, v V, u U, err error) (T, V, U) {
	if err != nil {
		panic(err)
	}
	return t, v, u
}

func Must4[T any, V any, U any, W any](t T, v V, u U, w W, err error) (T, V, U, W) {
	if err != nil {
		panic(err)
	}
	return t, v, u, w
}

func Must5[T any, V any, U any, W any, X any](t T, v V, u U, w W, x X, err error) (T, V, U, W, X) {
	if err != nil {
		panic(err)
	}
	return t, v, u, w, x
}

func Must6[T, V, U, W, X, Y any](t T, v V, u U, w W, x X, y Y, err error) (T, V, U, W, X, Y) {
	if err != nil {
		panic(err)
	}
	return t, v, u, w, x, y
}

func Must7[T, V, U, W, X, Y, Z any](t T, v V, u U, w W, x X, y Y, z Z, err error) (T, V, U, W, X, Y, Z) {
	if err != nil {
		panic(err)
	}
	return t, v, u, w, x, y, z
}

func Must8[T, V, U, W, X, Y, Z, A any](t T, v V, u U, w W, x X, y Y, z Z, a A, err error) (T, V, U, W, X, Y, Z, A) {
	if err != nil {
		panic(err)
	}
	return t, v, u, w, x, y, z, a
}
