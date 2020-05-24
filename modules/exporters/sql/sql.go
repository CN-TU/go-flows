package sql

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"strings"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/util"
	ipfix "github.com/CN-TU/go-ipfix"
)

// Produces a file with SQL statements that create and fill a table with flow data.
// May support multiple SQL engines.

const writeBufferSize = 64 * 1024

type typeTranslationTable map[ipfix.Type]string

type sqlExporter struct {
	id      string
	outfile string
	f       io.WriteCloser
	writer  *bufio.Writer

	fields    []string
	fieldsStr string
	seqNo     int

	ttrans        typeTranslationTable
	headerWritten bool
	tableName     string
}

func (se *sqlExporter) Fields(fields []string) {
	se.fields = make([]string, len(fields))
	copy(se.fields, fields)

	se.fieldsStr = "`_seqNo`"
	for _, name := range fields {
		se.fieldsStr += fmt.Sprintf(",`%s`", name)
	}
}

func (se *sqlExporter) ExportCreate(ies []ipfix.InformationElement, features []interface{}) error {
	if se.headerWritten {
		return nil
	}

	se.headerWritten = true

	_, err := fmt.Fprintln(se.writer, "CREATE TABLE ", se.tableName, " (_seqNo INT NOT NULL PRIMARY KEY")
	if err != nil {
		return err
	}

	for i := range features {
		_, err := fmt.Fprintf(se.writer, ",\n  `%s` %s DEFAULT NULL", se.fields[i], se.ttrans[ies[i].Type])
		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprintln(se.writer, "  );")
	return err
}

func (se *sqlExporter) valueToString(elem interface{}, t ipfix.Type) string {
	switch val := elem.(type) {
	case nil:
		return "NULL"
	case bool:
		if val {
			return "Y"
		}

		return "N"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, flows.FlowEndReason:
		return fmt.Sprintf("%d", val)
	case float32:
		// workaround for NaN values in integer-typed fields.
		if t != ipfix.Float32Type {
			//log.Println("found float32 (value = ", val, ") but type is ", t)
			return "NULL"
		}

		if math.IsNaN(float64(val)) {
			return "\"NaN\""
		}
		return fmt.Sprintf("%g", val)
	case float64:
		// workaround for NaN values in integer-typed fields.
		if t != ipfix.Float64Type {
			//log.Println("found float64 (value = ", val, ") but type is ", t)
			return "NULL"
		}

		if math.IsNaN(val) {
			return "\"NaN\""
		}
		return fmt.Sprintf("%g", val)
	case net.IP:
		return fmt.Sprintf("\"%s\"", val.String())
	case flows.DateTimeNanoseconds:
		switch t {
		case ipfix.DateTimeMicrosecondsType:
			val /= 1e3
		case ipfix.DateTimeMillisecondsType:
			val /= 1e6
		case ipfix.DateTimeSecondsType:
			val /= 1e9
		}

		return fmt.Sprintf("%d", val)
	case flows.DateTimeMicroseconds:
		switch t {
		case ipfix.DateTimeNanosecondsType:
			val *= 1e3
		case ipfix.DateTimeMillisecondsType:
			val /= 1e3
		case ipfix.DateTimeSecondsType:
			val /= 1e6
		}

		return fmt.Sprintf("%d", val)
	case flows.DateTimeMilliseconds:
		switch t {
		case ipfix.DateTimeNanosecondsType:
			val *= 1e6
		case ipfix.DateTimeMicrosecondsType:
			val *= 1e3
		case ipfix.DateTimeSecondsType:
			val /= 1e3
		}

		return fmt.Sprintf("%d", val)
	case flows.DateTimeSeconds:
		switch t {
		case ipfix.DateTimeNanosecondsType:
			val *= 1e9
		case ipfix.DateTimeMicrosecondsType:
			val *= 1e6
		case ipfix.DateTimeMillisecondsType:
			val *= 1e3
		}

		return fmt.Sprintf("%d", val)
	case []byte:
		return fmt.Sprintf("\"%s\"", string(val))
	case string:
		return fmt.Sprintf("\"%s\"", val)
	case net.HardwareAddr:
		return fmt.Sprintf("\"%s\"", val.String())
	default:
		return fmt.Sprintf("\"%v\"", val)
	}
}

//Export export given features
func (se *sqlExporter) Export(template flows.Template, features []interface{}, when flows.DateTimeNanoseconds) {
	ies := template.InformationElements()[:len(features)]
	values := make([]string, len(se.fields))

	if err := se.ExportCreate(ies, features); err != nil {
		panic(err)
	}

	for i := 0; i != len(values); i++ {
		if i < len(features) {
			values[i] = se.valueToString(features[i], ies[i].Type)
			continue
		}

		values[i] = "NULL"
	}

	se.seqNo++
	_, err := fmt.Fprintln(se.writer, "INSERT INTO ", se.tableName, "(", se.fieldsStr, ") VALUES (", se.seqNo, ",", strings.Join(values, ","), ");")
	if err != nil {
		panic(err)
	}
}

// Finish writes outstanding data and waits for completion.
func (se *sqlExporter) Finish() {
	if err := se.writer.Flush(); err != nil {
		panic(err)
	}

	if se.f != os.Stdout {
		se.f.Close()
	}
}

// ID returns the ID of the exporter instance.
func (se *sqlExporter) ID() string {
	return se.id
}

// Init performs initialization, such as creating the output file.
func (se *sqlExporter) Init() {
	if se.outfile == "-" {
		se.f = os.Stdout
	} else {
		var err error
		se.f, err = os.Create(se.outfile)
		if err != nil {
			log.Fatal("Couldn't open file ", se.outfile, err)
		}
	}
	se.writer = bufio.NewWriterSize(se.f, writeBufferSize)
}

func newSQLExporter(args []string) (arguments []string, ret *sqlExporter, err error) {
	set := flag.NewFlagSet("sql", flag.ExitOnError)
	set.Usage = func() { sqlhelp("sql") }

	table := set.String("table", "data", "Name of the table")
	set.Parse(args)

	arguments = set.Args()

	if len(arguments) < 1 {
		return nil, nil, errors.New("SQL exporter needs a filename as argument")
	}
	outfile := arguments[0]
	arguments = arguments[1:]

	ret = &sqlExporter{id: "SQL|" + outfile, outfile: outfile, tableName: *table}
	return
}

func newMySQLExporter(args []string) (arguments []string, ret util.Module, err error) {
	arguments, se, err := newSQLExporter(args)
	if err != nil {
		return
	}

	se.ttrans = mySQLTypesTable()
	return arguments, se, nil
}

func newPostgreSQLExporter(args []string) (arguments []string, ret util.Module, err error) {
	arguments, se, err := newSQLExporter(args)
	if err != nil {
		return
	}

	se.ttrans = postgreSQLTypesTable()
	return arguments, se, nil
}

func sqlhelp(name string) {
	fmt.Fprintf(os.Stderr, `
The %s exporter writes the output to an sql file with a flow per row and a
CREATE TABLE statement.

As argument, the output file is needed.

Usage:
  export %s file.csv

Flags:
  -table <table name>
    Name the table that is going to appear in the CREATE statement (default: "data")
`, name, name)
}

func init() {
	flows.RegisterExporter("mysql", "Exports flows to an sql file (MySQL syntax).", newMySQLExporter, sqlhelp)
	flows.RegisterExporter("postgresql", "Exports flows to an sql file (PostgreSQL syntax).", newPostgreSQLExporter, sqlhelp)
}
