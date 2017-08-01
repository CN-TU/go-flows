package flows

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
	new        func(args []string) ([]string, Exporter)
	help       func()
}

var exporters = make(map[string]exportModule)

func RegisterExporter(name, desc string, new func(args []string) ([]string, Exporter), help func()) {
	exporters[name] = exportModule{name, desc, new, help}
}

func MakeExporter(name string, args []string) ([]string, Exporter) {
	if exporter, ok := exporters[name]; ok {
		return exporter.new(args)
	}
	return nil, nil
}
