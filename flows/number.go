package flows

type Unsigned8 uint8
type Unsigned16 uint16
type Unsigned32 uint32
type Unsigned64 uint64
type Signed8 int8
type Signed16 int16
type Signed32 int32
type Signed64 int64
type Float32 float32
type Float64 float64

type Number interface {
	Add(Number) Number
	ToFloat() float64
	Less(Number) bool
	Greater(Number) bool
}

func (a Unsigned8) Add(x Number) Number {
	return a + x.(Unsigned8)
}
func (a Unsigned8) ToFloat() float64 {
	return float64(a)
}
func (a Unsigned8) Less(x Number) bool {
	return a < x.(Unsigned8)
}
func (a Unsigned8) Greater(x Number) bool {
	return a > x.(Unsigned8)
}

func (a Unsigned16) Add(x Number) Number {
	return a + x.(Unsigned16)
}
func (a Unsigned16) ToFloat() float64 {
	return float64(a)
}
func (a Unsigned16) Less(x Number) bool {
	return a < x.(Unsigned16)
}
func (a Unsigned16) Greater(x Number) bool {
	return a > x.(Unsigned16)
}

func (a Unsigned32) Add(x Number) Number {
	return a + x.(Unsigned32)
}
func (a Unsigned32) ToFloat() float64 {
	return float64(a)
}
func (a Unsigned32) Less(x Number) bool {
	return a < x.(Unsigned32)
}
func (a Unsigned32) Greater(x Number) bool {
	return a > x.(Unsigned32)
}

func (a Unsigned64) Add(x Number) Number {
	return a + x.(Unsigned64)
}
func (a Unsigned64) ToFloat() float64 {
	return float64(a)
}
func (a Unsigned64) Less(x Number) bool {
	return a < x.(Unsigned64)
}
func (a Unsigned64) Greater(x Number) bool {
	return a > x.(Unsigned64)
}

func (a Signed8) Add(x Number) Number {
	return a + x.(Signed8)
}
func (a Signed8) ToFloat() float64 {
	return float64(a)
}
func (a Signed8) Less(x Number) bool {
	return a < x.(Signed8)
}
func (a Signed8) Greater(x Number) bool {
	return a > x.(Signed8)
}

func (a Signed16) Add(x Number) Number {
	return a + x.(Signed16)
}
func (a Signed16) ToFloat() float64 {
	return float64(a)
}
func (a Signed16) Less(x Number) bool {
	return a < x.(Signed16)
}
func (a Signed16) Greater(x Number) bool {
	return a > x.(Signed16)
}

func (a Signed32) Add(x Number) Number {
	return a + x.(Signed32)
}
func (a Signed32) ToFloat() float64 {
	return float64(a)
}
func (a Signed32) Less(x Number) bool {
	return a < x.(Signed32)
}
func (a Signed32) Greater(x Number) bool {
	return a > x.(Signed32)
}

func (a Signed64) Add(x Number) Number {
	return a + x.(Signed64)
}
func (a Signed64) ToFloat() float64 {
	return float64(a)
}
func (a Signed64) Less(x Number) bool {
	return a < x.(Signed64)
}
func (a Signed64) Greater(x Number) bool {
	return a > x.(Signed64)
}

func (a Float32) Add(x Number) Number {
	return a + x.(Float32)
}
func (a Float32) ToFloat() float64 {
	return float64(a)
}
func (a Float32) Less(x Number) bool {
	return a < x.(Float32)
}
func (a Float32) Greater(x Number) bool {
	return a > x.(Float32)
}

func (a Float64) Add(x Number) Number {
	return a + x.(Float64)
}
func (a Float64) ToFloat() float64 {
	return float64(a)
}
func (a Float64) Less(x Number) bool {
	return a < x.(Float64)
}
func (a Float64) Greater(x Number) bool {
	return a > x.(Float64)
}
