package flows

type Unsigned8 uint8   //type
type Unsigned16 uint16 //type
type Unsigned32 uint32 //type
type Unsigned64 uint64 //type
type Signed8 int8      //type
type Signed16 int16    //type
type Signed32 int32    //type
type Signed64 int64    //type
type Float32 float32   //type
type Float64 float64   //type

type Number interface {
	Add(Number) Number //oper:a+b
	Log() Number       //oper:Float64(math.Log(float64(a)))

	Less(Number) bool    //oper:a<b
	Greater(Number) bool //oper:a>b

	ToFloat() float64 //oper:float64(a)
	ToInt() int64     //oper:int64(a)
	ToUint() uint64   //oper:uint64(a)
	To64() Number
}

//go:generate go run tool/number_generate.go

func (a Unsigned8) To64() Number {
	return Unsigned64(a)
}
func (a Unsigned16) To64() Number {
	return Unsigned64(a)
}
func (a Unsigned32) To64() Number {
	return Unsigned64(a)
}
func (a Unsigned64) To64() Number {
	return Unsigned64(a)
}
func (a Signed8) To64() Number {
	return Signed64(a)
}
func (a Signed16) To64() Number {
	return Signed64(a)
}
func (a Signed32) To64() Number {
	return Signed64(a)
}
func (a Signed64) To64() Number {
	return Signed64(a)
}
func (a Float32) To64() Number {
	return Float64(a)
}
func (a Float64) To64() Number {
	return Float64(a)
}
