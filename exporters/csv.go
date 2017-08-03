package exporters

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

type csvExporter struct {
	id         string
	outfile    string
	exportlist chan []string
	finished   chan struct{}
}

func (pe *csvExporter) Fields(fields []string) {
	pe.exportlist <- fields
}

//Export export given features
func (pe *csvExporter) Export(features []flows.Feature, when flows.Time) {
	n := len(features)
	var list = make([]string, n)
	for i, elem := range features {
		switch val := elem.Value().(type) {
		case flows.Number:
			list[i] = fmt.Sprint(val.GoValue())
		case byte:
			list[i] = fmt.Sprint(int(val))
		case flows.Time:
			list[i] = fmt.Sprint(int64(val))
		case flows.FlowEndReason:
			list[i] = fmt.Sprint(int(val))
		default:
			list[i] = fmt.Sprint(val)
		}
	}
	pe.exportlist <- list
}

//Finish Write outstanding data and wait for completion
func (pe *csvExporter) Finish() {
	close(pe.exportlist)
	<-pe.finished
}

func (pe *csvExporter) ID() string {
	return pe.id
}

func (pe *csvExporter) Init() {
	pe.exportlist = make(chan []string, 100)
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
	writer := csv.NewWriter(outfile)
	go func() {
		defer close(pe.finished)
		for data := range pe.exportlist {
			writer.Write(data)
		}
		writer.Flush()
	}()
}

func newCSVExporter(name string, opts interface{}, args []string) (arguments []string, ret flows.Exporter) {
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
		log.Fatalln("CSV exporter needs a filename as argument")
	}
	if name == "" {
		name = "CSV|" + outfile
	}
	ret = &csvExporter{id: name, outfile: outfile}
	return
}

func csvhelp(name string) {
	fmt.Fprintf(os.Stderr, `
The %s exporter writes the output to a csv file with a flow per line and a
header consisting of the feature description.

As argument, the output file is needed.

Usage command line:
  export %s file.csv

Usage json file:
  {
    "type": "%s",
    "options": "file.csv"
  }
`, name, name, name)
}

func init() {
	flows.RegisterExporter("csv", "Exports flows to a csv file.", newCSVExporter, csvhelp)
}
