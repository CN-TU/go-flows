package null

import (
	"fmt"
	"os"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/util"
)

type nullExporter struct {
}

func (pe *nullExporter) Fields(fields []string) {}

//Export export given features
func (pe *nullExporter) Export(template flows.Template, features []interface{}, when flows.DateTimeNanoseconds) {
}

//Finish Write outstanding data and wait for completion
func (pe *nullExporter) Finish() {}

func (pe *nullExporter) ID() string { return "null" }

func (pe *nullExporter) Init() {}

func newNullExporter(args []string) (arguments []string, ret util.Module, err error) {
	arguments = args
	ret = &nullExporter{}
	return
}

func nullHelp(name string) {
	fmt.Fprintf(os.Stderr, `
The %s exporter does not write out anything.
`, name)
}

func init() {
	flows.RegisterExporter("null", "Exports nothing.", newNullExporter, nullHelp)
}
