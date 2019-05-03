// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"text/template"
	"time"
)

// This binary generates the logic functions
//
// go run gen_logic.go | gofmt > logic_generated.go

type operation struct {
	Name, Operator, Description string
}

var comparisons = [...]operation{
	{Name: "geq", Operator: ">=", Description: "returns true if a >= b"},
	{Name: "leq", Operator: "<=", Description: "returns true if a <= b"},
	{Name: "less", Operator: "<", Description: "returns true if a < b"},
	{Name: "greater", Operator: ">", Description: "returns true if a > b"},
	{Name: "equal", Operator: "==", Description: "returns true if a == b"},
}

const heading = `package operations

// Created by gen_logic.go, don't edit manually!
// Generated at %s

import (
	"github.com/CN-TU/go-flows/flows"
	ipfix "github.com/CN-TU/go-ipfix"
)

`

var comparisonTmpl = template.Must(template.New("comparsion").Parse(`
type {{.Name}}Packet struct {
	flows.MultiBasePacketFeature
}

func (f *{{.Name}}Packet) Event(new interface{}, context *flows.EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}

	_, fl, a, b := flows.UpConvert(values[0], values[1])
	switch fl {
	case flows.UIntType:
		f.SetValue(a.(uint64) {{.Operator}} b.(uint64), context, f)
	case flows.IntType:
		f.SetValue(a.(int64) {{.Operator}} b.(int64), context, f)
	case flows.FloatType:
		f.SetValue(a.(float64) {{.Operator}} b.(float64), context, f)
	}
}

type {{.Name}}Flow struct {
	flows.MultiBaseFlowFeature
}

func (f *{{.Name}}Flow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	values := f.GetValues(context)

	_, fl, a, b := flows.UpConvert(values[0], values[1])
	switch fl {
	case flows.UIntType:
		f.SetValue(a.(uint64) {{.Operator}} b.(uint64), context, f)
	case flows.IntType:
		f.SetValue(a.(int64) {{.Operator}} b.(int64), context, f)
	case flows.FloatType:
		f.SetValue(a.(float64) {{.Operator}} b.(float64), context, f)
	}
}

func init() {
	flows.RegisterTypedFunction("{{.Name}}", "{{.Description}}", ipfix.BooleanType, 0, flows.PacketFeature, func() flows.Feature { return &{{.Name}}Packet{} }, flows.PacketFeature, flows.PacketFeature)
	flows.RegisterTypedFunction("{{.Name}}", "{{.Description}}", ipfix.BooleanType, 0, flows.FlowFeature, func() flows.Feature { return &{{.Name}}Flow{} }, flows.FlowFeature, flows.FlowFeature)
}`))

func main() {
	fmt.Printf(heading, time.Now())
	for _, comparison := range comparisons {
		if err := comparisonTmpl.Execute(os.Stdout, comparison); err != nil {
			log.Fatal(err)
		}
	}
}
