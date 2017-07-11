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

//NewCSVExporter Create a new exporter that just writes the features to filename (- for stdout)
func NewCSVExporter(filename string) flows.Exporter {
	ret := &csvExporter{make(chan []string, 100), make(chan struct{})}
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
		for data := range ret.exportlist {
			writer.Write(data)
		}
		writer.Flush()
	}()
	return ret
}
