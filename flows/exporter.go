package flows

type UseStringOption struct{}

// Exporter represents a genric exporter
type Exporter interface {
	// Export gets called upon flow export with a list of features and the export time.
	Export([]Feature, Time)
	// Fields gets called during flow list creation. A list of flow base types is supplied.
	Fields([]string)
	// Finish gets called before program exit. Eventual flushing needs to be implemented here.
	Finish()
	ID() string
	Init()
}

type exportModule struct {
	name, desc string
	new        func(string, interface{}, []string) ([]string, Exporter)
	help       func()
}

var exporters = make(map[string]exportModule)

func RegisterExporter(name, desc string, new func(string, interface{}, []string) ([]string, Exporter), help func()) {
	exporters[name] = exportModule{name, desc, new, help}
}

func MakeExporter(which, name string, options interface{}, args []string) ([]string, Exporter) {
	if exporter, ok := exporters[which]; ok {
		return exporter.new(name, options, args)
	}
	return nil, nil
}
