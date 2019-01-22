package kafka

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
	id       string
	kafka    string
	topic    string
	producer sarama.AsyncProducer
}

func (pe *kafkaExporter) Fields(fields []string) {
}

//Export export given features
func (pe *kafkaExporter) Export(template flows.Template, features []interface{}, when flows.DateTimeNanoseconds) {
	ies := template.InformationElements()[:len(features)]
	for i, elem := range features {
		switch val := elem.(type) {
		case byte:
			features[i] = int(val)
		case flows.DateTimeNanoseconds:
			switch ies[i].Type {
			case ipfix.DateTimeNanosecondsType:
				features[i] = uint64(val)
			case ipfix.DateTimeMicrosecondsType:
				features[i] = uint64(val / 1e3)
			case ipfix.DateTimeMillisecondsType:
				features[i] = uint64(val / 1e6)
			case ipfix.DateTimeSecondsType:
				features[i] = uint64(val / 1e9)
			default:
				features[i] = val
			}
		case flows.DateTimeMicroseconds:
			switch ies[i].Type {
			case ipfix.DateTimeNanosecondsType:
				features[i] = uint64(val * 1e3)
			case ipfix.DateTimeMicrosecondsType:
				features[i] = uint64(val)
			case ipfix.DateTimeMillisecondsType:
				features[i] = uint64(val / 1e3)
			case ipfix.DateTimeSecondsType:
				features[i] = uint64(val / 1e6)
			default:
				features[i] = val
			}
		case flows.DateTimeMilliseconds:
			switch ies[i].Type {
			case ipfix.DateTimeNanosecondsType:
				features[i] = uint64(val * 1e6)
			case ipfix.DateTimeMicrosecondsType:
				features[i] = uint64(val * 1e3)
			case ipfix.DateTimeMillisecondsType:
				features[i] = uint64(val)
			case ipfix.DateTimeSecondsType:
				features[i] = uint64(val / 1e3)
			default:
				features[i] = val
			}
		case flows.DateTimeSeconds:
			switch ies[i].Type {
			case ipfix.DateTimeNanosecondsType:
				features[i] = uint64(val * 1e9)
			case ipfix.DateTimeMicrosecondsType:
				features[i] = uint64(val * 1e6)
			case ipfix.DateTimeMillisecondsType:
				features[i] = uint64(val * 1e3)
			case ipfix.DateTimeSecondsType:
				features[i] = uint64(val)
			default:
				features[i] = val
			}
		case flows.FlowEndReason:
			features[i] = int(val)
		default:
			features[i] = val
		}
	}
	out, err := bson.Marshal(bson.M{"ts": int(when), "features": features})
	if err != nil {
		fmt.Println(err)
	}
	pe.producer.Input() <- &sarama.ProducerMessage{Topic: pe.topic, Key: nil, Value: sarama.ByteEncoder(out)}
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

func newKafkaExporter(args []string) (arguments []string, ret util.Module, err error) {
	if len(args) < 2 {
		return nil, nil, errors.New("Kafka exporter needs a kafka address and a topic name as argument")
	}

	kafka := args[0]
	topic := args[1]
	arguments = args[2:]

	ret = &kafkaExporter{id: "Kafka|" + topic, kafka: kafka, topic: topic}
	return
}

func kafkahelp(name string) {
	fmt.Fprintf(os.Stderr, `
The %s exporter writes the output to a Kafka topic with a flow per message,
in BSON format, with keys "features" and "ts", in which "features" are the requested
features (in order), and "ts" the timestamp that the flow was exported.

As argument, the Kafka address (e.g., "localhost:9092"), and a topic name to
which the producer will write are needed.

Usage:
	export %s kafka:9092 topic_name
`, name, name)
}

func init() {
	flows.RegisterExporter("kafka", "Exports flows to a kafka topic.", newKafkaExporter, kafkahelp)
}
