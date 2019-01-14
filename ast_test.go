package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/CN-TU/go-flows/flows"
	_ "github.com/CN-TU/go-flows/modules/features/nta"
	ipfix "github.com/CN-TU/go-ipfix"
)

type testResolve struct {
	flows.MultiBasePacketFeature
}

func (f *testResolve) Event(new interface{}, context *flows.EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}
	f.SetValue(values, context, f)
}

func resolveTestResolve(args []ipfix.InformationElement) (ipfix.InformationElement, error) {
	if len(args) < 2 {
		return ipfix.InformationElement{}, errors.New("testResolve must have at least one argument")
	}
	first := args[0]
	for _, arg := range args {
		if first != arg {
			return ipfix.InformationElement{}, flows.MakeIncompatibleVariantError("all argumemts to testResolve must have the same type, but got %s", args)
		}
	}
	return ipfix.NewBasicList("accumulate", args[0], 0), nil
}

func errid2String(id int) string {
	switch id {
	case -1:
		return "no error"
	case 0:
		return "general error"
	default:
		return fmt.Sprintf("error in feature %d", id)
	}
}

func TestAST(t *testing.T) {
	flows.RegisterCustomFunction("__testResolve", "returns arguments as list", resolveTestResolve, flows.MatchType, func() flows.Feature { return &testResolve{} }, flows.MatchType, flows.Ellipsis)
	for i, test := range []struct {
		def []interface{}
		err int
	}{
		{
			[]interface{}{
				"__nonexistent__",
			},
			1,
		},
		{
			[]interface{}{
				[]interface{}{"mean", "__nonexistent__"},
			},
			1,
		},
		{
			[]interface{}{
				"mean",
			},
			1,
		},
		{
			[]interface{}{
				[]interface{}{"mean", "minimumIpTotalLength"},
			},
			1,
		},
		{
			[]interface{}{
				[]interface{}{"mean", "sourceIPAddress"},
			},
			1,
		},
		{
			[]interface{}{
				"flowStartSeconds",
				"__NTAFlowID",
				"__NTAProtocol",
				"__NTAPorts",
				"__NTATData",
				"packetTotalCount",
				[]interface{}{"count", "__NTASecWindow"},
				[]interface{}{"divide", "__NTATData", []interface{}{"count", "__NTASecWindow"}},
				[]interface{}{"divide", "packetTotalCount", []interface{}{"count", "__NTASecWindow"}},
				[]interface{}{"max", []interface{}{"__NTATOn", "__NTASecWindow"}},
				[]interface{}{"min", []interface{}{"__NTATOn", "__NTASecWindow"}},
				[]interface{}{"max", []interface{}{"__NTATOff", "__NTASecWindow"}},
				[]interface{}{"min", []interface{}{"__NTATOff", "__NTASecWindow"}},
				[]interface{}{"count", []interface{}{"__NTATOn", "__NTASecWindow"}},
			},
			-1,
		},
		{
			[]interface{}{
				"sourceIPAddress",
				"destinationIPAddress",
				"protocolIdentifier",
				"sourceTransportPort",
				"destinationTransportPort",
				[]interface{}{"mean", "octetTotalCount"},
				"flowEndReason",
				"flowEndNanoseconds",
				"ipTotalLength",
				[]interface{}{"apply", "ipTotalLength", []interface{}{"select", []interface{}{"less", "ipTotalLength", 80}}},
				"minimumIpTotalLength",
				"maximumIpTotalLength",
			},
			-1,
		},
		{
			[]interface{}{
				[]interface{}{"apply", []interface{}{"min", "ipTotalLength"}, "forward"},
				[]interface{}{"apply", []interface{}{"min", "ipTotalLength"}, "backward"},
				[]interface{}{"apply", []interface{}{"max", "ipTotalLength"}, "forward"},
				[]interface{}{"apply", []interface{}{"max", "ipTotalLength"}, "backward"},
			},
			-1,
		},
		{
			[]interface{}{
				[]interface{}{"apply", []interface{}{"apply", []interface{}{"min", "ipTotalLength"}, "backward"}, "forward"},
			},
			-1,
		},
		{
			[]interface{}{
				"sourceIPAddress",
				"sourceIPAddress",
			},
			2,
		},
		{
			[]interface{}{
				[]interface{}{"__testResolve", "destinationIPAddress", "sourceIPAddress"},
			},
			1,
		},
		{
			[]interface{}{
				"sourceIPAddress",
				"destinationIPAddress",
				[]interface{}{"__testResolve", "sourceIPAddress", "sourceIPAddress"},
				[]interface{}{"__testResolve", "destinationIPAddress", "destinationIPAddress", "destinationIPAddress"},
				[]interface{}{"__testResolve", []interface{}{"__testResolve", "sourceIPAddress", "sourceIPAddress"}, []interface{}{"__testResolve", "sourceIPAddress", "sourceIPAddress", "sourceIPAddress"}},
			},
			-1,
		},
	} {
		rl := flows.RecordListMaker{}
		err := rl.AppendRecord(test.def, nil, testing.Verbose())
		id := -1
		if err != nil {
			if e, ok := err.(flows.FeatureError); ok {
				id = e.ID()
			} else {
				id = 0
			}
		}
		if test.err != id {
			expected := errid2String(test.err)
			got := errid2String(id)
			t.Errorf("test %d: expected %s but got %s %s (%#v)", i, expected, got, err, err)
		}
	}
}
