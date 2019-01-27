package features

import (
	"bytes"
	"fmt"
	"math"

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
	Select(left, right, k int)
	Max(left, right int) int
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

func sign(x float64) float64 {
	if x == 0 {
		return 0
	}
	if x < 0 {
		return -1
	}
	return 1
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// Select implements Floyd-Rivest-Selection-Algorithm for finding the k-lowest number
// 600 and 0.5 are constants from the original paper
func (s *uint64Slice) Select(left, right, k int) {
	array := *s
	for right > left {
		if right-left > 600 {
			n := float64(right - left + 1)
			i := float64(k - left + 1)
			z := math.Log(n)
			S := 0.5 * math.Exp(2*z/3)
			sd := 0.5 * math.Sqrt(z*S*(n-S)/n) * sign(i-n/2)
			newleft := max(left, int(float64(k)-i*S/n+sd))
			newright := min(right, int(float64(k)+(n-i)*S/n+sd))
			s.Select(newleft, newright, k)
		}
		t := array[k]
		i := left
		j := right
		array[left], array[k] = array[k], array[left]
		if array[right] > t {
			array[right], array[left] = array[left], array[right]
		}
		for i < j {
			array[i], array[j] = array[j], array[i]
			i++
			j--
			for array[i] < t {
				i++
			}
			for array[j] > t {
				j--
			}
		}
		if array[left] == t {
			array[left], array[j] = array[j], array[left]
		} else {
			j++
			array[j], array[right] = array[right], array[j]
		}
		if j <= k {
			left = j + 1
		}
		if k <= j {
			right = j - 1
		}
	}
}

func (s *uint64Slice) Max(left, right int) int {
	array := *s
	m := array[left]
	id := left
	for i := left + 1; i <= right; i++ {
		if array[i] > m {
			m = array[i]
			id = i
		}
	}
	return id
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

// Select implements Floyd-Rivest-Selection-Algorithm for finding the k-lowest number
// 600 and 0.5 are constants from the original paper
func (s *int64Slice) Select(left, right, k int) {
	array := *s
	for right > left {
		if right-left > 600 {
			n := float64(right - left + 1)
			i := float64(k - left + 1)
			z := math.Log(n)
			S := 0.5 * math.Exp(2*z/3)
			sd := 0.5 * math.Sqrt(z*S*(n-S)/n) * sign(i-n/2)
			newleft := max(left, int(float64(k)-i*S/n+sd))
			newright := min(right, int(float64(k)+(n-i)*S/n+sd))
			s.Select(newleft, newright, k)
		}
		t := array[k]
		i := left
		j := right
		array[left], array[k] = array[k], array[left]
		if array[right] > t {
			array[right], array[left] = array[left], array[right]
		}
		for i < j {
			array[i], array[j] = array[j], array[i]
			i++
			j--
			for array[i] < t {
				i++
			}
			for array[j] > t {
				j--
			}
		}
		if array[left] == t {
			array[left], array[j] = array[j], array[left]
		} else {
			j++
			array[j], array[right] = array[right], array[j]
		}
		if j <= k {
			left = j + 1
		}
		if k <= j {
			right = j - 1
		}
	}
}

func (s *int64Slice) Max(left, right int) int {
	array := *s
	m := array[left]
	id := left
	for i := left + 1; i <= right; i++ {
		if array[i] > m {
			m = array[i]
			id = i
		}
	}
	return id
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

// Select implements Floyd-Rivest-Selection-Algorithm for finding the k-lowest number
// 600 and 0.5 are constants from the original paper
func (s *float64Slice) Select(left, right, k int) {
	array := *s
	for right > left {
		if right-left > 600 {
			n := float64(right - left + 1)
			i := float64(k - left + 1)
			z := math.Log(n)
			S := 0.5 * math.Exp(2*z/3)
			sd := 0.5 * math.Sqrt(z*S*(n-S)/n) * sign(i-n/2)
			newleft := max(left, int(float64(k)-i*S/n+sd))
			newright := min(right, int(float64(k)+(n-i)*S/n+sd))
			s.Select(newleft, newright, k)
		}
		t := array[k]
		i := left
		j := right
		array[left], array[k] = array[k], array[left]
		if array[right] > t {
			array[right], array[left] = array[left], array[right]
		}
		for i < j {
			array[i], array[j] = array[j], array[i]
			i++
			j--
			for array[i] < t {
				i++
			}
			for array[j] > t {
				j--
			}
		}
		if array[left] == t {
			array[left], array[j] = array[j], array[left]
		} else {
			j++
			array[j], array[right] = array[right], array[j]
		}
		if j <= k {
			left = j + 1
		}
		if k <= j {
			right = j - 1
		}
	}
}

func (s *float64Slice) Max(left, right int) int {
	array := *s
	m := array[left]
	id := left
	for i := left + 1; i <= right; i++ {
		if array[i] > m {
			m = array[i]
			id = i
		}
	}
	return id
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

// Select implements Floyd-Rivest-Selection-Algorithm for finding the k-lowest number
// 600 and 0.5 are constants from the original paper
func (s *boolSlice) Select(left, right, k int) {
	array := *s
	for right > left {
		if right-left > 600 {
			n := float64(right - left + 1)
			i := float64(k - left + 1)
			z := math.Log(n)
			S := 0.5 * math.Exp(2*z/3)
			sd := 0.5 * math.Sqrt(z*S*(n-S)/n) * sign(i-n/2)
			newleft := max(left, int(float64(k)-i*S/n+sd))
			newright := min(right, int(float64(k)+(n-i)*S/n+sd))
			s.Select(newleft, newright, k)
		}
		t := array[k]
		i := left
		j := right
		array[left], array[k] = array[k], array[left]
		if array[right] && !t {
			array[right], array[left] = array[left], array[right]
		}
		for i < j {
			array[i], array[j] = array[j], array[i]
			i++
			j--
			for !array[i] && t {
				i++
			}
			for array[j] && !t {
				j--
			}
		}
		if array[left] == t {
			array[left], array[j] = array[j], array[left]
		} else {
			j++
			array[j], array[right] = array[right], array[j]
		}
		if j <= k {
			left = j + 1
		}
		if k <= j {
			right = j - 1
		}
	}
}

func (s *boolSlice) Max(left, right int) int {
	array := *s
	if array[left] {
		return left
	}
	for i := left + 1; i <= right; i++ {
		if array[i] {
			return i
		}
	}
	return left
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

// Select implements Floyd-Rivest-Selection-Algorithm for finding the k-lowest number
// 600 and 0.5 are constants from the original paper
func (s *stringSlice) Select(left, right, k int) {
	array := *s
	for right > left {
		if right-left > 600 {
			n := float64(right - left + 1)
			i := float64(k - left + 1)
			z := math.Log(n)
			S := 0.5 * math.Exp(2*z/3)
			sd := 0.5 * math.Sqrt(z*S*(n-S)/n) * sign(i-n/2)
			newleft := max(left, int(float64(k)-i*S/n+sd))
			newright := min(right, int(float64(k)+(n-i)*S/n+sd))
			s.Select(newleft, newright, k)
		}
		t := array[k]
		i := left
		j := right
		array[left], array[k] = array[k], array[left]
		if array[right] > t {
			array[right], array[left] = array[left], array[right]
		}
		for i < j {
			array[i], array[j] = array[j], array[i]
			i++
			j--
			for array[i] < t {
				i++
			}
			for array[j] > t {
				j--
			}
		}
		if array[left] == t {
			array[left], array[j] = array[j], array[left]
		} else {
			j++
			array[j], array[right] = array[right], array[j]
		}
		if j <= k {
			left = j + 1
		}
		if k <= j {
			right = j - 1
		}
	}
}

func (s *stringSlice) Max(left, right int) int {
	array := *s
	m := array[left]
	id := left
	for i := left + 1; i <= right; i++ {
		if array[i] > m {
			m = array[i]
			id = i
		}
	}
	return id
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

// Select implements Floyd-Rivest-Selection-Algorithm for finding the k-lowest number
// 600 and 0.5 are constants from the original paper
func (s *bytesSlice) Select(left, right, k int) {
	array := *s
	for right > left {
		if right-left > 600 {
			n := float64(right - left + 1)
			i := float64(k - left + 1)
			z := math.Log(n)
			S := 0.5 * math.Exp(2*z/3)
			sd := 0.5 * math.Sqrt(z*S*(n-S)/n) * sign(i-n/2)
			newleft := max(left, int(float64(k)-i*S/n+sd))
			newright := min(right, int(float64(k)+(n-i)*S/n+sd))
			s.Select(newleft, newright, k)
		}
		t := array[k]
		i := left
		j := right
		array[left], array[k] = array[k], array[left]
		if bytes.Compare(array[right], t) > 0 {
			array[right], array[left] = array[left], array[right]
		}
		for i < j {
			array[i], array[j] = array[j], array[i]
			i++
			j--
			for bytes.Compare(array[i], t) < 0 {
				i++
			}
			for bytes.Compare(array[j], t) > 0 {
				j--
			}
		}
		if bytes.Equal(array[left], t) {
			array[left], array[j] = array[j], array[left]
		} else {
			j++
			array[j], array[right] = array[right], array[j]
		}
		if j <= k {
			left = j + 1
		}
		if k <= j {
			right = j - 1
		}
	}
}

func (s *bytesSlice) Max(left, right int) int {
	array := *s
	m := array[left]
	id := left
	for i := left + 1; i <= right; i++ {
		if bytes.Compare(array[i], m) > 0 {
			m = array[i]
			id = i
		}
	}
	return id
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
