package ipfix

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/util"
	"github.com/CN-TU/go-ipfix"
)

const pen uint32 = 1234
const tmpBase uint16 = 0x7000

type ipfixExporter struct {
	id         string
	outfile    string
	specfile   string
	exportlist chan ipfixRecord
	finished   chan struct{}
	allocated  map[string]ipfix.InformationElement
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

func (pe *ipfixExporter) writeSpec(w io.Writer) {
	ies := make([]ipfix.InformationElement, len(pe.allocated))
	for _, ie := range pe.allocated {
		ies[ie.ID-tmpBase] = ie
	}
	for _, ie := range ies {
		fmt.Fprintln(w, ie)
	}
}

func (pe *ipfixExporter) ID() string {
	return pe.id
}

var normalizer = strings.NewReplacer("(", "❲", ")", "❳")

func normalizeName(name string) string {
	return normalizer.Replace(name)
}

func (pe *ipfixExporter) AllocateIE(ies []ipfix.InformationElement) []ipfix.InformationElement {
	for i, ie := range ies {
		if ie.ID == 0 && ie.Pen == 0 { //Temporary Element
			if ie, ok := pe.allocated[ie.Name]; ok {
				ies[i] = ie
				continue
			}
			name := ie.Name
			ie = ipfix.InformationElement{
				Name:   normalizeName(name),
				Pen:    pen,
				ID:     uint16(len(pe.allocated)) + tmpBase,
				Type:   ie.Type,
				Length: ie.Length,
			}
			ies[i] = ie
			pe.allocated[name] = ie
		}
	}
	return ies
}

func (pe *ipfixExporter) Init() {
	pe.exportlist = make(chan ipfixRecord, 100)
	pe.finished = make(chan struct{})
	pe.allocated = make(map[string]ipfix.InformationElement)
	var outfile io.WriteCloser
	var specfile io.WriteCloser
	if pe.outfile == "-" {
		outfile = os.Stdout
	} else {
		var err error
		outfile, err = os.Create(pe.outfile)
		if err != nil {
			log.Fatal("Couldn't open file ", pe.outfile, err)
		}
	}
	if pe.specfile == "-" {
		specfile = os.Stdout
	} else if pe.specfile != "" {
		var err error
		specfile, err = os.Create(pe.specfile)
		if err != nil {
			log.Fatal("Couldn't open file ", pe.specfile, err)
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
				template, err = writer.AddTemplate(now, pe.AllocateIE(data.template.InformationElements())...)
				if err != nil {
					log.Panic(err)
				}
				templates[id] = template
			}
			writer.SendData(now, template, data.features...)
		}
		writer.Finalize(now)
		outfile.Close()
		if specfile != nil {
			pe.writeSpec(specfile)
			specfile.Close()
		}
	}()
}

func newIPFIXExporter(args []string) (arguments []string, ret util.Module, err error) {
	set := flag.NewFlagSet("ipfix", flag.ExitOnError)
	set.Usage = func() { ipfixhelp("ipfix") }
	flowSpec := set.String("spec", "", "Flowspec file")

	set.Parse(args)
	if set.NArg() < 1 {
		return nil, nil, errors.New("IPFIX exporter needs a filename as argument")
	}
	outfile := set.Args()[0]
	specfile := *flowSpec
	arguments = set.Args()[1:]

	ipfix.LoadIANASpec()
	ret = &ipfixExporter{id: "IPFIX|" + outfile, outfile: outfile, specfile: specfile}
	return
}

func ipfixhelp(name string) {
	fmt.Fprintf(os.Stderr, `
The %s exporter writes the output to a ipfix file with a flow per line and a
header consisting of the feature description.

As argument, the output file is needed.

Usage:
  export %s [-spec file.iespec] file.ipfix

Flags:
  -spec string
    	Write iespec of temporary ies to file
`, name, name)
}

func init() {
	flows.RegisterExporter("ipfix", "Exports flows to a ipfix file.", newIPFIXExporter, ipfixhelp)
}
