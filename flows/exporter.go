package flows

import (
	"errors"
	"fmt"
	"sync"

	"github.com/CN-TU/go-flows/util"
)

const debugMerge = false
const debugElements = false

const exportQueueDepth = 100

type exportQueue chan *exportRecord

// next on last element nil
// prev on first element point to last

type mergeTreeQueues struct {
	head, tail [3]*exportRecord
	result     exportQueue
}

func (e *mergeTreeQueues) hasHead(which int) bool {
	return e.head[which] != nil
}

func (e *mergeTreeQueues) hasBothHeads() bool {
	return e.head[0] != nil && e.head[1] != nil
}

func (e *mergeTreeQueues) bothHaveTwoElements() bool {
	return e.head[0].next != nil && e.head[1].next != nil
}

func (e *mergeTreeQueues) hasTwoElements(which int) bool {
	return e.head[which] != nil && e.head[which].next != nil
}

func (e *mergeTreeQueues) append(which int, elem *exportRecord) {
	if e.tail[which] == nil {
		e.head[which] = elem
		e.tail[which] = elem.prev
		return
	}
	e.tail[which].next = elem
	e.tail[which] = elem.prev
}

func (e *mergeTreeQueues) mergeOne() {
	var elem *exportRecord
	if e.head[0].less(e.head[1]) {
		elem = e.head[0]
		e.head[0] = e.head[0].next
	} else {
		elem = e.head[1]
		e.head[1] = e.head[1].next
	}
	elem.next = nil
	if e.head[2] == nil {
		elem.prev = elem
		e.head[2] = elem
	} else {
		elem.prev = nil
		e.tail[2].next = elem
		e.head[2].prev = elem
	}
	e.tail[2] = elem
}

func (e *mergeTreeQueues) publish(verbose int) {
	if e.head[2] != nil {
		if debugMerge {
			fmt.Println(verbose, "publish")
			debugList(e.head[2], verbose)
		}
		e.result <- e.head[2]
		e.head[2] = nil
		e.tail[2] = nil
	}
}

func (e *mergeTreeQueues) clear() {
	for i := range e.head {
		e.head[i] = nil
	}
	for i := range e.tail {
		e.tail[i] = nil
	}
}

func debugList(elem *exportRecord, verbose int) {
	nr := 0
	for elem != nil {
		if debugElements {
			fmt.Println(verbose, "\t", elem.exportKey, "\t", elem.features[0], "\t", elem.features[1])
		}
		nr++
		elem = elem.next
	}
	fmt.Println(verbose, "elements moved", nr)
}

// mergeTreeWorker fetches sorted exportRecords from a, b and sends them sorted to result
func mergeTreeWorker(a, b, result exportQueue, verbose int) {
	queue := mergeTreeQueues{result: result}
	var stopped int
	// get initial elements for a and b
	for !queue.hasBothHeads() {
		select {
		case elem, ok := <-a:
			if !ok {
				stopped = 0
				goto PARTIAL
			}
			if debugMerge {
				fmt.Println(verbose, "gota")
				debugList(elem, verbose)
			}
			queue.append(0, elem)
		case elem, ok := <-b:
			if !ok {
				stopped = 1
				goto PARTIAL
			}
			if debugMerge {
				fmt.Println(verbose, "gotb")
				debugList(elem, verbose)
			}
			queue.append(1, elem)
		}
	}

	if debugMerge {
		fmt.Println(verbose, "got initial")
	}

	for {
		// fetch possibly buffered elements
	POLL:
		for {
			select {
			case elem, ok := <-a:
				if !ok {
					stopped = 0
					goto PARTIAL
				}
				if debugMerge {
					fmt.Println(verbose, "gota")
					debugList(elem, verbose)
				}
				queue.append(0, elem)
			case elem, ok := <-b:
				if !ok {
					stopped = 1
					goto PARTIAL
				}
				if debugMerge {
					fmt.Println(verbose, "gotb")
					debugList(elem, verbose)
				}
				queue.append(1, elem)
			default:
				if debugMerge {
					fmt.Println(verbose, "nothing to poll")
				}
				break POLL
			}
		}

		// merge the two list until we have at least one element in both queues left
		for queue.bothHaveTwoElements() {
			queue.mergeOne() //FIXME limit maximum list size?
		}

		if debugMerge {
			fmt.Println(verbose, "merged")
		}

		// send the merged list to the next level
		queue.publish(verbose)

		// fetch next element
		select {
		case elem, ok := <-a:
			if !ok {
				stopped = 0
				goto PARTIAL
			}
			if debugMerge {
				fmt.Println(verbose, "gota")
				debugList(elem, verbose)
			}
			queue.append(0, elem)
		case elem, ok := <-b:
			if !ok {
				stopped = 1
				goto PARTIAL
			}
			if debugMerge {
				fmt.Println(verbose, "gotb")
				debugList(elem, verbose)
			}
			queue.append(1, elem)
		}
	}

PARTIAL:
	var q exportQueue
	var remaining int
	if stopped == 0 {
		q = b
		remaining = 1
	} else {
		q = a
		remaining = 0
	}

	if debugMerge {
		fmt.Println(verbose, "partial", [2]string{"a", "b"}[stopped])
	}

	ok := true

	for {
		// merge remaing
		for queue.hasHead(stopped) && queue.hasTwoElements(remaining) {
			queue.mergeOne()
		}
		queue.publish(verbose)
		if !queue.hasHead(stopped) {
			goto FLUSH
		}
		if debugMerge {
			fmt.Println(verbose, "partial", ok)
		}
		if !ok {
			// if we are here, we have one element in stopped and 2 in remaining
			if debugMerge {
				fmt.Println(verbose, "publish (!ok)", queue)
			}
			for queue.hasBothHeads() {
				queue.mergeOne()
			}
			// now only one should have elements
			for i := range []int{1, 2} {
				elem := queue.head[i]
				for elem != nil {
					next := elem.next
					elem.prev = nil
					queue.tail[2].next = elem
					queue.tail[2] = elem
					queue.head[2].prev = elem
					elem = next
				}
			}
			queue.publish(verbose)
			goto DONE
		}
		var elem *exportRecord
		elem, ok = <-q
		if ok {
			if debugMerge {
				fmt.Println(verbose, "got", [2]string{"a", "b"}[remaining])
				debugList(elem, verbose)
			}
			queue.append(remaining, elem)
		}
	}

FLUSH:
	if queue.hasHead(remaining) {
		if debugMerge {
			fmt.Println(verbose, "publish (FLUSH)")
			debugList(queue.head[remaining], verbose)
		}
		queue.head[remaining].prev = queue.tail[remaining]
		result <- queue.head[remaining]
	}
	queue.clear()
	for elem := range q {
		if debugMerge {
			fmt.Println(verbose, "publish (forward)")
			debugList(elem, verbose)
		}
		result <- elem
	}

DONE:
	if debugMerge {
		fmt.Println(verbose, "close")
	}
	close(result)
}

// ExportPipeline contains all functionality for merging results from multiple tables and exporting
type ExportPipeline struct {
	exporter  []Exporter
	out       []exportQueue
	in        []exportQueue
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

func makeMergeTree(in []exportQueue) []exportQueue {
	if len(in) == 2 {
		ret := make(exportQueue, exportQueueDepth)
		go mergeTreeWorker(in[0], in[1], ret, -1)
		return []exportQueue{ret}
	}
	num := len(in) / 2
	ret := make([]exportQueue, num)
	for i := 0; i < num; i++ {
		ret[i] = make(exportQueue, exportQueueDepth)
		go mergeTreeWorker(in[i*2], in[i*2+1], ret[i], i)
	}
	return makeMergeTree(ret)
}

func (e *ExportPipeline) init(fields []string) {
	e.out = make([]exportQueue, len(e.exporter))
	e.finished = &sync.WaitGroup{}
	for i, exporter := range e.exporter {
		exporter.Fields(fields)
		q := make(exportQueue, exportQueueDepth)
		e.out[i] = q
		e.finished.Add(1)
		go exporterWorker(q, exporter, e.finished)
	}
	if e.sortOrder == SortTypeNone {
		e.sorted = make(exportQueue, exportQueueDepth)
	} else {
		e.in = make([]exportQueue, e.tables)
		for i := range e.in {
			e.in[i] = make(exportQueue, exportQueueDepth)
		}
		e.sorted = makeMergeTree(e.in)[0]
	}
	go e.exportSplicer(e.sorted)
}

func (e *ExportPipeline) export(record *exportRecord, tableid int) {
	if e.sortOrder == SortTypeNone {
		e.sorted <- record
		return
	}
	e.in[tableid] <- record
}

func (e *ExportPipeline) shutdown() {
	if e.sortOrder == SortTypeNone {
		close(e.sorted)
		return
	}
	for _, queue := range e.in {
		close(queue)
	}
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
