package flows

import (
	"fmt"
	"io"
	"log"
	"strings"
	"text/template"
)

// Record holds multiple features that belong to a single record
type Record interface {
	// Destroy must be called before a record is removed to clean up unexported exportlists
	Destroy()
	// Event gets called for every event
	Event(Event, *EventContext, *FlowTable, int)
	// Export exports this record
	Export(FlowEndReason, *EventContext, DateTimeNanoseconds, *FlowTable, int)
	// Returns true if this record is still active
	Active() bool
}

type control struct {
	control []int
	event   []int
	export  []int
	variant []int
}

type record struct {
	features []Feature
	filter   []Feature
	control  *control
	export   *exportRecord
	last     DateTimeNanoseconds
	active   bool // this record forwards events to features
	alive    bool // this record forwards events to filters
}

func (r *record) Destroy() {
	if r.export != nil {
		r.export.unlink()
	}
}

func (r *record) start(data Event, context *EventContext, table *FlowTable, recordID int) {
	r.active = true
	context.record = r
	for _, feature := range r.features {
		feature.Start(context)
	}
	for _, feature := range r.control.event {
		r.features[feature].FinishEvent(context) //Same for finishevents
	}
	if table.SortOutput == SortTypeNone {
		return
	}
	if r.export != nil {
		// clean up stopped + unexported leftovers
		r.export.unlink()
	}
	r.export = &exportRecord{
		exportKey: exportKey{
			packetID: data.EventNr(),
			recordID: recordID,
		},
	}
	table.pushExport(recordID, r.export)
}

func (r *record) stop(reason FlowEndReason, context *EventContext, recordID int) {
	// make a two-level stop for filter and rest
	context.record = r
	for _, feature := range r.features {
		feature.Stop(reason, context)
	}
	r.active = false
	r.alive = false
	for _, feature := range r.filter {
		feature.Stop(reason, context)
		r.alive = (context.keep || r.alive) && !context.hard
	}
	for _, feature := range r.control.event {
		r.features[feature].FinishEvent(context) //Same for finishevents
	}
}

func (r *record) filteredEvent(data Event, context *EventContext, table *FlowTable, recordID int) {
RESTART:
	if !r.active {
		r.start(data, context, table, recordID)
	}
	context.clear()
	for _, feature := range r.control.control {
		r.features[feature].Event(data, context, nil) // no tree for control
		if context.stop {
			r.stop(context.reason, context, recordID)
			goto OUT
		}
		if context.now {
			if context.export {
				tmp := *context
				tmp.when = r.last
				r.Export(context.reason, &tmp, context.when, table, recordID)
				goto RESTART
			}
			if context.restart {
				r.start(data, context, table, recordID)
				goto RESTART
			}
		}
	}
	for _, feature := range r.control.event {
		r.features[feature].Event(data, context, nil) //Events trickle down the tree
	}
	for _, feature := range r.control.event {
		r.features[feature].FinishEvent(context) //Same for finishevents
	}
	if table.SortOutput == SortTypeStopTime || table.SortOutput == SortTypeExpiryTime {
		r.export.packetID = data.EventNr()
		table.pushExport(recordID, r.export)
	}
	if !context.now {
		if context.export {
			r.Export(context.reason, context, context.when, table, recordID)
		}
		if context.restart {
			r.start(data, context, table, recordID)
		}
	}
OUT:
	r.last = context.when
}

func (r *record) Event(data Event, context *EventContext, table *FlowTable, recordID int) {
	nfilter := len(r.filter)
	context.record = r
	if nfilter == 0 {
		r.filteredEvent(data, context, table, recordID)
	} else {
		if !r.alive {
			for _, feature := range r.filter {
				feature.Start(context)
			}
			r.alive = true
		}
		filter := r.filter
		context.event = func(data interface{}, context *EventContext, pos interface{}) {
			i := pos.(int)
			if i == nfilter {
				r.filteredEvent(data.(Event), context, table, recordID)
				return
			}
			filter[i].Event(data, context, i+1)
		}
		context.event(data, context, 0)
	}
}

func (r *record) Export(reason FlowEndReason, context *EventContext, now DateTimeNanoseconds, table *FlowTable, recordID int) {
	if !r.active {
		return
	}
	context.record = r
	r.stop(reason, context, recordID)

	record := table.records.list[recordID]
	template := record.template
	for _, variant := range r.control.variant {
		template = template.subTemplate(r.features[variant].Variant())
	}
	export := make([]interface{}, len(r.control.export))
	for i := range export {
		export[i] = r.features[r.control.export[i]].Value()
	}

	if table.SortOutput == SortTypeNone {
		record.export.export(&exportRecord{
			exportTime: now,
			template:   template,
			features:   export,
		}, int(table.id))
	} else {
		r.export.expiryTime = context.When()
		r.export.exportTime = now
		r.export.template = template
		r.export.features = export
		if table.SortOutput == SortTypeExpiryTime {
			table.pushExport(recordID, r.export)
		}
		r.export = nil
	}
}

func (r *record) Active() bool {
	return r.active || r.alive
}

type recordList []*record

func (r recordList) Destroy() {
	for _, record := range r {
		record.Destroy()
	}
}

func (r recordList) Event(data Event, context *EventContext, table *FlowTable, recordID int) {
	for i, record := range r {
		record.Event(data, context, table, i)
	}
}

func (r recordList) Export(reason FlowEndReason, context *EventContext, now DateTimeNanoseconds, table *FlowTable, recordID int) {
	for i, record := range r {
		record.Export(reason, context, now, table, i)
	}
}

func (r recordList) Active() bool {
	for _, record := range r {
		if record.Active() {
			return true
		}
	}
	return false
}

// RecordListMaker holds metadata for instantiating a list of records with included features
type RecordListMaker struct {
	list      []RecordMaker
	templates int
}

func (rl RecordListMaker) make() Record {
	l := len(rl.list)
	if l == 1 {
		return rl.list[0].make()
	}
	ret := make(recordList, len(rl.list))
	ret = ret[:len(rl.list)]
	for i, record := range rl.list {
		ret[i] = record.make()
	}
	return ret
}

// Init must be called after instantiating a record list
func (rl RecordListMaker) Init() {
	for _, record := range rl.list {
		record.Init()
	}
}

// Clean execution graph, which is not needed for execution
func (rl RecordListMaker) Clean() {
	for _, record := range rl.list {
		record.ast = nil
	}
}

// Flush flushes all outstanding exports and waits for them to finish. Must be called before exporters can be shut down
func (rl RecordListMaker) Flush() {
	for _, record := range rl.list {
		record.export.shutdown()
	}
	for _, record := range rl.list {
		record.export.wait()
	}
}

// RecordMaker holds metadata for instantiating a record
type RecordMaker struct {
	export   *ExportPipeline
	template Template
	fields   []string
	ast      *ast
	make     func() *record
}

// Init must be called after a Record was instantiated
func (rm *RecordMaker) Init() {
	rm.export.init(rm.fields)
}

// AppendRecord creates a internal representation needed for instantiating records from a feature
// specification, a list of exporters and a needed base (only FlowFeature supported so far)
func (rl *RecordListMaker) AppendRecord(features []interface{}, control, filter []string, exporter *ExportPipeline, verbose bool) error {
	tree, err := makeAST(features, control, filter, exporter.exporter, RawPacket, FlowFeature) // only packets -> flows for now
	if err != nil {
		return err
	}
	if err := tree.compile(verbose); err != nil {
		return err
	}

	template, fields := tree.template(&rl.templates)
	if verbose {
		log.Println("Fields: ", strings.Join(fields, ", "))
		log.Println("Template(s): ", template)
	}

	featureMakers, filterMakers, args, tocall, ctrl := tree.convert()

	hasfilter := len(filterMakers) > 0

	rl.list = append(rl.list, RecordMaker{
		exporter,
		template,
		fields,
		tree,
		func() *record {
			features := make([]Feature, len(featureMakers))
			features = features[:len(featureMakers)] //BCE
			for i, maker := range featureMakers {
				features[i] = maker()
			}
			for i, arg := range args {
				if len(arg) > 0 {
					if f, ok := features[i].(FeatureWithArguments); ok {
						f.SetArguments(arg, features)
					}
				}
			}
			for i, tocall := range tocall {
				features[i].setDependent(tocall)
			}
			var filter []Feature
			if hasfilter {
				filter = make([]Feature, len(filterMakers))
				filter = filter[:len(filterMakers)] //BCE
				for i, feature := range filterMakers {
					filter[i] = feature()
				}
			}

			return &record{
				features: features,
				filter:   filter,
				control:  ctrl,
			}

		},
	})
	return nil
}

var graphTemplate = template.Must(template.New("callgraph").Parse(`digraph callgraph {
	label="call graph"
	node [shape=box, gradientangle=90]
	"source" [style="rounded,filled", fillcolor=red]
	{{ range $index, $element := .Nodes }}
	subgraph cluster_{{$index}} {
	{{ range $element.Nodes }}	"{{.Name}}" [label={{if .Label}}"{{.Label}}"{{else}}<{{.HTML}}>{{end}}{{range .Style}}, {{index . 0}}="{{index . 1}}"{{end}}]
	{{end}}	"export{{$index}}" [label="export",style="rounded,filled", fillcolor=red]
	}
	{{ range $element.Export }}"{{.Name}}" [label={{if .Label}}"{{.Label}}"{{else}}<{{.HTML}}>{{end}}{{range .Style}}, {{index . 0}}="{{index . 1}}"{{end}}]
	"export{{$index}}" -> "{{.Name}}"
	{{end}}
	{{end}}
	{{ range .Edges }}"{{.Start}}"{{if .StartNode}}:{{.StartNode}}{{end}} -> "{{.Stop}}"{{if .StopNode}}:{{.StopNode}}{{end}}
	{{end}}
}
`))

// CallGraph generates a call graph in the graphviz language and writes the result to w.
func (rl RecordListMaker) CallGraph(w io.Writer) {
	styles := map[FeatureType][][]string{
		FlowFeature: {
			{"shape", "invhouse"},
			{"style", "filled"},
		},
		PacketFeature: {
			{"style", "filled"},
		},
		Const: {
			{"shape", "oval"},
		},
		RawPacket: {
			{"shape", "invtrapezium"},
		},
	}
	type Node struct {
		Name  string
		Label string
		HTML  string
		Style [][]string
	}
	type Edge struct {
		Start     string
		StartNode string
		Stop      string
		StopNode  string
	}
	type Subgraph struct {
		Nodes  []Node
		Export []Node
	}
	data := struct {
		Nodes []Subgraph
		Edges []Edge
	}{}

	for listID, fl := range rl.list {
		var nodes []Node
		export := make([]Node, len(fl.export.exporter))
		for i, exporter := range fl.export.exporter {
			export[i] = Node{Name: fmt.Sprintf("%p", exporter), Label: exporter.ID()}
		}

		raw := fmt.Sprintf("%draw", listID)

		nodes = append(nodes, Node{
			Name:  raw,
			Label: "RawPacket",
		})

		data.Edges = append(data.Edges, Edge{
			Start: "source",
			Stop:  raw,
		})

		for i, filter := range fl.ast.filter {
			f := fmt.Sprintf("%d,fi%d", listID, i)
			nodes = append(nodes, Node{
				Name:  f,
				Label: filter,
				Style: styles[RawPacket],
			})
			data.Edges = append(data.Edges, Edge{
				Start: raw,
				Stop:  f,
			})
			raw = f
		}

		for _, fragment := range fl.ast.fragments {
			var node Node
			var html func() string
			if fragment.Control() {
				node.Name = fmt.Sprintf("%d,c%d", listID, fragment.Register())
				node.Label = fragment.Name()
				node.Style = [][]string{{"shape", "doubleoctagon"}}

				data.Edges = append(data.Edges, Edge{
					Start: raw,
					Stop:  node.Name,
				})
			} else {
				node.Name = fmt.Sprintf("%d,f%d", listID, fragment.Register())
				if fragment.Data() != nil {
					node.Label = fmt.Sprint(fragment.Data())
					node.Style = styles[Const]
				} else {
					node.Label = fragment.Name()
					args := fragment.Arguments()
					if len(args) == 1 {
						if args[0].IsRaw() {
							node.Style = append(styles[fragment.Returns()], []string{"fillcolor", "green"})
							data.Edges = append(data.Edges, Edge{
								Start: raw,
								Stop:  node.Name,
							})
						} else {
							composite := fragment.Composite()
							if composite == "" {
								node.Style = append(styles[fragment.Returns()], []string{"fillcolor", "orange"})
							} else {
								node.Style = append(styles[fragment.Returns()], []string{"fillcolor", "green:orange"})
								node.Label = fmt.Sprintf("%s\n%s", node.Label, composite)
							}
							data.Edges = append(data.Edges, Edge{
								Start: fmt.Sprintf("%d,f%d", listID, args[0].Register()),
								Stop:  node.Name,
							})
						}
					} else if len(args) > 1 {
						node.Style = append([][]string{}, styles[fragment.Returns()]...)
						if node.Label != "select" && node.Label != "select_slice" {
							composite := fragment.Composite()
							if composite == "" {
								node.Style = append(node.Style, []string{"fillcolor", "orange"})
							} else {
								node.Style = append(node.Style, []string{"fillcolor", "green:orange"})
								node.Label = fmt.Sprintf("%s\n%s", node.Label, composite)
							}
						}
						stringArgs := make([]string, len(args))
						for i := range stringArgs {
							stringArgs[i] = fmt.Sprintf(`<TD PORT="%d" BORDER="1">%d</TD>`, i, i)
							start := raw
							if !args[i].IsRaw() {
								start = fmt.Sprintf("%d,f%d", listID, args[i].Register())
							}
							data.Edges = append(data.Edges, Edge{
								Start:    start,
								Stop:     node.Name,
								StopNode: fmt.Sprint(i),
							})
						}
						html = func() string {
							return fmt.Sprintf(`<TABLE BORDER="0" CELLBORDER="0" CELLSPACING="2"><TR>%s</TR><TR><TD COLSPAN="%d">%s</TD></TR></TABLE>`, strings.Join(stringArgs, ""), len(args), strings.Replace(node.Label, "\\n", "<BR/>", -1))
						}
					}
				}
				if fragment.Export() {
					if node.Label != fragment.ExportName() && fragment.Composite() != fragment.ExportName() {
						node.Label = fmt.Sprintf("%s\\n%s", node.Label, fragment.ExportName())
					}
					data.Edges = append(data.Edges, Edge{
						Start: node.Name,
						Stop:  fmt.Sprintf("export%d", listID),
					})
				}
				if html != nil {
					node.HTML = html()
					node.Label = ""
				}
			}

			nodes = append(nodes, node)
		}
		data.Nodes = append(data.Nodes, Subgraph{nodes, export})
	}
	graphTemplate.Execute(w, data)
}
