// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"text/template"
	"time"
)

// This binary generates the math functions
//
// go run gen_math.go | gofmt > math_generated.go

type operation struct {
	Name, Operator, Description string
}

var single = [...]operation{
	{Name: "floor", Operator: "math.Floor", Description: "returns ⌊a⌋"},
	{Name: "ceil", Operator: "math.Ceil", Description: "returns ⌈a⌉"},
	{Name: "log", Operator: "math.Log", Description: "returns log(a)"},
	{Name: "exp", Operator: "math.Exp", Description: "returns exp(a)"},
}

var dual = [...]operation{
	{Name: "add", Operator: "+", Description: "returns a + b"},
	{Name: "subtract", Operator: "-", Description: "returns a - b"},
	{Name: "multiply", Operator: "*", Description: "returns a * b"},
	{Name: "divide", Operator: "/", Description: "returns a / b"},
}

const heading = `package operations

// Created by gen_math.go, don't edit manually!
// Generated at %s

import (
	"math"
	"github.com/CN-TU/go-flows/flows"
)

`

var singleTmpl = template.Must(template.New("comparsion").Parse(`
type {{.Name}}Packet struct {
	flows.BaseFeature
}

func (f *{{.Name}}Packet) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.SetValue({{.Operator}}(flows.ToFloat(new)), context, f)
}

type {{.Name}}Flow struct {
	flows.BaseFeature
}

func (f *{{.Name}}Flow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		f.SetValue({{.Operator}}(flows.ToFloat(new)), context, f)
	}
}

func init() {
	flows.RegisterFunction("{{.Name}}", "{{.Description}}", flows.PacketFeature, func() flows.Feature { return &{{.Name}}Packet{} }, flows.PacketFeature, flows.PacketFeature)
	flows.RegisterFunction("{{.Name}}", "{{.Description}}", flows.FlowFeature, func() flows.Feature { return &{{.Name}}Flow{} }, flows.FlowFeature, flows.FlowFeature)
}`))

var dualTmpl = template.Must(template.New("comparsion").Parse(`
type {{.Name}}Packet struct {
	flows.MultiBasePacketFeature
}

func (f *{{.Name}}Packet) Event(new interface{}, context *flows.EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}

	dst, fl, a, b := flows.UpConvert(values[0], values[1])
	var result interface{}
	switch fl {
	case flows.UIntType:
		result = a.(uint64) {{.Operator}} b.(uint64)
	case flows.IntType:
		result = a.(int64) {{.Operator}} b.(int64)
	case flows.FloatType:
		result = a.(float64) {{.Operator}} b.(float64)
	}
	f.SetValue(flows.FixType(result, dst), context, f)
}

type {{.Name}}Flow struct {
	flows.MultiBaseFlowFeature
}

func (f *{{.Name}}Flow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	values := f.GetValues()

	dst, fl, a, b := flows.UpConvert(values[0], values[1])
	var result interface{}
	switch fl {
	case flows.UIntType:
		result = a.(uint64) {{.Operator}} b.(uint64)
	case flows.IntType:
		result = a.(int64) {{.Operator}} b.(int64)
	case flows.FloatType:
		result = a.(float64) {{.Operator}} b.(float64)
	}
	f.SetValue(flows.FixType(result, dst), context, f)
}

func init() {
	flows.RegisterFunction("{{.Name}}", "{{.Description}}", flows.PacketFeature, func() flows.Feature { return &{{.Name}}Packet{} }, flows.PacketFeature, flows.PacketFeature)
	flows.RegisterFunction("{{.Name}}", "{{.Description}}", flows.FlowFeature, func() flows.Feature { return &{{.Name}}Flow{} }, flows.FlowFeature, flows.FlowFeature)
}`))

func main() {
	fmt.Printf(heading, time.Now())
	for _, comparison := range single {
		if err := singleTmpl.Execute(os.Stdout, comparison); err != nil {
			log.Fatal(err)
		}
	}
	for _, comparison := range dual {
		if err := dualTmpl.Execute(os.Stdout, comparison); err != nil {
			log.Fatal(err)
		}
	}
}
