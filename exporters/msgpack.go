package exporters

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/ugorji/go/codec"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

type msgPack struct {
	id         string
	outfile    string
	exportlist chan []interface{}
	finished   chan struct{}
}

//FIXME: remove
func (pe *msgPack) Fields(fields []string) {
	list := make([]interface{}, len(fields))
	list = list[:len(fields)]
	for i, elem := range fields {
		list[i] = elem
	}
	pe.exportlist <- list
}

//Export export given features
func (pe *msgPack) Export(template flows.Template, features []flows.Feature, when flows.DateTimeNanoseconds) {
	list := make([]interface{}, len(features))
	list = list[:len(features)]
	for i, elem := range features {
		switch val := elem.Value().(type) {
		case net.IP:
			list[i] = []byte(val)
		case byte:
			list[i] = int(val)
		case flows.DateTimeNanoseconds:
			list[i] = int64(val)
		case flows.FlowEndReason:
			list[i] = int(val)
		default:
			list[i] = val
		}
	}
	pe.exportlist <- list
}

//Finish Write outstanding data and wait for completion
func (pe *msgPack) Finish() {
	close(pe.exportlist)
	<-pe.finished
}

func (pe *msgPack) ID() string {
	return pe.id
}

func (pe *msgPack) Init() {
	pe.exportlist = make(chan []interface{}, 100)
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
	buf := bufio.NewWriter(outfile)
	var mh codec.MsgpackHandle
	enc := codec.NewEncoder(buf, &mh)
	go func() {
		defer close(pe.finished)
		for data := range pe.exportlist {
			enc.MustEncode(data)
		}
		buf.Flush()
		outfile.Close()
	}()
}

func newMsgPack(name string, opts interface{}, args []string) (arguments []string, ret flows.Exporter) {
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
		log.Fatalln("msgpack exporter needs a filename as argument")
	}
	if name == "" {
		name = "MSGPACK|" + outfile
	}
	ret = &msgPack{id: name, outfile: outfile}
	return
}

func msgpackhelp(name string) {
	fmt.Fprintf(os.Stderr, `
The %s exporter writes the output to a file with one msgpack per flow
and a header in msgpack format consisting of the feature description.

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
	flows.RegisterExporter("msgpack", "Exports flows to a msgpack file.", newMsgPack, msgpackhelp)
}
