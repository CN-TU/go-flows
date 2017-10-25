package exporters

import (
	"fmt"
	"io"
	"log"
	"os"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
	"pm.cn.tuwien.ac.at/ipfix/go-ipfix"
)

type ipfixExporter struct {
	id         string
	outfile    string
	exportlist chan ipfixRecord
	finished   chan struct{}
}

type ipfixRecord struct {
	template flows.Template
	features []interface{}
	when     flows.DateTimeNanoseconds
}

func (pe *ipfixExporter) Fields([]string) {}

//Export export given features
func (pe *ipfixExporter) Export(template flows.Template, features []flows.Feature, when flows.DateTimeNanoseconds) {
	list := make([]interface{}, len(features))
	list = list[:len(features)]
	for i, elem := range features {
		list[i] = elem.Value()
	}
	pe.exportlist <- ipfixRecord{template, list, when}
}

//Finish Write outstanding data and wait for completion
func (pe *ipfixExporter) Finish() {
	close(pe.exportlist)
	<-pe.finished
}

func (pe *ipfixExporter) ID() string {
	return pe.id
}

func (pe *ipfixExporter) Init() {
	pe.exportlist = make(chan ipfixRecord, 100)
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
	writer := ipfix.MakeMessageStream(outfile, 65535, 0)
	go func() {
		defer close(pe.finished)
		templates := make([]int, 1)
		var now flows.DateTimeNanoseconds
		var err error
		for data := range pe.exportlist {
			id := data.template.ID()
			now = data.when
			if id >= len(templates) {
				templates = append(templates, make([]int, id-len(templates)+1)...)
			}
			template := templates[id]
			if template == 0 {
				template, err = writer.AddTemplate(now, data.template.InformationElements()...)
				if err != nil {
					log.Panic(err)
				}
				templates[id] = template
			}
			writer.SendData(now, template, data.features...)
		}
		writer.Finalize(now)
		outfile.Close()
	}()
}

func newIPFIXExporter(name string, opts interface{}, args []string) (arguments []string, ret flows.Exporter) {
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
		log.Fatalln("IPFIX exporter needs a filename as argument")
	}
	if name == "" {
		name = "IPFIX|" + outfile
	}
	ipfix.LoadIANASpec()
	ret = &ipfixExporter{id: name, outfile: outfile}
	return
}

func ipfixhelp(name string) {
	fmt.Fprintf(os.Stderr, `
The %s exporter writes the output to a ipfix file with a flow per line and a
header consisting of the feature description.

As argument, the output file is needed.

Usage command line:
  export %s file.ipfix

Usage json file:
  {
    "type": "%s",
    "options": "file.ipfix"
  }
`, name, name, name)
}

func init() {
	flows.RegisterExporter("ipfix", "Exports flows to a ipfix file.", newIPFIXExporter, ipfixhelp)
}
