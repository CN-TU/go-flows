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
	subscriber string
	topic      string
	exportlist chan []byte
	finished   chan struct{}
	context    *zmq.Context
	publisher  *zmq.Socket
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
	pe.context.Term()
}

func (pe *zeromqExporter) ID() string {
	return pe.id
}

func (pe *zeromqExporter) Init() {
	pe.exportlist = make(chan []byte, 100)
	pe.finished = make(chan struct{})

	context, _ := zmq.NewContext()
	pe.context = context

	publisher, _ := context.NewSocket(zmq.PUB)
	publisher.Connect("tcp://" + pe.subscriber)

	go func() {
		defer close(pe.finished)
		defer publisher.Close()
		for data := range pe.exportlist {
			_, err := publisher.SendMessage([][]byte{[]byte(pe.topic), data})
			if err != nil {
				log.Println("Failed to produce message with error ", err)
			}
		}
	}()
}

func newZeromqExporter(name string, opts interface{}, args []string) (arguments []string, ret flows.Exporter) {
	var subscriber string
	var topic string
	if _, ok := opts.(flows.UseStringOption); ok {
		if len(args) > 1 {
			subscriber = args[0]
			topic = args[1]
			arguments = args[2:]
		}
	} else {
		switch o := opts.(type) {
		case []interface{}:
			if len(o) != 2 {
				log.Fatalln("ZeroMQ needs at least subscriber address and topic in list specification")
			}
			subscriber = o[0].(string)
			topic = o[1].(string)
		case map[string]interface{}:
			if val, ok := o["subscriber"]; ok {
				subscriber = val.(string)
			}
			if val, ok := o["topic"]; ok {
				topic = val.(string)
			}
		}
	}
	if subscriber == "" {
		log.Fatalln("ZeroMQ exporter needs a subscriber address as argument")
	}
	if topic == "" {
		log.Fatalln("ZeroMQ exporter needs a topic as argument")
	}
	if name == "" {
		name = "ZeroMQ|" + subscriber + "|" + topic
	}
	ret = &zeromqExporter{id: name, subscriber: subscriber, topic: topic}
	return
}

func zeromqhelp(name string) {
	fmt.Fprintf(os.Stderr, `
The %s exporter writes the output to a ZeroMQ topic with a flow per message,
in BSON format, with keys "features" and "ts", in which "features" are the requested
features (in order), and "ts" the timestamp that the flow was exported.

As argument, the ZeroMQ subscriber address (e.g., "localhost:5559"), the Zookeeper address
(e.g., "localhost:2181"), and a topic name to which the producer will write are needed.

Usage command line:
	export %s subscriber:5559  topic_name

Usage json file:
  {
    "type": "%s",
	"options": "subscriber:5559 zookeeper:2181 topic_name"
  }

  {
    "type": "%s",
	"options": ["subscriber:5559", "zookeeper:2181", "topic_name"]
  }

  {
    "type": "%s",
    "options": {
	  "subscriber": "subscriber:9092",
	  "zookeeper": "zookeeper:2181",
	  "topic": "topic_name"
    }
  }
`, name, name, name, name, name)
}

func init() {
	flows.RegisterExporter("zeromq", "Exports flows to a ZeroMQ subscriber and topic.", newZeromqExporter, zeromqhelp)
}
