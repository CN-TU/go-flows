package exporters

import (
	"fmt"
	"io"
	"log"
	"os"

	"encoding/csv"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

type csvExporter struct {
	exportlist chan []interface{}
	finished   chan struct{}
}

func (pe *csvExporter) Fields(fields []string) {
	list := make([]interface{}, len(fields))
	for i, elem := range fields {
		list[i] = elem
	}
	pe.exportlist <- list
}

//Export export given features
func (pe *csvExporter) Export(features []flows.Feature, when flows.Time) {
	n := len(features)
	var list = make([]interface{}, n)
	for i, elem := range features {
		list[i] = elem.Value()
	}
	pe.exportlist <- list
}

//Finish Write outstanding data and wait for completion
func (pe *csvExporter) Finish() {
	close(pe.exportlist)
	<-pe.finished
}

//NewCSVExporter Create a new exporter that just writes the features to filename (- for stdout)
func NewCSVExporter(filename string) flows.Exporter {
	ret := &csvExporter{make(chan []interface{}, 100), make(chan struct{})}
	var outfile io.WriteCloser
	if filename == "-" {
		outfile = os.Stdout
	} else {
		var err error
		outfile, err = os.Create(filename)
		if err != nil {
			log.Fatal("Couldn't open file ", filename, err)
		}
	}
	writer := csv.NewWriter(outfile)
	go func() {
		defer close(ret.finished)
		var record []string
		for data := range ret.exportlist {
			n := len(data)
			if cap(record) < n {
				record = make([]string, n)
			}
			record = record[:n]
			for i, elem := range data {
				record[i] = fmt.Sprint(elem)
			}
			writer.Write(record)
		}
		writer.Flush()
	}()
	return ret
}
