package features

import (
	"bytes"
	"fmt"

	"github.com/CN-TU/go-flows/flows"
)

// BoolInt returns 1 if b is true, otherwise 0
func BoolInt(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

// Less returns a < b
func Less(a, b interface{}) bool {
	switch v := a.(type) {
	case float64:
		return v < b.(float64)
	case float32:
		return v < b.(float32)
	case int64:
		return v < b.(int64)
	case int32:
		return v < b.(int32)
	case int16:
		return v < b.(int16)
	case int8:
		return v < b.(int8)
	case int:
		return v < b.(int)
	case uint64:
		return v < b.(uint64)
	case uint32:
		return v < b.(uint32)
	case uint16:
		return v < b.(uint16)
	case uint8:
		return v < b.(uint8)
	case uint:
		return v < b.(uint)
	case flows.DateTimeSeconds:
		return v < b.(flows.DateTimeSeconds)
	case flows.DateTimeMilliseconds:
		return v < b.(flows.DateTimeMilliseconds)
	case flows.DateTimeMicroseconds:
		return v < b.(flows.DateTimeMicroseconds)
	case flows.DateTimeNanoseconds:
		return v < b.(flows.DateTimeNanoseconds)
	case bool:
		return !v && b.(bool)
	case string:
		return v < b.(string)
	case []byte:
		return bytes.Compare(v, b.([]byte)) < 0
	}
	panic(fmt.Sprintf("%v can't be used in comparison", a))
}
