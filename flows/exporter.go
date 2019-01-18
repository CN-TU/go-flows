package flows

import (
	"errors"
	"sync"

	"github.com/CN-TU/go-flows/util"
)

const exportQueueDepth = 100

type exportQueue chan *exportRecord

// ExportPipeline contains all functionality for merging results from multiple tables and exporting
type ExportPipeline struct {
	exporter  []Exporter
	out       []exportQueue
	sorted    exportQueue
	finished  *sync.WaitGroup
	sortOrder SortType
	tables    int
}

// MakeExportPipeline creates an ExportPipeline for a list of Exporters
func MakeExportPipeline(exporter []Exporter, sortOrder SortType, numTables uint) (*ExportPipeline, error) {
	if sortOrder != SortTypeNone && numTables%2 != 0 {
		return nil, errors.New("sorting requires an even number of processing tables")
	}
	return &ExportPipeline{exporter: exporter, sortOrder: sortOrder, tables: int(numTables)}, nil
}

func exporterWorker(q exportQueue, exporter Exporter, wg *sync.WaitGroup) {
	for r := range q {
		e := r
		for e != nil {
			exporter.Export(e.template, e.features, e.exportTime)
			e = e.next
		}
	}
	wg.Done()
}

func (e *ExportPipeline) exportSplicer(q exportQueue) {
	for r := range q {
		for _, out := range e.out {
			out <- r
		}
	}
	for _, out := range e.out {
		close(out)
	}
}

func (e *ExportPipeline) init(fields []string) {
	e.sorted = make(exportQueue, exportQueueDepth)
	e.out = make([]exportQueue, len(e.exporter))
	e.finished = &sync.WaitGroup{}
	for i, exporter := range e.exporter {
		exporter.Fields(fields)
		q := make(exportQueue, exportQueueDepth)
		e.out[i] = q
		e.finished.Add(1)
		go exporterWorker(q, exporter, e.finished)
	}
	// TODO: add merge-sorting here
	go e.exportSplicer(e.sorted)
}

func (e *ExportPipeline) export(record *exportRecord, tableid int) {
	if e.sortOrder == SortTypeNone {
		e.sorted <- record
		return
	}
	// FIXME merge-sorting
}

func (e *ExportPipeline) shutdown() {
	close(e.sorted)
}

// wait waits for all outstanding events to be finished
func (e *ExportPipeline) wait() {
	e.finished.Wait()
}

// Flush shuts the pipline down and waits for it to finish
func (e *ExportPipeline) Flush() {
	e.shutdown()
	e.wait()
}

const exporterName = "exporter"

// Exporter represents a generic exporter
type Exporter interface {
	util.Module
	// Export gets called upon record export with a list of features and the export time.
	Export(Template, []interface{}, DateTimeNanoseconds)
	// Fields gets called during flow-exporter initialization with the list of fieldnames as argument
	Fields([]string)
	// Finish gets called before program exit. Eventual flushing needs to be implemented here.
	Finish()
}

// RegisterExporter registers an exporter (see module system in util)
func RegisterExporter(name, desc string, new util.ModuleCreator, help util.ModuleHelp) {
	util.RegisterModule(exporterName, name, desc, new, help)
}

// ExporterHelp displays help for a specific exporter (see module system in util)
func ExporterHelp(which string) error {
	return util.GetModuleHelp(exporterName, which)
}

// MakeExporter creates an exporter instance (see module system in util)
func MakeExporter(which string, args []string) ([]string, Exporter, error) {
	args, module, err := util.CreateModule(exporterName, which, args)
	if err != nil {
		return args, nil, err
	}
	return args, module.(Exporter), nil
}

// ListExporters returns a list of exporters (see module system in util)
func ListExporters() ([]util.ModuleDescription, error) {
	return util.GetModules(exporterName)
}
