// Package packet_test contains some utility functions for creating feature tests.
package packet_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/packet"
)

// FeatureResult holds the result for the named feature
type FeatureResult struct {
	// Name of the feature
	Name string
	// Value of the feature
	Value interface{}
}

// FeatureLine holds one record (i.e. Feature results + point in time)
type FeatureLine struct {
	// When is the point in time this was exported
	When flows.DateTimeNanoseconds
	// Features is a list of feature results
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

// TestTable is flow table implementation which can be used for testing
type TestTable struct {
	selector packet.DynamicKeySelector
	exporter *assertExporter
	table    *flows.FlowTable
	t        *testing.T
}

// MakeFeatureTest creates a flow table for testing purposes with the given features, wanted return type, and flow options
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
	ret.selector = packet.MakeDynamicKeySelector([]string{
		"sourceIPAddress",
		"destinationIPAddress",
		"protocolIdentifier",
		"sourceTransportPort",
		"destinationTransportPort",
	}, true, false)
	return
}

// MakeFlowFeatureTest returns a flow table for testing a single flow feature
func MakeFlowFeatureTest(t *testing.T, feature string) TestTable {
	return MakeFeatureTest(t, []string{feature}, flows.FlowFeature, flows.FlowOptions{})
}

// MakePacketFeatureTest returns a flow table for testing a single packet feature
func MakePacketFeatureTest(t *testing.T, feature string) TestTable {
	return MakeFeatureTest(t, []string{feature}, flows.PacketFeature, flows.FlowOptions{ActiveTimeout: 0})
}

// EventLayers simulates a packet arriving at the given point in time with the given layers populated
func (t *TestTable) EventLayers(when flows.DateTimeNanoseconds, layerList ...packet.SerializableLayerType) {
	data := packet.BufferFromLayers(when, layerList...)
	key, fw, _ := t.selector.Key(data)
	data.SetInfo(key, fw)
	t.table.Event(data)
}

// Finish finalizes all the flows in the table
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

func (t *TestTable) fail(msg string) {
	t.t.Errorf("%s\nObserved Features:\n%s\n", msg, t.sprintObserved())
}

// AssertFeatureValue fails the test if the named feature did not return the expected value
func (t *TestTable) AssertFeatureValue(name string, value interface{}) {
	for _, line := range t.exporter.seen {
		for _, feature := range line.Features {
			if name == feature.Name && reflect.DeepEqual(value, feature.Value) {
				return
			}
		}
	}
	t.fail(fmt.Sprintf("Couldn't observe %s == %v", name, value))
}

// AssertFeatureList fails the test if the table did not export the given resultset
func (t *TestTable) AssertFeatureList(result []FeatureLine) {
	for i, line := range result {
		if i >= len(t.exporter.seen) {
			t.fail(fmt.Sprintf("Expected another result line - but only %d seen", i))
			return
		}
		realLine := t.exporter.seen[i]
		if line.When != realLine.When {
			t.fail(fmt.Sprintf("Result time #%d mismatches (is %s, expected %s)", i+1, pprintTime(line.When), pprintTime(realLine.When)))
			return
		}
		for j, feature := range line.Features {
			if j >= len(realLine.Features) {
				t.fail(fmt.Sprintf("Expected %d features @%s, but only %d seen", len(line.Features), pprintTime(line.When), len(realLine.Features)))
				return
			}
			realFeature := realLine.Features[j]
			if feature.Name != realFeature.Name || !reflect.DeepEqual(feature.Value, realFeature.Value) {
				t.fail(fmt.Sprintf("Result @%s mismatches; wanted %s = %v, but got %s = %v", pprintTime(line.When), feature.Name, feature.Value, realFeature.Name, realFeature.Value))
				return
			}
		}
	}
}
