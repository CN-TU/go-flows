package csv

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/CN-TU/go-flows/packet"
	"github.com/CN-TU/go-flows/util"
)

type csvLabels struct {
	id       string
	labels   []string
	file     io.Closer
	csv      *csv.Reader
	nextData interface{}
	nextPos  uint64
}

func (cl *csvLabels) ID() string {
	return cl.id
}

func (cl *csvLabels) Init() {
}

func (cl *csvLabels) open() {
	if cl.csv != nil {
		cl.close()
	}
	if len(cl.labels) == 0 {
		return
	}
	var f string
	f, cl.labels = cl.labels[0], cl.labels[1:]
	r, err := os.Open(f)
	if err != nil {
		panic(err)
	}
	cl.file = r
	cl.csv = csv.NewReader(r)
	_, err = cl.csv.Read() // Read title line
	if err != nil {
		panic(err)
	}
}

func (cl *csvLabels) close() {
	if cl.csv != nil {
		cl.csv = nil
		cl.file.Close()
	}
}

func (cl *csvLabels) GetLabel(packet packet.Buffer) (interface{}, error) {
	packetnr := packet.PacketNr()
	if cl.nextPos == packetnr {
		return cl.nextData, nil
	}
	if cl.nextPos > packetnr {
		return nil, nil
	}
	if cl.csv == nil {
		if len(cl.labels) == 0 {
			return nil, io.EOF
		}
		cl.open()
	}
	record, err := cl.csv.Read()
	if err == io.EOF {
		cl.open()
		if cl.csv == nil {
			return nil, io.EOF
		}
		record, err = cl.csv.Read()
	}
	if record == nil && err != nil {
		panic(err)
	}
	if len(record) == 1 {
		return record, nil
	}
	cl.nextPos, err = strconv.ParseUint(record[0], 10, 64)
	if err != nil {
		panic(err)
	}
	if cl.nextPos <= 0 {
		panic("Label packet position must be >= 0")
	}
	if cl.nextPos == packetnr {
		return record[1:], nil
	}
	cl.nextData = record[1:]
	return nil, nil
}

func newcsvLabels(name string, opts interface{}, args []string) (arguments []string, ret util.Module, err error) {
	var files []string

	arguments = args

	if _, ok := opts.(util.UseStringOption); ok {
		for len(arguments) > 0 {
			if arguments[0] == "--" {
				arguments = arguments[1:]
				break
			}
			files = append(files, arguments[0])
			arguments = arguments[1:]
		}
	} else {
		panic("FIXME: implement this")
	}

	if len(files) == 0 {
		return nil, nil, errors.New("csv labels needs at least one input file")
	}

	if name == "" {
		name = fmt.Sprint("csvlabel|", strings.Join(files, ";"))
	}
	ret = &csvLabels{
		id:     name,
		labels: files,
	}
	return
}

func csvLabelsHelp(name string) {
	fmt.Fprintf(os.Stderr, `
The %s label source reads labels from one or more csv files. If further
commands need to be provided, then "--" can be used to stop the file list.

The csv files must start with a header. If there is only one column in the
csv file, every row is matched up with a packet. If there are at least two
columns, the first column must be the packet number this label belongs to.
The number of the first packet is 1!

Usage command line:
  label %s a.csv [b.csv] [..] [--]

Usage json file (not working):
  {
    "type": "%s",
    "options": "file.csv"
  }
`, name, name, name)
}

func init() {
	packet.RegisterLabel("csv", "Read labels from a csv file.", newcsvLabels, csvLabelsHelp)
}
