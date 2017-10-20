package flows

import (
	"fmt"
)

// DateTimeSeconds represents time in units of seconds from 00:00 UTC, Januray 1, 1970 according to RFC5102.
type DateTimeSeconds uint64

// DateTimeMilliseconds represents time in units of milliseconds from 00:00 UTC, Januray 1, 1970 according to RFC5102.
type DateTimeMilliseconds uint64

// DateTimeMicroseconds represents time in units of microseconds from 00:00 UTC, Januray 1, 1970 according to RFC5102.
type DateTimeMicroseconds uint64

// DateTimeNanoseconds represents time in units of nanoseconds from 00:00 UTC, Januray 1, 1970 according to RFC5102.
type DateTimeNanoseconds uint64

func ToFloat(a interface{}) float64 {
	switch i := a.(type) {
	case float64:
		return i
	case float32:
		return float64(i)
	case int64:
		return float64(i)
	case int32:
		return float64(i)
	case int16:
		return float64(i)
	case int8:
		return float64(i)
	case int:
		return float64(i)
	case uint64:
		return float64(i)
	case uint32:
		return float64(i)
	case uint16:
		return float64(i)
	case uint8:
		return float64(i)
	case uint:
		return float64(i)
	case DateTimeSeconds:
		return float64(i)
	case DateTimeMilliseconds:
		return float64(i)
	case DateTimeMicroseconds:
		return float64(i)
	case DateTimeNanoseconds:
		return float64(i)
	case nil:
		return 0
	case bool:
		if i {
			return 1
		}
		return 0
	}
	panic(fmt.Sprintf("Can't convert %v to float", a))
}

func ToInt(a interface{}) int64 {
	switch i := a.(type) {
	case float64:
		return int64(i)
	case float32:
		return int64(i)
	case int64:
		return i
	case int32:
		return int64(i)
	case int16:
		return int64(i)
	case int8:
		return int64(i)
	case int:
		return int64(i)
	case uint64:
		return int64(i)
	case uint32:
		return int64(i)
	case uint16:
		return int64(i)
	case uint8:
		return int64(i)
	case uint:
		return int64(i)
	case DateTimeSeconds:
		return int64(i)
	case DateTimeMilliseconds:
		return int64(i)
	case DateTimeMicroseconds:
		return int64(i)
	case DateTimeNanoseconds:
		return int64(i)
	case nil:
		return 0
	case bool:
		if i {
			return 1
		}
		return 0
	}
	panic(fmt.Sprintf("Can't convert %v to int", a))
}

func ToUInt(a interface{}) uint64 {
	switch i := a.(type) {
	case float64:
		return uint64(i)
	case float32:
		return uint64(i)
	case int64:
		return uint64(i)
	case int32:
		return uint64(i)
	case int16:
		return uint64(i)
	case int8:
		return uint64(i)
	case int:
		return uint64(i)
	case uint64:
		return i
	case uint32:
		return uint64(i)
	case uint16:
		return uint64(i)
	case uint8:
		return uint64(i)
	case uint:
		return uint64(i)
	case DateTimeSeconds:
		return uint64(i)
	case DateTimeMilliseconds:
		return uint64(i)
	case DateTimeMicroseconds:
		return uint64(i)
	case DateTimeNanoseconds:
		return uint64(i)
	case nil:
		return 0
	case bool:
		if i {
			return 1
		}
		return 0
	}
	panic(fmt.Sprintf("Can't convert %v to int", a))
}

type NumberType int

const (
	IntType NumberType = iota
	FloatType
	SecondsType
	MillisecondsType
	MicrosecondsType
	NanosecondsType
)

func cleanUp(a interface{}) (ct NumberType, ret interface{}) {
	switch i := a.(type) {
	case float64:
		return FloatType, i
	case float32:
		return FloatType, float64(i)
	case int64:
		return IntType, i
	case int32:
		return IntType, int64(i)
	case int16:
		return IntType, int64(i)
	case int8:
		return IntType, int64(i)
	case int:
		return IntType, int64(i)
	case uint64:
		return IntType, int64(i)
	case uint32:
		return IntType, int64(i)
	case uint16:
		return IntType, int64(i)
	case uint8:
		return IntType, int64(i)
	case uint:
		return IntType, int64(i)
	case DateTimeSeconds:
		return SecondsType, i
	case DateTimeMilliseconds:
		return MillisecondsType, i
	case DateTimeMicroseconds:
		return MicrosecondsType, i
	case DateTimeNanoseconds:
		return NanosecondsType, i
	case bool:
		if i {
			return IntType, int64(1)
		}
		return IntType, int64(0)
	}
	panic(fmt.Sprintf("Can't upconvert %v", a))
}

func intToFloat(val interface{}) float64 {
	return float64(val.(int64))
}

func floatToInt(val interface{}) int64 {
	return int64(val.(float64))
}

func intToTime(val interface{}, kind NumberType) interface{} {
	switch kind {
	case SecondsType:
		return DateTimeSeconds(val.(int64))
	case MillisecondsType:
		return DateTimeMilliseconds(val.(int64))
	case MicrosecondsType:
		return DateTimeMicroseconds(val.(int64))
	case NanosecondsType:
		return DateTimeNanoseconds(val.(int64))
	}
	panic("This should never happen")
}

func floatToTime(val interface{}, kind NumberType) interface{} {
	switch kind {
	case SecondsType:
		return DateTimeSeconds(val.(float64))
	case MillisecondsType:
		return DateTimeMilliseconds(val.(float64))
	case MicrosecondsType:
		return DateTimeMicroseconds(val.(float64))
	case NanosecondsType:
		return DateTimeNanoseconds(val.(float64))
	}
	panic("This should never happen")
}

func scaleTimetoNano(from NumberType, val interface{}) interface{} {
	switch from {
	case SecondsType:
		return DateTimeSeconds(val.(int64) * 1e9)
	case MillisecondsType:
		return DateTimeMilliseconds(val.(int64) * 1e6)
	case MicrosecondsType:
		return DateTimeMicroseconds(val.(int64) * 1e3)
	case NanosecondsType:
		return val
	}
	panic("This should never happen")
}

// UpConvert returns either two Signed64 or two Float64 depending on the numbers
func UpConvert(a, b interface{}) (dst NumberType, fl bool, ai, bi interface{}) {
	var tA, tB NumberType
	tA, ai = cleanUp(a)
	tB, bi = cleanUp(b)
	if tA == tB {
		dst = tA
		fl = tA == FloatType
		return
	}
	if tA == IntType {
		if tB == FloatType {
			return FloatType, true, intToFloat(ai), bi
		}
		return tB, false, intToTime(ai, tB), tB
	}
	if tA == FloatType {
		if tB == IntType {
			return FloatType, true, ai, intToFloat(bi)
		}
		return tB, false, floatToTime(ai, tB), tB
	}
	// both types are time - but differen timebases
	return NanosecondsType, false, scaleTimetoNano(tA, a), scaleTimetoNano(tB, b)
}

func FixType(val interface{}, t NumberType) interface{} {
	switch t {
	case SecondsType:
		return DateTimeSeconds(val.(int64))
	case MillisecondsType:
		return DateTimeMilliseconds(val.(int64))
	case MicrosecondsType:
		return DateTimeMicroseconds(val.(int64))
	case NanosecondsType:
		return DateTimeNanoseconds(val.(int64))
	}
	return val
}
