package exporters

import (
	"fmt"
	"io"
	"log"
	"os"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

type printExporter struct {
	outfile    string
	exportlist chan []interface{}
	finished   chan struct{}
}

func (pe *printExporter) Fields([]string) {}

//Export export given features
func (pe *printExporter) Export(features []flows.Feature, when flows.Time) {
	n := len(features)
	var list = make([]interface{}, n*2)
	for i, elem := range features {
		list[i*2] = elem.Type()
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
	return "PRINT|" + pe.outfile
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

func newPrintExporter(args []string) ([]string, flows.Exporter) {
	if len(args) < 1 {
		return nil, nil
	}
	return args[1:], &msgPack{outfile: args[0]}
}

func printhelp() {
	log.Fatal("not implemented")
}

func init() {
	flows.RegisterExporter("print", "Exports flows without formating.", newPrintExporter, printhelp)
}
