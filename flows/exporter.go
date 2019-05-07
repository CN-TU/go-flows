package flows

import (
	"fmt"
	"sync"

	"github.com/CN-TU/go-flows/util"
)

const debugSort = false
const debugMerge = false
const debugElements = false

const exportQueueDepth = 100
const exportQueueWorkerDepth = 10

type exportQueue chan *exportRecord

type mergeTreeQueues struct {
	head, tail [3]*exportRecord
	result     exportQueue
	order      SortType
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
	var less bool
	if e.order == SortTypeExpiryTime {
		less = e.head[0].lessExpiry(e.head[1])
	} else {
		less = e.head[0].lessPacket(e.head[1])
	}
	if less {
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
			debugList("publish", e.head[2], verbose)
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

func debugList(prefix string, elem *exportRecord, verbose int) {
	nr := 0
	for elem != nil {
		if debugElements {
			fmt.Println(verbose, prefix, "\t", elem.exportKey, "\t", elem.features[0], "\t", elem.features[1], "\t", elem.features[2])
		}
		nr++
		elem = elem.next
	}
	fmt.Println(verbose, "elements moved", nr)
}

// mergeTreeWorker fetches sorted exportRecords from a, b and sends them sorted to result
func mergeTreeWorker(a, b, result exportQueue, order SortType, verbose int) {
	queue := mergeTreeQueues{result: result, order: order}
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
				debugList("gota", elem, verbose)
			}
			queue.append(0, elem)
		case elem, ok := <-b:
			if !ok {
				stopped = 1
				goto PARTIAL
			}
			if debugMerge {
				fmt.Println(verbose, "gotb")
				debugList("gotb", elem, verbose)
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
					debugList("gota", elem, verbose)
				}
				queue.append(0, elem)
			case elem, ok := <-b:
				if !ok {
					stopped = 1
					goto PARTIAL
				}
				if debugMerge {
					fmt.Println(verbose, "gotb")
					debugList("gotb", elem, verbose)
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
			queue.mergeOne()
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
				debugList("gota", elem, verbose)
			}
			queue.append(0, elem)
		case elem, ok := <-b:
			if !ok {
				stopped = 1
				goto PARTIAL
			}
			if debugMerge {
				fmt.Println(verbose, "gotb")
				debugList("gotb", elem, verbose)
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
			elem := queue.head[0]
			if elem == nil {
				elem = queue.head[1]
			}
			queue.append(2, elem)
			queue.publish(verbose)
			goto DONE
		}
		var elem *exportRecord
		elem, ok = <-q
		if ok {
			if debugMerge {
				fmt.Println(verbose, "got", [2]string{"a", "b"}[remaining])
				debugList("got", elem, verbose)
			}
			queue.append(remaining, elem)
		}
	}

FLUSH:
	if queue.hasHead(remaining) {
		if debugMerge {
			fmt.Println(verbose, "publish (FLUSH)")
			debugList("publish", queue.head[remaining], verbose)
		}
		queue.head[remaining].prev = queue.tail[remaining]
		result <- queue.head[remaining]
	}
	queue.clear()
	for elem := range q {
		if debugMerge {
			fmt.Println(verbose, "publish (forward)")
			debugList("publish", elem, verbose)
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

func makeMergeTree(in []exportQueue, order SortType) []exportQueue {
	if len(in) == 2 {
		ret := make(exportQueue, exportQueueWorkerDepth)
		go mergeTreeWorker(in[0], in[1], ret, order, -1)
		return []exportQueue{ret}
	}
	numEven := len(in) / 2
	num := numEven + len(in)%2
	ret := make([]exportQueue, num)
	for i := 0; i < numEven; i++ {
		ret[i] = make(exportQueue, exportQueueWorkerDepth)
		go mergeTreeWorker(in[i*2], in[i*2+1], ret[i], order, i)
	}
	if numEven != num {
		ret[num-1] = in[len(in)-1]
	}
	return makeMergeTree(ret, order)
}

func findNaturalSublists(heads []*exportRecord, head *exportRecord, id int) []*exportRecord {
	if debugSort {
		fmt.Println(id, "START")
	}

	var prev, first *exportRecord
	more := false
	for head != nil {
		if prev == nil {
			if head.next == nil {
				// last element
				heads = append(heads, head)
				if debugSort {
					fmt.Println(id, "LAST", len(heads))
				}
				return heads
			}
			first = head
			prev = head
			head = head.next
			continue
		}

		if !prev.lessExpiry(head) {
			if !more {
				// two unsorted elements
				// add reversed (== sorted)
				newhead := head.next

				head.next = prev
				prev.next = nil
				if debugSort {
					fmt.Println(id, "unsorted")
				}
				heads = append(heads, head)
				first = nil

				prev = nil
				head = newhead
				if debugSort {
					fmt.Println(id, "head:", head)
				}
				continue
			}
			// two or more sorted elements
			if debugSort {
				fmt.Println(id, "sorted")
			}
			heads = append(heads, first)
			first = nil
			prev.next = nil

			more = false
			prev = nil
			continue
		}

		more = true
		prev = head
		head = head.next
	}
	if first != nil {
		heads = append(heads, first)
	}
	if debugSort {
		fmt.Println(id, "Went trough", len(heads))
	}
	return heads
}

func mergeSublist(a, b *exportRecord, last bool, id int) *exportRecord {
	var ret, tail *exportRecord
	for a != nil && b != nil {
		var elem *exportRecord
		if a.lessExpiry(b) {
			elem = a
			a = a.next
		} else {
			elem = b
			b = b.next
		}
		if ret == nil {
			if last {
				elem.prev = elem
			}
			ret = elem
		} else {
			if last {
				elem.prev = nil
			}
			tail.next = elem
		}
		tail = elem
	}

	if a != nil {
		tail.next = a
	}
	if b != nil {
		tail.next = b
	}

	if last {
		for a != nil {
			tail = a
			a.prev = nil
			a = a.next
		}
		for b != nil {
			tail = b
			b.prev = nil
			b = b.next
		}
		ret.prev = tail
	}
	return ret
}

func sortWorker(in, out exportQueue, id int) {
	var sortHeads []*exportRecord
	for elem := range in {
		// find already sorted sublists for merging
		if debugSort {
			fmt.Println(id, "unsorted:")
			debugList("unsorted", elem, id)
			fmt.Println(id, "sort:")
		}
		sortHeads = findNaturalSublists(sortHeads, elem, id)
		if debugSort {
			for i, head := range sortHeads {
				fmt.Println(id, "list", i)
				debugList("sort", head, id)
			}
			fmt.Println(id, "=========")
		}
		if len(sortHeads) == 1 {
			sortHeads[0] = nil
			sortHeads = sortHeads[:0]
			// already sorted
			out <- elem
			if debugSort {
				fmt.Println(id, "already sorted list:")
				debugList("already sorted", elem, id)
			}
			continue
		}

		//merge lists until we have two
		for len(sortHeads) > 2 {
			numEven := len(sortHeads) / 2
			for i := 0; i < numEven; i++ {
				sortHeads[i] = mergeSublist(sortHeads[i*2], sortHeads[i*2+1], false, id)
			}
			if len(sortHeads)%2 == 1 {
				sortHeads[numEven] = sortHeads[len(sortHeads)-1]
				clear := sortHeads[numEven+1 : len(sortHeads)]
				for i := range clear {
					clear[i] = nil
				}
				sortHeads = sortHeads[:numEven+1]
			} else {
				clear := sortHeads[numEven:len(sortHeads)]
				for i := range clear {
					clear[i] = nil
				}
				sortHeads = sortHeads[:numEven]
			}
		}
		// this final merge fixes up the list: *.prev = nil, first.prev = last; last.next = nil
		elem = mergeSublist(sortHeads[0], sortHeads[1], true, id)
		sortHeads[0] = nil
		sortHeads[1] = nil
		sortHeads = sortHeads[:0]
		if debugSort {
			fmt.Println(id, "sorted list:")
			debugList("sorted", elem, id)
		}

		out <- elem
	}
	close(out)
}

func makeSortWorker(in, out exportQueue, id int) {
	go sortWorker(in, out, id)
}

func (e *ExportPipeline) init(fields []string) {
	e.out = make([]exportQueue, len(e.exporter))
	e.finished = &sync.WaitGroup{}
	for i, exporter := range e.exporter {
		exporter.Fields(fields)
		var q exportQueue
		if e.sortOrder == SortTypeNone {
			q = make(exportQueue, exportQueueDepth)
		} else {
			q = make(exportQueue, exportQueueWorkerDepth)
		}
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
		var unmerged []exportQueue
		if e.sortOrder == SortTypeExpiryTime {
			unmerged = make([]exportQueue, e.tables)
			for i := range unmerged {
				unmerged[i] = make(exportQueue, exportQueueWorkerDepth)
				makeSortWorker(e.in[i], unmerged[i], i)
			}
		} else {
			unmerged = e.in
		}
		if e.tables == 1 {
			e.sorted = unmerged[0]
		} else {
			e.sorted = makeMergeTree(unmerged, e.sortOrder)[0]
		}
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
