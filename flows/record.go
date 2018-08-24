package flows

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/template"

	"github.com/CN-TU/go-ipfix"
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

type record struct {
	filter   []Feature
	control  []Feature
	event    []Feature
	export   []Feature
	startup  []Feature
	exporter []Exporter
	variant  []Feature
	template Template
	last     DateTimeNanoseconds
	active   bool // this record forwards events to features
	alive    bool // this record forwards events to filters
}

func (r *record) Start(context *EventContext) {
	r.active = true
	for _, feature := range r.startup {
		feature.Start(context)
	}
	for _, feature := range r.event {
		feature.FinishEvent() //Same for finishevents
	}
}

func (r *record) Stop(reason FlowEndReason, context *EventContext) {
	// make a two-level stop for filter and rest
	for _, feature := range r.startup {
		feature.Stop(reason, context)
	}
	r.active = false
	r.alive = false
	for _, feature := range r.filter {
		feature.Stop(reason, context)
		r.alive = (context.keep || r.alive) && !context.hard
	}
	for _, feature := range r.event {
		feature.FinishEvent() //Same for finishevents
	}
}

func (r *record) filteredEvent(data interface{}, context *EventContext) {
RESTART:
	if !r.active {
		r.Start(context)
	}
	context.clear()
	for _, feature := range r.control {
		feature.Event(data, context, nil) // no tree for control
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
	for _, feature := range r.event {
		feature.Event(data, context, nil) //Events trickle down the tree
	}
	for _, feature := range r.event {
		feature.FinishEvent() //Same for finishevents
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
	r.Stop(reason, context)
	template := r.template
	for _, variant := range r.variant {
		template = template.subTemplate(variant.Variant())
	}
	for _, exporter := range r.exporter {
		exporter.Export(template, r.export, now)
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
	init       []featureToInit
	exportList []int
	exporter   []Exporter
	fields     []string
	make       func() *record
}

// Init must be called after a Record was instantiated
func (rm RecordMaker) Init() {
	for _, exporter := range rm.exporter {
		exporter.Fields(rm.fields)
	}
}

type graphInfo struct {
	ret          FeatureType
	data         interface{}
	nonComposite bool
	apply        string
}

type featureToInit struct {
	feature   featureMaker
	info      graphInfo
	ie        ipfix.InformationElement
	arguments []int
	call      []int
	event     bool
	eventid   int
	control   bool
	controlid int
	variant   bool
	variantid int
	export    bool
	exportid  int
}

// AppendRecord creates a internal representation needed for instantiating records from a feature
// specification, a list of exporters and a needed base (only FlowFeature supported so far)
func (rl *RecordListMaker) AppendRecord(features []interface{}, exporter []Exporter, base FeatureType) {
	type featureWithType struct {
		feature     interface{}
		ret         FeatureType
		ie          ipfix.InformationElement
		export      bool
		reset       bool
		selection   string
		compositeID string
		apply       string
	}

	init := make([]featureToInit, 0, len(features))
	var tofilter []featureToInit

	stack := make([]featureWithType, len(features))
	for i := range features {
		stack[i] = featureWithType{
			feature: features[i],
			ret:     base,
			export:  true,
		}
	}

	type selection struct {
		argument []int
		seen     map[string]int
	}

	selections := make(map[string]*selection)

	mainSelection := &selection{nil, make(map[string]int, len(features))}
	selections[feature2id([]interface{}{"select", true}, Selection)] = mainSelection
	currentSelection := mainSelection

	var exportList []int

	var currentFeature featureWithType
MAIN:
	for len(stack) > 0 {
		currentFeature, stack = stack[0], stack[1:]
		id := feature2id(currentFeature.feature, currentFeature.ret)
		seen := currentSelection.seen
		if i, ok := seen[id]; ok {
			if currentFeature.export {
				exportList = append(exportList, i)
			}
			continue MAIN
		}
		switch typedFeature := currentFeature.feature.(type) {
		case string:
			if feature, ok := getFeature(typedFeature, currentFeature.ret, 1); !ok {
				if composite, ok := compositeFeatures[typedFeature]; !ok {
					if currentSelection == mainSelection {
						if feature, ok := getFeature(typedFeature, ControlFeature, 1); ok {
							seen[id] = len(init)
							init = append(init, featureToInit{
								feature: feature,
								control: true,
							})
						} else {
							if feature, ok := getFeature(typedFeature, RawPacket, 1); ok {
								tofilter = append(tofilter, featureToInit{
									feature: feature,
								})
							} else {
								panic(fmt.Sprintf("(Control/Filter)Feature %s returning %s with input raw packet/flow not found", currentFeature.feature, currentFeature.ret))
							}
						}
					} else {
						panic(fmt.Sprintf("Feature %s returning %s with input raw packet/flow not found", currentFeature.feature, currentFeature.ret))
					}
				} else {
					stack = append([]featureWithType{
						{
							feature:     composite.definition,
							ie:          composite.ie,
							ret:         currentFeature.ret,
							export:      currentFeature.export,
							compositeID: id,
						}}, stack...)
				}
			} else {
				if feature.arguments[0] != RawPacket { //TODO: implement flow input
					panic(fmt.Sprintf("Feature %s returning %s with input raw packet not found", currentFeature.feature, currentFeature.ret))
				}
				seen[id] = len(init)
				if currentFeature.export {
					exportList = append(exportList, len(init))
				}
				ie := feature.ie
				if currentFeature.ie.Name != "" {
					ie.Name = currentFeature.ie.Name
					ie.Pen = 0
					ie.ID = 0
				}
				init = append(init, featureToInit{
					feature: feature,
					info: graphInfo{
						ret:   currentFeature.ret,
						apply: currentFeature.apply,
					},
					ie:        ie,
					arguments: currentSelection.argument,
					event:     currentSelection.argument == nil,
				})
			}
		case bool, float64, int64, uint64, int:
			feature := newConstantMetaFeature(typedFeature)
			seen[id] = len(init)
			if currentFeature.export {
				exportList = append(exportList, len(init))
			}
			init = append(init, featureToInit{
				feature: feature,
				info: graphInfo{
					data: typedFeature,
				},
				ie: feature.ie,
			})
		case []interface{}:
			fun := typedFeature[0].(string)
			if feature, ok := getFeature(fun, currentFeature.ret, len(typedFeature)-1); !ok {
				panic(fmt.Sprintf("Feature %s returning %s with arguments %v not found", fun, currentFeature.ret, typedFeature[1:]))
			} else {
				if fun == "apply" || fun == "map" {
					sel := feature2id(typedFeature[2], Selection)
					if fun == "apply" && currentFeature.ret != FlowFeature {
						panic("Unexpected apply - did you mean map?")
					} else if fun == "map" && currentFeature.ret != PacketFeature {
						panic("Unexpected map - did you mean apply?")
					}
					ie := ipfix.InformationElement{Name: strings.Join(compositeToCall(typedFeature), ""), Type: ipfix.IllegalType}
					if s, ok := selections[sel]; ok {
						stack = append([]featureWithType{
							{
								feature: typedFeature[1],
								ret:     currentFeature.ret,
								export:  currentFeature.export,
								apply:   fun,
								reset:   true,
								ie:      ie,
							},
						}, stack...)
						currentSelection = s
					} else {
						stack = append([]featureWithType{
							{
								feature:   typedFeature[2],
								ret:       Selection,
								selection: sel,
							},
							{
								feature: typedFeature[1],
								ret:     currentFeature.ret,
								export:  currentFeature.export,
								apply:   fun,
								reset:   true,
								ie:      ie,
							},
						}, stack...)
					}
					continue MAIN
				}
				argumentTypes := feature.getArguments(currentFeature.ret, len(typedFeature)-1)
				argumentPos := make([]int, 0, len(typedFeature)-1)
				for i, f := range typedFeature[1:] {
					if pos, ok := seen[feature2id(f, argumentTypes[i])]; !ok {
						newstack := make([]featureWithType, len(typedFeature)-1)
						for i, arg := range typedFeature[1:] {
							newstack[i] = featureWithType{
								feature: arg,
								ret:     argumentTypes[i],
							}
						}
						stack = append(append(newstack, currentFeature), stack...)
						continue MAIN
					} else {
						argumentPos = append(argumentPos, pos)
					}
				}
				seen[id] = len(init)
				if currentFeature.export {
					exportList = append(exportList, len(init))
				}
				ie := feature.ie
				ie.Pen = 0
				ie.ID = 0
				if currentFeature.compositeID != "" {
					seen[currentFeature.compositeID] = len(init)
					ie = currentFeature.ie
				} else {
					if feature.function {
						if feature.resolver != nil {
							ieargs := make([]ipfix.InformationElement, len(argumentPos))
							for i := range ieargs {
								ieargs[i] = init[argumentPos[i]].ie
							}
							ie = feature.resolver(ieargs)
						} else if feature.ie.Type != ipfix.IllegalType {
							ie.Type = feature.ie.Type
							ie.Length = feature.ie.Length
						} else {
							if len(argumentPos) == 1 {
								ie.Type = init[argumentPos[0]].ie.Type
								ie.Length = init[argumentPos[0]].ie.Length
							} else if len(argumentPos) == 2 {
								ie.Type = UpConvertTypes(init[argumentPos[0]].ie.Type, init[argumentPos[1]].ie.Type)
								ie.Length = ipfix.DefaultSize[ie.Type]
							} else {
								ie.Type = ipfix.IllegalType //FIXME
							}
						}
					}
					if currentFeature.export {
						ie.Name = strings.Join(compositeToCall(typedFeature), "")
					}
				}
				if currentFeature.selection != "" {
					currentSelection = &selection{[]int{len(init)}, make(map[string]int, len(features))}
					selections[currentFeature.selection] = currentSelection
				}
				//select: set event true (event from logic + event from base)
				init = append(init, featureToInit{
					feature: feature,
					info: graphInfo{
						ret:          currentFeature.ret,
						nonComposite: currentFeature.compositeID == "",
						apply:        currentFeature.apply,
					},
					arguments: argumentPos,
					ie:        ie,
					event:     fun == "select" || (fun == "select_slice" && len(argumentPos) == 2),
				})
			}
		default:
			panic(fmt.Sprint("Don't know what to do with ", currentFeature))
		}
		if currentFeature.reset {
			currentSelection = mainSelection
		}
	}

	for i, f := range init {
		for _, arg := range f.arguments {
			init[arg].call = append(init[arg].call, i)
		}
	}

	nevent := 0
	ncontrol := 0
	nvariants := 0
	type callSpec struct {
		id   int
		args []int
	}
	var tocall []callSpec

	for i, feature := range init {
		if len(feature.feature.variants) != 0 {
			init[i].variant = true
			init[i].variantid = nvariants
			nvariants++
		}
		if feature.event {
			init[i].eventid = nevent
			nevent++
		}
		if feature.control {
			init[i].controlid = ncontrol
			ncontrol++
		}
		if len(feature.call) > 0 {
			tocall = append(tocall, callSpec{i, feature.call})
		}
	}

	toexport := make([]featureToInit, len(exportList))
	toexport = toexport[:len(exportList)]
	for i, val := range exportList {
		toexport[i] = init[val]
		init[val].export = true
		init[val].exportid = i
	}

	template, fields := makeTemplate(toexport, &rl.templates)

	hasfilter := false
	if len(tofilter) > 0 {
		hasfilter = true
	}

	rl.list = append(rl.list, RecordMaker{
		init,
		exportList,
		exporter,
		fields,
		func() *record {
			toinit := init
			control := make([]Feature, ncontrol)
			event := make([]Feature, nevent)
			export := make([]Feature, len(exportList))
			exporter := exporter
			variant := make([]Feature, nvariants)
			template := template
			startup := make([]Feature, len(toinit))
			startup = startup[:len(toinit)] //BCE
			for i := range toinit {
				f := toinit[i].feature.make()
				startup[i] = f
				if toinit[i].event {
					event[toinit[i].eventid] = f
				}
				if toinit[i].control {
					control[toinit[i].controlid] = f
				}
				if toinit[i].variant {
					variant[toinit[i].variantid] = f
				}
				if toinit[i].export {
					export[toinit[i].exportid] = f
				}
				if len(toinit[i].arguments) > 0 {
					args := make([]Feature, len(toinit[i].arguments))
					args = args[:len(toinit[i].arguments)] //BCE
					for i, arg := range toinit[i].arguments {
						args[i] = startup[arg]
					}
					startup[i].SetArguments(args)
				}
			}
			var filter []Feature
			if hasfilter {
				filter = make([]Feature, len(tofilter))
				filter = filter[:len(tofilter)] //BCE
				for i, feature := range tofilter {
					filter[i] = feature.feature.make()
				}
			}
			for _, spec := range tocall {
				args := make([]Feature, len(spec.args))
				args = args[:len(spec.args)] //BCE
				for i, call := range spec.args {
					args[i] = startup[call]
				}
				startup[spec.id].setDependent(args)
			}
			return &record{
				startup:  startup,
				filter:   filter,
				control:  control,
				event:    event,
				export:   export,
				exporter: exporter,
				variant:  variant,
				template: template,
			}

		},
	})
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
			node.Label = feature.ie.Name
			if node.Label != feature.feature.ie.Name {
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
}
