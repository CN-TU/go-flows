package features

import (
	"bytes"
	"fmt"

	"github.com/CN-TU/go-flows/flows"
)

// TypedSlice is a slice that can hold uint64, int64, bool, string, or []slice
type TypedSlice interface {
	Append(interface{})
	Get(i int) interface{}
	GetFloat(i int) float64
	Len() int
	Less(i, j int) bool
	Equal(i, j int) bool
	Swap(i, j int)
	IsNumeric() bool
}

type uint64Slice []uint64

func (s *uint64Slice) Append(val interface{}) {
	*s = append(*s, flows.ToUInt(val))
}

func (s *uint64Slice) Len() int {
	return len(*s)
}

func (s *uint64Slice) Less(i, j int) bool {
	return (*s)[i] < (*s)[j]
}

func (s *uint64Slice) Equal(i, j int) bool {
	return (*s)[i] == (*s)[j]
}

func (s *uint64Slice) Swap(i, j int) {
	(*s)[i], (*s)[j] = (*s)[j], (*s)[i]
}

func (s *uint64Slice) Get(i int) interface{} {
	return (*s)[i]
}

func (s *uint64Slice) GetFloat(i int) float64 {
	return float64((*s)[i])
}

func (s *uint64Slice) IsNumeric() bool {
	return true
}

type int64Slice []int64

func (s *int64Slice) Append(val interface{}) {
	*s = append(*s, flows.ToInt(val))
}

func (s *int64Slice) Len() int {
	return len(*s)
}

func (s *int64Slice) Less(i, j int) bool {
	return (*s)[i] < (*s)[j]
}

func (s *int64Slice) Equal(i, j int) bool {
	return (*s)[i] == (*s)[j]
}

func (s *int64Slice) Swap(i, j int) {
	(*s)[i], (*s)[j] = (*s)[j], (*s)[i]
}

func (s *int64Slice) Get(i int) interface{} {
	return (*s)[i]
}

func (s *int64Slice) GetFloat(i int) float64 {
	return float64((*s)[i])
}

func (s *int64Slice) IsNumeric() bool {
	return true
}

type float64Slice []float64

func (s *float64Slice) Append(val interface{}) {
	*s = append(*s, flows.ToFloat(val))
}

func (s *float64Slice) Len() int {
	return len(*s)
}

func (s *float64Slice) Less(i, j int) bool {
	return (*s)[i] < (*s)[j]
}

func (s *float64Slice) Equal(i, j int) bool {
	return (*s)[i] == (*s)[j]
}

func (s *float64Slice) Swap(i, j int) {
	(*s)[i], (*s)[j] = (*s)[j], (*s)[i]
}

func (s *float64Slice) Get(i int) interface{} {
	return (*s)[i]
}

func (s *float64Slice) GetFloat(i int) float64 {
	return (*s)[i]
}

func (s *float64Slice) IsNumeric() bool {
	return true
}

type boolSlice []bool

func (s *boolSlice) Append(val interface{}) {
	*s = append(*s, val.(bool))
}

func (s *boolSlice) Len() int {
	return len(*s)
}

func (s *boolSlice) Less(i, j int) bool {
	return !(*s)[i] && (*s)[j]
}

func (s *boolSlice) Equal(i, j int) bool {
	return (*s)[i] == (*s)[j]
}

func (s *boolSlice) Swap(i, j int) {
	(*s)[i], (*s)[j] = (*s)[j], (*s)[i]
}

func (s *boolSlice) Get(i int) interface{} {
	return (*s)[i]
}

func (s *boolSlice) GetFloat(i int) float64 {
	if (*s)[i] {
		return 1
	}
	return 0
}

func (s *boolSlice) IsNumeric() bool {
	return true
}

type stringSlice []string

func (s *stringSlice) Append(val interface{}) {
	*s = append(*s, val.(string))
}

func (s *stringSlice) Len() int {
	return len(*s)
}

func (s *stringSlice) Less(i, j int) bool {
	return (*s)[i] < (*s)[j]
}

func (s *stringSlice) Equal(i, j int) bool {
	return (*s)[i] == (*s)[j]
}

func (s *stringSlice) Swap(i, j int) {
	(*s)[i], (*s)[j] = (*s)[j], (*s)[i]
}

func (s *stringSlice) Get(i int) interface{} {
	return (*s)[i]
}

func (s *stringSlice) GetFloat(i int) float64 {
	panic("Can't convert to float")
}

func (s *stringSlice) IsNumeric() bool {
	return false
}

type bytesSlice [][]byte

func (s *bytesSlice) Append(val interface{}) {
	*s = append(*s, val.([]byte))
}

func (s *bytesSlice) Len() int {
	return len(*s)
}

func (s *bytesSlice) Less(i, j int) bool {
	return bytes.Compare((*s)[i], (*s)[j]) == -1
}

func (s *bytesSlice) Equal(i, j int) bool {
	return bytes.Equal((*s)[i], (*s)[j])
}

func (s *bytesSlice) Swap(i, j int) {
	(*s)[i], (*s)[j] = (*s)[j], (*s)[i]
}

func (s *bytesSlice) Get(i int) interface{} {
	return (*s)[i]
}

func (s *bytesSlice) GetFloat(i int) float64 {
	panic("Can't convert to float")
}

func (s *bytesSlice) IsNumeric() bool {
	return false
}

// NewTypedSlice returns a new TypedSlice with a backing implementation depending on val type.
// The returned slice already contains val
func NewTypedSlice(val interface{}) TypedSlice {
	switch v := val.(type) {
	case float64:
		return &float64Slice{v}
	case float32:
		return &float64Slice{float64(v)}
	case int64:
		return &int64Slice{v}
	case int32:
		return &int64Slice{int64(v)}
	case int16:
		return &int64Slice{int64(v)}
	case int8:
		return &int64Slice{int64(v)}
	case int:
		return &int64Slice{int64(v)}
	case uint64:
		return &uint64Slice{v}
	case uint32:
		return &uint64Slice{uint64(v)}
	case uint16:
		return &uint64Slice{uint64(v)}
	case uint8:
		return &uint64Slice{uint64(v)}
	case uint:
		return &uint64Slice{uint64(v)}
	case flows.DateTimeSeconds:
		return &uint64Slice{uint64(v)}
	case flows.DateTimeMilliseconds:
		return &uint64Slice{uint64(v)}
	case flows.DateTimeMicroseconds:
		return &uint64Slice{uint64(v)}
	case flows.DateTimeNanoseconds:
		return &uint64Slice{uint64(v)}
	case bool:
		return &boolSlice{v}
	case string:
		return &stringSlice{v}
	case []byte:
		return &bytesSlice{v}
	}
	panic(fmt.Sprintf("%v can't be used in a TypedSlice", val))
}
