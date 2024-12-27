package reflectx

import (
	"reflect"
)

func IsRefinedType[R any](value reflect.Type) bool {
	var toMatch R
	mt := reflect.TypeOf(toMatch)

	isRefinedType := mt == value
	return isRefinedType
}
