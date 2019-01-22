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
	id        string
	outfile   string
	specfile  string
	out       io.WriteCloser
	spec      io.WriteCloser
	writer    *ipfix.MessageStream
	allocated map[string]ipfix.InformationElement
	templates []int
	now       flows.DateTimeNanoseconds
}

func (pe *ipfixExporter) Fields([]string) {}

//Export export given features
func (pe *ipfixExporter) Export(template flows.Template, features []interface{}, when flows.DateTimeNanoseconds) {
	id := template.ID()
	if id >= len(pe.templates) {
		pe.templates = append(pe.templates, make([]int, id-len(pe.templates)+1)...)
	}
	templateID := pe.templates[id]
	if templateID == 0 {
		var err error
		templateID, err = pe.writer.AddTemplate(when, pe.AllocateIE(template.InformationElements())...)
		if err != nil {
			log.Panic(err)
		}
		pe.templates[id] = templateID
	}
	//TODO make templates for nil features
	pe.writer.SendData(when, templateID, features...)
	pe.now = when
}

//Finish Write outstanding data and wait for completion
func (pe *ipfixExporter) Finish() {
	pe.writer.Flush(pe.now)
	if pe.out != os.Stdout {
		pe.out.Close()
	}
	if pe.spec != nil {
		pe.writeSpec(pe.spec)
		if pe.spec != os.Stdout {
			pe.spec.Close()
		}
	}
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
	pe.allocated = make(map[string]ipfix.InformationElement)
	var err error
	if pe.outfile == "-" {
		pe.out = os.Stdout
	} else {
		pe.out, err = os.Create(pe.outfile)
		if err != nil {
			log.Fatal("Couldn't open file ", pe.outfile, err)
		}
	}
	if pe.specfile == "-" {
		pe.spec = os.Stdout
	} else if pe.specfile != "" {
		pe.spec, err = os.Create(pe.specfile)
		if err != nil {
			log.Fatal("Couldn't open file ", pe.specfile, err)
		}
	}
	pe.writer, err = ipfix.MakeMessageStream(pe.out, 65535, 0)
	if err != nil {
		log.Fatal("Couldn't create ipfix message stream: ", err)
	}
	pe.templates = make([]int, 1)
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
