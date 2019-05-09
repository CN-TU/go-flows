package csv

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/util"
	ipfix "github.com/CN-TU/go-ipfix"
)

// produces RFC4180 conforming csv (except for the line ending, which is LF instead of CRLF)
// this does not use encoding/csv since this results in a huge performance drop due to first writing to strings and then checking those for commas, which is unnessecary for all the numbers/ips

const writeBufferSize = 64 * 1024

type csvExporter struct {
	id      string
	outfile string
	f       io.WriteCloser
	writer  *bufio.Writer
	flush   bool
}

func (pe *csvExporter) writeString(field string) {
	if field == "" {
		return
	}
	if !strings.ContainsAny(field, "\"\r\n,") {
		r1, _ := utf8.DecodeRuneInString(field)
		if unicode.IsSpace(r1) {
			err := pe.writer.WriteByte('"')
			if err != nil {
				panic(err)
			}
			_, err = pe.writer.WriteString(field)
			if err != nil {
				panic(err)
			}
			err = pe.writer.WriteByte('"')
			if err != nil {
				panic(err)
			}
			return
		}
		_, err := pe.writer.WriteString(field)
		if err != nil {
			panic(err)
		}
		return
	}
	err := pe.writer.WriteByte('"')
	if err != nil {
		panic(err)
	}
	for len(field) > 0 {
		special := strings.IndexAny(field, "\"")
		if special == -1 {
			_, err := pe.writer.WriteString(field)
			if err != nil {
				panic(err)
			}
			break
		}
		_, err := pe.writer.WriteString(field[:special])
		if err != nil {
			panic(err)
		}
		_, err = pe.writer.WriteString("\"\"")
		if err != nil {
			panic(err)
		}
		field = field[special+1:]
	}
	err = pe.writer.WriteByte('"')
	if err != nil {
		panic(err)
	}
}

func (pe *csvExporter) Fields(fields []string) {
	for i, field := range fields {
		if i > 0 {
			err := pe.writer.WriteByte(',')
			if err != nil {
				panic(err)
			}
		}
		pe.writeString(field)
	}
	err := pe.writer.WriteByte('\n')
	if err != nil {
		panic(err)
	}
	if pe.flush {
		err = pe.writer.Flush()
		if err != nil {
			panic(err)
		}
	}
}

//Export export given features
func (pe *csvExporter) Export(template flows.Template, features []interface{}, when flows.DateTimeNanoseconds) {
	ies := template.InformationElements()[:len(features)]
	for i, elem := range features {
		var err error
		if i > 0 {
			err = pe.writer.WriteByte(',')
			if err != nil {
				panic(err)
			}
		}
		switch val := elem.(type) {
		case int:
			_, err = pe.writer.WriteString(strconv.FormatInt(int64(val), 10))
		case int8:
			_, err = pe.writer.WriteString(strconv.FormatInt(int64(val), 10))
		case int16:
			_, err = pe.writer.WriteString(strconv.FormatInt(int64(val), 10))
		case int32:
			_, err = pe.writer.WriteString(strconv.FormatInt(int64(val), 10))
		case int64:
			_, err = pe.writer.WriteString(strconv.FormatInt(val, 10))
		case uint:
			_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val), 10))
		case uint8:
			_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val), 10))
		case uint16:
			_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val), 10))
		case uint32:
			_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val), 10))
		case uint64:
			_, err = pe.writer.WriteString(strconv.FormatUint(val, 10))
		case float32:
			_, err = pe.writer.WriteString(strconv.FormatFloat(float64(val), 'g', -1, 32))
		case float64:
			_, err = pe.writer.WriteString(strconv.FormatFloat(val, 'g', -1, 64))
		case net.IP:
			_, err = pe.writer.WriteString(val.String())
		case nil:
			continue
		case flows.DateTimeNanoseconds:
			switch ies[i].Type {
			case ipfix.DateTimeNanosecondsType:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val), 10))
			case ipfix.DateTimeMicrosecondsType:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val/1e3), 10))
			case ipfix.DateTimeMillisecondsType:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val/1e6), 10))
			case ipfix.DateTimeSecondsType:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val/1e9), 10))
			default:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val), 10))
			}
		case flows.DateTimeMicroseconds:
			switch ies[i].Type {
			case ipfix.DateTimeNanosecondsType:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val*1e3), 10))
			case ipfix.DateTimeMicrosecondsType:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val), 10))
			case ipfix.DateTimeMillisecondsType:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val/1e3), 10))
			case ipfix.DateTimeSecondsType:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val/1e6), 10))
			default:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val), 10))
			}
		case flows.DateTimeMilliseconds:
			switch ies[i].Type {
			case ipfix.DateTimeNanosecondsType:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val*1e6), 10))
			case ipfix.DateTimeMicrosecondsType:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val*1e3), 10))
			case ipfix.DateTimeMillisecondsType:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val), 10))
			case ipfix.DateTimeSecondsType:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val/1e3), 10))
			default:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val), 10))
			}
		case flows.DateTimeSeconds:
			switch ies[i].Type {
			case ipfix.DateTimeNanosecondsType:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val*1e9), 10))
			case ipfix.DateTimeMicrosecondsType:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val*1e6), 10))
			case ipfix.DateTimeMillisecondsType:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val*1e3), 10))
			case ipfix.DateTimeSecondsType:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val), 10))
			default:
				_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val), 10))
			}
		case flows.FlowEndReason:
			_, err = pe.writer.WriteString(strconv.FormatUint(uint64(val), 10))
		case []byte:
			pe.writeString(string(val))
		case string:
			pe.writeString(val)
		case net.HardwareAddr:
			_, err = pe.writer.WriteString(val.String())
		default:
			pe.writeString(fmt.Sprint(val))
		}
		if err != nil {
			panic(err)
		}
	}
	err := pe.writer.WriteByte('\n')
	if err != nil {
		panic(err)
	}
	if pe.flush {
		err = pe.writer.Flush()
		if err != nil {
			panic(err)
		}
	}
}

//Finish Write outstanding data and wait for completion
func (pe *csvExporter) Finish() {
	pe.writer.Flush()
	if pe.f != os.Stdout {
		pe.f.Close()
	}
}

func (pe *csvExporter) ID() string {
	return pe.id
}

func (pe *csvExporter) Init() {
	if pe.outfile == "-" {
		pe.f = os.Stdout
	} else {
		var err error
		pe.f, err = os.Create(pe.outfile)
		if err != nil {
			log.Fatal("Couldn't open file ", pe.outfile, err)
		}
	}
	pe.writer = bufio.NewWriterSize(pe.f, writeBufferSize)
}

func newCSVExporter(args []string) (arguments []string, ret util.Module, err error) {
	set := flag.NewFlagSet("csv", flag.ExitOnError)
	set.Usage = func() { csvhelp("csv") }

	flush := set.Bool("flush", false, "Flush after each line")

	set.Parse(args)

	arguments = set.Args()

	if len(arguments) < 1 {
		return nil, nil, errors.New("CSV exporter needs a filename as argument")
	}
	outfile := arguments[0]
	arguments = arguments[1:]

	ret = &csvExporter{id: "CSV|" + outfile, outfile: outfile, flush: *flush}
	return
}

func csvhelp(name string) {
	fmt.Fprintf(os.Stderr, `
The %s exporter writes the output to a csv file with a flow per line and a
header consisting of the feature description.

As argument, the output file is needed.

Usage:
  export %s file.csv

Flags:
-flush
	  Flush after each line (default off).
`, name, name)
}

func init() {
	flows.RegisterExporter("csv", "Exports flows to a csv file.", newCSVExporter, csvhelp)
}
