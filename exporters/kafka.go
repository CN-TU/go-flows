package exporters

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/CN-TU/go-ipfix"
	"github.com/Shopify/sarama"
	"labix.org/v2/mgo/bson"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/util"
)

type kafkaExporter struct {
	id        string
	kafka     string
	zookeeper string
	topic     string
	producer  sarama.AsyncProducer
}

//FIXME: remove
func (pe *kafkaExporter) Fields(fields []string) {
}

//Export export given features
func (pe *kafkaExporter) Export(template flows.Template, features []flows.Feature, when flows.DateTimeNanoseconds) {
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
		fmt.Println(err)
	}
	pe.producer.Input() <- &sarama.ProducerMessage{Topic: pe.topic, Key: nil, Value: sarama.ByteEncoder(out)}
	//pe.exportlist <- bson.Marshal(&Flow{ts: int(when), features: list})
}

//Finish Write outstanding data and wait for completion
func (pe *kafkaExporter) Finish() {
	pe.producer.Close()
}

func (pe *kafkaExporter) ID() string {
	return pe.id
}

func (pe *kafkaExporter) Init() {
	producer, err := sarama.NewAsyncProducer([]string{pe.kafka}, nil)
	if err != nil {
		log.Fatal("Couldn't open connect to Kafka at ", pe.kafka, ". Error message: ", err)
	}
	pe.producer = producer
	go func() {
		for err := range producer.Errors() {
			log.Println("Failed to produce message with error ", err)
		}
	}()
}

func newKafkaExporter(name string, opts interface{}, args []string) (arguments []string, ret util.Module, err error) {
	var kafka string
	var zookeeper string
	var topic string
	if _, ok := opts.(util.UseStringOption); ok {
		if len(args) > 2 {
			kafka = args[0]
			zookeeper = args[1]
			topic = args[2]
			arguments = args[3:]
		}
	} else {
		switch o := opts.(type) {
		case []interface{}:
			if len(o) != 3 {
				return nil, nil, errors.New("Kafka needs at least kafka address, zookeeper address, and topic in list specification")
			}
			kafka = o[0].(string)
			zookeeper = o[1].(string)
			topic = o[2].(string)
		case map[string]interface{}:
			if val, ok := o["kafka"]; ok {
				kafka = val.(string)
			}
			if val, ok := o["zookeeper"]; ok {
				zookeeper = val.(string)
			}
			if val, ok := o["topic"]; ok {
				topic = val.(string)
			}
		}
	}
	if kafka == "" {
		return nil, nil, errors.New("Kafka exporter needs a kafka address as argument")
	}
	if zookeeper == "" {
		return nil, nil, errors.New("Kafka exporter needs a zookeeper address as argument")
	} // TODO: Do we really need zookeeper address?
	if topic == "" {
		return nil, nil, errors.New("Kafka exporter needs a topic as argument")
	}
	if name == "" {
		name = "Kafka|" + zookeeper + "|" + topic
	}
	ret = &kafkaExporter{id: name, kafka: kafka, zookeeper: zookeeper, topic: topic}
	return
}

func kafkahelp(name string) {
	fmt.Fprintf(os.Stderr, `
The %s exporter writes the output to a Kafka topic with a flow per message,
in BSON format, with keys "features" and "ts", in which "features" are the requested
features (in order), and "ts" the timestamp that the flow was exported.

As argument, the Kafka address (e.g., "localhost:9092"), the Zookeeper address
(e.g., "localhost:2181"), and a topic name to which the producer will write are needed.

Usage command line:
	export %s kafka:9092 zookeeper:2181 topic_name

Usage json file:
  {
    "type": "%s",
	"options": "kafka:9092 zookeeper:2181 topic_name"
  }

  {
    "type": "%s",
	"options": ["kafka:9092", "zookeeper:2181", "topic_name"]
  }

  {
    "type": "%s",
    "options": {
	  "kafka": "kafka:9092",
	  "zookeeper": "zookeeper:2181",
	  "topic": "topic_name"
    }
  }
`, name, name, name, name, name)
}

func init() {
	flows.RegisterExporter("kafka", "Exports flows to a kafka topic.", newKafkaExporter, kafkahelp)
}
