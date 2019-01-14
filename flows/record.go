package flows

import (
	"io"
	"log"
	"strings"
	"text/template"
)

// Record holds multiple features that belong to a single record
type Record interface {
	// Start gets called after flow initialisation
	Start(*EventContext)
	// Event gets called for every event
	Event(interface{}, *EventContext)
	// Stop gets called before export
	Stop(FlowEndReason, *EventContext)
	// Export exports this record
	Export(FlowEndReason, *EventContext, DateTimeNanoseconds)
	// Returns true if this record is still active
	Active() bool
}

type control struct {
	control  []int
	event    []int
	export   []int
	variant  []int
	exporter []Exporter
	template Template
}

type record struct {
	features []Feature
	filter   []Feature
	control  *control
	last     DateTimeNanoseconds
	active   bool // this record forwards events to features
	alive    bool // this record forwards events to filters
}

func (r *record) Start(context *EventContext) {
	r.active = true
	context.record = r
	for _, feature := range r.features {
		feature.Start(context)
	}
	for _, feature := range r.control.event {
		r.features[feature].FinishEvent(context) //Same for finishevents
	}
}

func (r *record) Stop(reason FlowEndReason, context *EventContext) {
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

func (r *record) filteredEvent(data interface{}, context *EventContext) {
RESTART:
	if !r.active {
		r.Start(context)
	}
	context.clear()
	for _, feature := range r.control.control {
		r.features[feature].Event(data, context, nil) // no tree for control
		if context.stop {
			r.Stop(context.reason, context)
			goto OUT
		}
		if context.now {
			if context.export {
				tmp := *context
				tmp.when = r.last
				r.Export(context.reason, &tmp, context.when)
				goto RESTART
			}
			if context.restart {
				r.Start(context)
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
	if !context.now {
		if context.export {
			r.Export(context.reason, context, context.when)
		}
		if context.restart {
			r.Start(context)
		}
	}
OUT:
	r.last = context.when
}

func (r *record) Event(data interface{}, context *EventContext) {
	nfilter := len(r.filter)
	context.record = r
	if nfilter == 0 {
		r.filteredEvent(data, context)
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
				r.filteredEvent(data, context)
				return
			}
			filter[i].Event(data, context, i+1)
		}
		context.event(data, context, 0)
	}
}

func (r *record) Export(reason FlowEndReason, context *EventContext, now DateTimeNanoseconds) {
	if !r.active {
		return
	}
	context.record = r
	r.Stop(reason, context)
	template := r.control.template
	for _, variant := range r.control.variant {
		template = template.subTemplate(r.features[variant].Variant())
	}
	for _, exporter := range r.control.exporter {
		export := make([]Feature, len(r.control.export))
		for i := range export {
			export[i] = r.features[r.control.export[i]]
		}
		exporter.Export(template, export, now)
	}
}

func (r *record) Active() bool {
	return r.active || r.alive
}

type recordList []*record

func (r recordList) Start(context *EventContext) {
	for _, record := range r {
		record.Start(context)
	}
}

func (r recordList) Stop(reason FlowEndReason, context *EventContext) {
	for _, record := range r {
		record.Stop(reason, context)
	}
}

func (r recordList) Event(data interface{}, context *EventContext) {
	for _, record := range r {
		record.Event(data, context)
	}
}

func (r recordList) Export(reason FlowEndReason, context *EventContext, now DateTimeNanoseconds) {
	for _, record := range r {
		record.Export(reason, context, now)
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

// RecordMaker holds metadata for instantiating a record
type RecordMaker struct {
	exporter []Exporter
	fields   []string
	make     func() *record
}

// Init must be called after a Record was instantiated
func (rm RecordMaker) Init() {
	for _, exporter := range rm.exporter {
		exporter.Fields(rm.fields)
	}
}

// AppendRecord creates a internal representation needed for instantiating records from a feature
// specification, a list of exporters and a needed base (only FlowFeature supported so far)
func (rl *RecordListMaker) AppendRecord(features []interface{}, control, filter []string, exporter []Exporter, verbose bool) error {
	tree, err := makeAST(features, control, filter, exporter, RawPacket, FlowFeature) // only packets -> flows for now
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
	ctrl.exporter = exporter
	ctrl.template = template

	hasfilter := len(filterMakers) > 0

	rl.list = append(rl.list, RecordMaker{
		exporter,
		fields,
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
	"event" [style="rounded,filled", fillcolor=red]
	{{ range $index, $element := .Nodes }}
	subgraph cluster_{{$index}} {
	{{ range $element.Nodes }}	"{{.Name}}" [label={{if .Label}}"{{.Label}}"{{else}}<{{.HTML}}>{{end}}{{range .Style}}, {{index . 0}}="{{index . 1}}"{{end}}]
	{{end}}}
	"export{{$index}}" [label="export",style="rounded,filled", fillcolor=red]
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
	panic("not implemented")
	/*
		FIXME
		toID := func(id, i int) string {
			return fmt.Sprintf("%d,%d", id, i)
		}
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
			export := make([]Node, len(fl.exporter))
			for i, exporter := range fl.exporter {
				export[i] = Node{Name: fmt.Sprintf("%p", exporter), Label: exporter.ID()} //FIXME: better style
			}
			for i, feature := range fl.init {
				var node Node
				node.Label = feature.name
				if node.Label != feature.feature.ie.Name { //FIXME
					node.Label = fmt.Sprintf("%s\\n%s", feature.feature.ie.Name, node.Label)
				}
				if feature.info.apply != "" {
					node.Label = fmt.Sprintf("%s\\n%s", feature.info.apply, node.Label)
				}
				node.Name = toID(listID, i)
				if feature.info.data == nil {
					if len(feature.arguments) == 0 {
						node.Style = append(styles[feature.info.ret], []string{"fillcolor", "green"})
					} else if len(feature.arguments) == 1 {
						if feature.info.nonComposite {
							node.Style = append(styles[feature.info.ret], []string{"fillcolor", "orange"})
						} else {
							node.Style = append(styles[feature.info.ret], []string{"fillcolor", "green:orange"})
						}
					} else {
						node.Style = append(styles[feature.info.ret], []string{"fillcolor", "orange"})
						args := make([]string, len(feature.arguments))
						for i := range args {
							args[i] = fmt.Sprintf(`<TD PORT="%d" BORDER="1">%d</TD>`, i, i)
						}
						node.HTML = fmt.Sprintf(`<TABLE BORDER="0" CELLBORDER="0" CELLSPACING="2"><TR>%s</TR><TR><TD COLSPAN="%d">%s</TD></TR></TABLE>`, strings.Join(args, ""), len(feature.arguments), strings.Replace(node.Label, "\\n", "<BR/>", -1))
						node.Label = ""
					}
				} else {
					node.Label = fmt.Sprint(feature.info.data)
					node.Style = styles[Const]
				}

				nodes = append(nodes, node)
			}
			for i, feature := range fl.init {
				if feature.event {
					data.Edges = append(data.Edges, Edge{"event", "", toID(listID, i), ""})
				}
				for index, j := range feature.arguments {
					index := strconv.Itoa(index)
					if len(feature.arguments) <= 1 {
						index = ""
					}
					data.Edges = append(data.Edges, Edge{toID(listID, j), "", toID(listID, i), index})
				}
			}
			for _, i := range fl.exportList {
				data.Edges = append(data.Edges, Edge{toID(listID, i), "", fmt.Sprintf("export%d", listID), ""})
			}
			data.Nodes = append(data.Nodes, Subgraph{nodes, export})
		}
		graphTemplate.Execute(w, data)
	*/
}
