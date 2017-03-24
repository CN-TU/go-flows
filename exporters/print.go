package exporters

import (
	"fmt"
	"io"
	"log"
	"os"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

type printExporter struct {
	exportlist chan []interface{}
	finished   chan struct{}
}

//Export export given features
func (pe *printExporter) Export(features []flows.Feature, reason flows.FlowEndReason, when flows.Time) {
	n := len(features)
	var list = make([]interface{}, n*2+2)
	for i, elem := range features {
		list[i*2] = elem.Type()
		list[i*2+1] = elem.Value()
	}
	list[n*2] = reason
	list[n*2+1] = when
	pe.exportlist <- list
}

//Finish Write outstanding data and wait for completion
func (pe *printExporter) Finish() {
	close(pe.exportlist)
	<-pe.finished
}

//NewPrintExporter Create a new exporter that just writes the features to filename (- for stdout)
func NewPrintExporter(filename string) flows.Exporter {
	ret := &printExporter{make(chan []interface{}, 1000), make(chan struct{})}
	var outfile io.Writer
	if filename == "-" {
		outfile = os.Stdout
	} else {
		var err error
		outfile, err = os.Create(filename)
		if err != nil {
			log.Fatal("Couldn't open file ", filename, err)
		}
	}
	go func() {
		defer close(ret.finished)
		for data := range ret.exportlist {
			fmt.Fprintln(outfile, data...)
		}
	}()
	return ret
}
