package exporters

import (
	"fmt"
	"io"
	"log"
	"os"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

type printExporter struct {
	id         string
	outfile    string
	exportlist chan []interface{}
	finished   chan struct{}
}

func (pe *printExporter) Fields([]string) {}

//Export export given features
func (pe *printExporter) Export(template flows.Template, features []flows.Feature, when flows.DateTimeNanoseconds) {
	n := len(features)
	var list = make([]interface{}, n*2)
	for i, elem := range features {
		//list[i*2] = elem.Type() FIXME
		list[i*2+1] = elem.Value()
	}
	pe.exportlist <- list
}

//Finish Write outstanding data and wait for completion
func (pe *printExporter) Finish() {
	close(pe.exportlist)
	<-pe.finished
}

func (pe *printExporter) ID() string {
	return pe.id
}

func (pe *printExporter) Init() {
	pe.exportlist = make(chan []interface{}, 1000)
	pe.finished = make(chan struct{})
	var outfile io.WriteCloser
	if pe.outfile == "-" {
		outfile = os.Stdout
	} else {
		var err error
		outfile, err = os.Create(pe.outfile)
		if err != nil {
			log.Fatal("Couldn't open file ", pe.outfile, err)
		}
	}
	go func() {
		defer outfile.Close()
		defer close(pe.finished)
		for data := range pe.exportlist {
			fmt.Fprintln(outfile, data...)
		}
	}()
}

func newPrintExporter(name string, opts interface{}, args []string) (arguments []string, ret flows.Exporter) {
	var outfile string
	if _, ok := opts.(flows.UseStringOption); ok {
		if len(args) > 0 {
			outfile = args[0]
			arguments = args[1:]
		}
	} else {
		if f, ok := opts.(string); ok {
			outfile = f
		}
	}
	if outfile == "" {
		log.Fatalln("print exporter needs a filename as argument")
	}
	if name == "" {
		name = "PRINT|" + outfile
	}
	ret = &printExporter{id: name, outfile: outfile}
	return
}

func printhelp(name string) {
	fmt.Fprintf(os.Stderr, `
The %s exporter writes the output to a file with one flow per line
and a header consisting of the feature description. Flow features and
header fields are ouput one after another using golang representation of
the individual values.

As argument, the output file is needed.

Usage command line:
  export %s file.out

Usage json file:
  {
    "type": "%s",
    "options": "file.out"
  }
`, name, name, name)
}

func init() {
	flows.RegisterExporter("print", "Exports flows without formating.", newPrintExporter, printhelp)
}
