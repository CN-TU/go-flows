package exporters

import (
	"bufio"
	"io"
	"log"
	"net"
	"os"

	"github.com/ugorji/go/codec"
	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

type msgPack struct {
	exportlist chan []interface{}
	finished   chan struct{}
}

func (pe *msgPack) Fields(fields []string) {
	n := len(fields)
	var list = make([]interface{}, n)
	for i, elem := range fields {
		list[i] = elem
	}
	pe.exportlist <- list
}

//Export export given features
func (pe *msgPack) Export(features []flows.Feature, when flows.Time) {
	n := len(features)
	var list = make([]interface{}, n)
	for i, elem := range features {
		switch val := elem.Value().(type) {
		case flows.Number:
			list[i] = val.GoValue()
		case net.IP:
			list[i] = []byte(val)
		case byte:
			list[i] = int(val)
		case flows.Time:
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

//NewMsgPack Create a new exporter that just writes the features to filename (- for stdout)
func NewMsgPack(filename string) flows.Exporter {
	ret := &msgPack{make(chan []interface{}, 100), make(chan struct{})}
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
	buf := bufio.NewWriter(outfile)
	var mh codec.MsgpackHandle
	enc := codec.NewEncoder(buf, &mh)
	go func() {
		defer close(ret.finished)
		for data := range ret.exportlist {
			enc.MustEncode(data)
		}
		buf.Flush()
		outfile.Close()
	}()
	return ret
}
