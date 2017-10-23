package packet

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

type featureResult struct {
	name  string
	value interface{}
}

type featureLine struct {
	when     flows.DateTimeNanoseconds
	features []featureResult
}

type assertExporter struct {
	seen []featureLine
}

func makeAssertExporter() *assertExporter {
	return &assertExporter{}
}

func (ae *assertExporter) Fields([]string) {}

//Export export given features
func (ae *assertExporter) Export(template flows.Template, features []flows.Feature, when flows.DateTimeNanoseconds) {
	line := make([]featureResult, len(features))
	ies := template.InformationElements()
	for i, feature := range features {
		line[i] = featureResult{ies[i].Name, feature.Value()}
	}
	ae.seen = append(ae.seen, featureLine{when, line})
}

//Finish Write outstanding data and wait for completion
func (ae *assertExporter) Finish() {
}

func (ae *assertExporter) ID() string {
	return "assert"
}

func (ae *assertExporter) Init() {
}

type testTable struct {
	exporter *assertExporter
	table    *flows.FlowTable
	t        *testing.T
}

func makeFeatureTest(t *testing.T, features []string, ft flows.FeatureType, opt flows.FlowOptions) (ret testTable) {
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
	ret.table = flows.NewFlowTable(f, NewFlow, opt, true)
	return
}

func makeFlowFeatureTest(t *testing.T, feature string) testTable {
	return makeFeatureTest(t, []string{feature}, flows.FlowFeature, flows.FlowOptions{})
}

func makePacketFeatureTest(t *testing.T, feature string) testTable {
	return makeFeatureTest(t, []string{feature}, flows.PacketFeature, flows.FlowOptions{PerPacket: true})
}

func (t *testTable) EventLayers(when flows.DateTimeNanoseconds, layerList ...SerializableLayerType) {
	data := bufferFromLayers(when, layerList...)
	key, fw := fivetuple(data)
	data.setInfo(key, fw)
	t.table.Event(data)
}

func (t *testTable) Finish(when flows.DateTimeNanoseconds) {
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

func (t *testTable) sprintObserved() string {
	var lines []string
	for _, line := range t.exporter.seen {
		lines = append(lines, fmt.Sprintf("@%s:", pprintTime(line.when)))
		for _, feature := range line.features {
			lines = append(lines, fmt.Sprintf("\t%s = %v", feature.name, feature.value))
		}
	}
	return strings.Join(lines, "\n")
}

func (t *testTable) fail(msg string) {
	t.t.Errorf("%s\nObserved Features:\n%s\n", msg, t.sprintObserved())
}

func (t *testTable) assertFeatureValue(name string, value interface{}) {
	for _, line := range t.exporter.seen {
		for _, feature := range line.features {
			if name == feature.name && reflect.DeepEqual(value, feature.value) {
				return
			}
		}
	}
	t.fail(fmt.Sprintf("Couldn't observe %s == %v", name, value))
}

func (t *testTable) assertFeatureList(result []featureLine) {
	for i, line := range result {
		if i >= len(t.exporter.seen) {
			t.fail(fmt.Sprintf("Expected another result line - but only %d seen", i))
			return
		}
		realLine := t.exporter.seen[i]
		if line.when != realLine.when {
			t.fail(fmt.Sprintf("Result time #%d mismatches (is %s, expected %s)", i+1, pprintTime(line.when), pprintTime(realLine.when)))
			return
		}
		for j, feature := range line.features {
			if j >= len(realLine.features) {
				t.fail(fmt.Sprintf("Expected %d features @%s, but only %d seen", len(line.features), pprintTime(line.when), len(realLine.features)))
				return
			}
			realFeature := realLine.features[j]
			if feature.name != realFeature.name || !reflect.DeepEqual(feature.value, realFeature.value) {
				t.fail(fmt.Sprintf("Result @%s mismatches; wanted %s = %v, but got %s = %v", pprintTime(line.when), feature.name, feature.value, realFeature.name, realFeature.value))
				return
			}
		}
	}
}
