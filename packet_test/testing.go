package packet_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/packet"
)

type FeatureResult struct {
	Name  string
	Value interface{}
}

type FeatureLine struct {
	When     flows.DateTimeNanoseconds
	Features []FeatureResult
}

type assertExporter struct {
	seen []FeatureLine
}

func makeAssertExporter() *assertExporter {
	return &assertExporter{}
}

func (ae *assertExporter) Fields([]string) {}

//Export export given features
func (ae *assertExporter) Export(template flows.Template, features []flows.Feature, when flows.DateTimeNanoseconds) {
	line := make([]FeatureResult, len(features))
	ies := template.InformationElements()
	for i, feature := range features {
		line[i] = FeatureResult{ies[i].Name, feature.Value()}
	}
	ae.seen = append(ae.seen, FeatureLine{when, line})
}

//Finish Write outstanding data and wait for completion
func (ae *assertExporter) Finish() {
}

func (ae *assertExporter) ID() string {
	return "assert"
}

func (ae *assertExporter) Init() {
}

type TestTable struct {
	exporter *assertExporter
	table    *flows.FlowTable
	t        *testing.T
}

func MakeFeatureTest(t *testing.T, features []string, ft flows.FeatureType, opt flows.FlowOptions) (ret TestTable) {
	ret.t = t
	if opt.ActiveTimeout == 0 {
		opt.ActiveTimeout = flows.SecondsInNanoseconds * 1800
	}
	if opt.IdleTimeout == 0 {
		opt.IdleTimeout = flows.SecondsInNanoseconds * 300
	}
	ret.exporter = makeAssertExporter()
	featuresI := make([]interface{}, len(features))
	for i, feature := range features {
		featuresI[i] = feature
	}
	var f flows.RecordListMaker
	f.AppendRecord(featuresI, []flows.Exporter{ret.exporter}, ft)
	ret.table = flows.NewFlowTable(f, packet.NewFlow, opt, true, 0)
	return
}

func MakeFlowFeatureTest(t *testing.T, feature string) TestTable {
	return MakeFeatureTest(t, []string{feature}, flows.FlowFeature, flows.FlowOptions{})
}

func MakePacketFeatureTest(t *testing.T, feature string) TestTable {
	return MakeFeatureTest(t, []string{feature}, flows.PacketFeature, flows.FlowOptions{PerPacket: true})
}

func (t *TestTable) EventLayers(when flows.DateTimeNanoseconds, layerList ...packet.SerializableLayerType) {
	data := packet.BufferFromLayers(when, layerList...)
	key, fw := packet.Fivetuple(data, false)
	data.SetInfo(key, fw)
	t.table.Event(data)
}

func (t *TestTable) Finish(when flows.DateTimeNanoseconds) {
	t.table.EOF(when)
	t.exporter.Finish()
}

func pprintTime(t flows.DateTimeNanoseconds) string {
	if t == 0 {
		return "0ns"
	}
	unit := []string{"ns", "us", "ms", "s", "min"}
	div := []flows.DateTimeNanoseconds{1000, 1000, 1000, 60, 60}
	for i := 0; i < len(div); i++ {
		if t%div[i] == 0 {
			t /= div[i]
		} else {
			return fmt.Sprintf("%d%s", t, unit[i])
		}
	}
	return fmt.Sprintf("%dh", t)
}

func (t *TestTable) sprintObserved() string {
	var lines []string
	for _, line := range t.exporter.seen {
		lines = append(lines, fmt.Sprintf("@%s:", pprintTime(line.When)))
		for _, feature := range line.Features {
			lines = append(lines, fmt.Sprintf("\t%s = %v", feature.Name, feature.Value))
		}
	}
	return strings.Join(lines, "\n")
}

func (t *TestTable) Fail(msg string) {
	t.t.Errorf("%s\nObserved Features:\n%s\n", msg, t.sprintObserved())
}

func (t *TestTable) AssertFeatureValue(name string, value interface{}) {
	for _, line := range t.exporter.seen {
		for _, feature := range line.Features {
			if name == feature.Name && reflect.DeepEqual(value, feature.Value) {
				return
			}
		}
	}
	t.Fail(fmt.Sprintf("Couldn't observe %s == %v", name, value))
}

func (t *TestTable) AssertFeatureList(result []FeatureLine) {
	for i, line := range result {
		if i >= len(t.exporter.seen) {
			t.Fail(fmt.Sprintf("Expected another result line - but only %d seen", i))
			return
		}
		realLine := t.exporter.seen[i]
		if line.When != realLine.When {
			t.Fail(fmt.Sprintf("Result time #%d mismatches (is %s, expected %s)", i+1, pprintTime(line.When), pprintTime(realLine.When)))
			return
		}
		for j, feature := range line.Features {
			if j >= len(realLine.Features) {
				t.Fail(fmt.Sprintf("Expected %d features @%s, but only %d seen", len(line.Features), pprintTime(line.When), len(realLine.Features)))
				return
			}
			realFeature := realLine.Features[j]
			if feature.Name != realFeature.Name || !reflect.DeepEqual(feature.Value, realFeature.Value) {
				t.Fail(fmt.Sprintf("Result @%s mismatches; wanted %s = %v, but got %s = %v", pprintTime(line.When), feature.Name, feature.Value, realFeature.Name, realFeature.Value))
				return
			}
		}
	}
}
