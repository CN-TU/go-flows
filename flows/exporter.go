package flows

import (
	"github.com/CN-TU/go-flows/util"
)

const exporterName = "exporter"

// Exporter represents a generic exporter
type Exporter interface {
	util.Module
	// Export gets called upon record export with a list of features and the export time.
	Export(Template, []Feature, DateTimeNanoseconds)
	// Fields gets called during flow-exporter initialization with the list of fieldnames as argument
	Fields([]string)
	// Finish gets called before program exit. Eventual flushing needs to be implemented here.
	Finish()
}

// RegisterExporter registers an exporter (see module system in util)
func RegisterExporter(name, desc string, new util.ModuleCreator, help util.ModuleHelp) {
	util.RegisterModule(exporterName, name, desc, new, help)
}

// ExporterHelp displays help for a specific exporter (see module system in util)
func ExporterHelp(which string) error {
	return util.GetModuleHelp(exporterName, which)
}

// MakeExporter creates an exporter instance (see module system in util)
func MakeExporter(which, name string, options interface{}, args []string) ([]string, Exporter, error) {
	args, module, err := util.CreateModule(exporterName, which, name, options, args)
	if err != nil {
		return args, nil, err
	}
	return args, module.(Exporter), nil
}

// ListExporters returns a list of exporters (see module system in util)
func ListExporters() ([]util.ModuleDescription, error) {
	return util.GetModules(exporterName)
}
