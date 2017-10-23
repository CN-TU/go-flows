package flows

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
)

type UseStringOption struct{}

// Exporter represents a genric exporter
type Exporter interface {
	// Export gets called upon flow export with a list of features and the export time.
	Export(Template, []Feature, DateTimeNanoseconds)
	// Finish gets called before program exit. Eventual flushing needs to be implemented here.
	Fields([]string)
	Finish()
	ID() string
	Init()
}

type exportModule struct {
	name, desc string
	new        func(string, interface{}, []string) ([]string, Exporter)
	help       func(string)
}

var exporters = make(map[string]exportModule)

func RegisterExporter(name, desc string, new func(string, interface{}, []string) ([]string, Exporter), help func(string)) {
	exporters[name] = exportModule{name, desc, new, help}
}

func ExporterHelp(which string) error {
	if exporter, ok := exporters[which]; ok {
		exporter.help(which)
		return nil
	}
	return errors.New("Exporter not found")
}

func MakeExporter(which, name string, options interface{}, args []string) ([]string, Exporter) {
	if exporter, ok := exporters[which]; ok {
		return exporter.new(name, options, args)
	}
	return nil, nil
}

func ListExporters(w io.Writer) {
	var names []string
	t := tabwriter.NewWriter(w, 3, 4, 5, ' ', 0)
	for exporter := range exporters {
		names = append(names, exporter)
	}
	sort.Strings(names)
	for _, name := range names {
		line := new(bytes.Buffer)
		fmt.Fprintf(line, "%s\t%s\n", name, exporters[name].desc)
		t.Write(line.Bytes())
	}
	t.Flush()
}
