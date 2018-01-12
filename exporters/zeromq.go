package exporters

import (
	"fmt"
	"log"
	"os"

	zmq "github.com/pebbe/zmq4"
	"labix.org/v2/mgo/bson"
	"pm.cn.tuwien.ac.at/ipfix/go-ipfix"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

type zeromqExporter struct {
	id         string
	listen     string
	exportlist chan []byte
	finished   chan struct{}
}

//FIXME: remove
func (pe *zeromqExporter) Fields(fields []string) {
}

//Export export given features
func (pe *zeromqExporter) Export(template flows.Template, features []flows.Feature, when flows.DateTimeNanoseconds) {
	list := make([]interface{}, len(features))
	ies := template.InformationElements()[:len(features)]
	list = list[:len(features)]
	for i, elem := range features {
		switch val := elem.Value().(type) {
		case byte:
			list[i] = int(val)
		case flows.DateTimeNanoseconds:
			switch ies[i].Type {
			case ipfix.DateTimeNanosecondsType:
				list[i] = uint64(val)
			case ipfix.DateTimeMicrosecondsType:
				list[i] = uint64(val / 1e3)
			case ipfix.DateTimeMillisecondsType:
				list[i] = uint64(val / 1e6)
			case ipfix.DateTimeSecondsType:
				list[i] = uint64(val / 1e9)
			default:
				list[i] = val
			}
		case flows.DateTimeMicroseconds:
			switch ies[i].Type {
			case ipfix.DateTimeNanosecondsType:
				list[i] = uint64(val * 1e3)
			case ipfix.DateTimeMicrosecondsType:
				list[i] = uint64(val)
			case ipfix.DateTimeMillisecondsType:
				list[i] = uint64(val / 1e3)
			case ipfix.DateTimeSecondsType:
				list[i] = uint64(val / 1e6)
			default:
				list[i] = val
			}
		case flows.DateTimeMilliseconds:
			switch ies[i].Type {
			case ipfix.DateTimeNanosecondsType:
				list[i] = uint64(val * 1e6)
			case ipfix.DateTimeMicrosecondsType:
				list[i] = uint64(val * 1e3)
			case ipfix.DateTimeMillisecondsType:
				list[i] = uint64(val)
			case ipfix.DateTimeSecondsType:
				list[i] = uint64(val / 1e3)
			default:
				list[i] = val
			}
		case flows.DateTimeSeconds:
			switch ies[i].Type {
			case ipfix.DateTimeNanosecondsType:
				list[i] = uint64(val * 1e9)
			case ipfix.DateTimeMicrosecondsType:
				list[i] = uint64(val * 1e6)
			case ipfix.DateTimeMillisecondsType:
				list[i] = uint64(val * 1e3)
			case ipfix.DateTimeSecondsType:
				list[i] = uint64(val)
			default:
				list[i] = val
			}
		case flows.FlowEndReason:
			list[i] = int(val)
		default:
			list[i] = val
		}
	}
	out, err := bson.Marshal(bson.M{"ts": int(when), "features": list})
	if err != nil {
		log.Println(err)
	}
	pe.exportlist <- out
}

//Finish Write outstanding data and wait for completion
func (pe *zeromqExporter) Finish() {
	close(pe.exportlist)
	<-pe.finished
}

func (pe *zeromqExporter) ID() string {
	return pe.id
}

func (pe *zeromqExporter) Init() {
	pe.exportlist = make(chan []byte, 100)
	pe.finished = make(chan struct{})

	context, err := zmq.NewContext()
	if err != nil {
		panic(err)
	}

	socket, err := context.NewSocket(zmq.PUSH)
	if err != nil {
		panic(err)
	}
	socket.Bind(pe.listen)

	go func() {
		defer close(pe.finished)
		n := 0
		for data := range pe.exportlist {
			_, err := socket.SendBytes(data, 0)
			if err != nil {
				log.Println("Failed to produce message with error ", err)
			}
			n++
		}
		_, err := socket.SendBytes([]byte("END"), 0)
		if err != nil {
			log.Println("Failed to produce message with error ", err)
		}
		socket.Close()
		log.Println(n, "flows exported")
	}()
}

func newZeromqExporter(name string, opts interface{}, args []string) (arguments []string, ret flows.Exporter) {
	var listen string
	if _, ok := opts.(flows.UseStringOption); ok {
		if len(args) > 1 {
			listen = args[0]
			arguments = args[1:]
		}
	} else {
		switch o := opts.(type) {
		case map[string]interface{}:
			if val, ok := o["listen"]; ok {
				listen = val.(string)
			}
		}
	}
	if listen == "" {
		log.Fatalln("ZeroMQ exporter needs a listen address as argument")
	}
	if name == "" {
		name = "ZeroMQ|" + listen
	}
	ret = &zeromqExporter{id: name, listen: listen}
	return
}

func zeromqhelp(name string) {
	fmt.Fprintf(os.Stderr, `
The %s exporter PUSHes the output to ZeroMQ with a flow per message
in BSON format with keys "features" and "ts", in which "features" are the requested
features (in order), and "ts" the timestamp that the flow was exported.

As argument, the ZeroMQ listen address (e.g., "tcp://*:5559") is needed.

Usage command line:
	export %s tcp://*:5559

Usage json file:
  {
    "type": "%s",
    "options": {
	  "listen": "tcp://*:5559",
    }
  }
`, name, name, name)
}

func init() {
	flows.RegisterExporter("zeromq", "Exports flows to a ZeroMQ subscriber and topic.", newZeromqExporter, zeromqhelp)
}
