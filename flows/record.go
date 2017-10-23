package flows

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/template"

	"pm.cn.tuwien.ac.at/ipfix/go-ipfix"
)

type record struct {
	event    []Feature
	export   []Feature
	startup  []Feature
	exporter []Exporter
	variant  []Feature
	template Template
}

func (r *record) Start(context EventContext) {
	for _, feature := range r.startup {
		feature.Start(context)
	}
}

func (r *record) Stop(reason FlowEndReason, context EventContext) {
	for _, feature := range r.startup {
		feature.Stop(reason, context)
	}
}

func (r *record) Event(data interface{}, context EventContext) {
	for _, feature := range r.event {
		feature.Event(data, context, nil) //Events trickle down the tree
	}
	for _, feature := range r.event {
		feature.FinishEvent() //Same for finishevents
	}
	//FIXME read stuff todo (reset, stop, whatever) from context
}

func (r *record) Export(context EventContext) {
	template := r.template
	for _, variant := range r.variant {
		template = template.subTemplate(variant.Variant())
	}
	for _, exporter := range r.exporter {
		exporter.Export(template, r.export, context.When)
	}
}

type RecordListMaker struct {
	list      []RecordMaker
	templates int
}

func (rl RecordListMaker) make() (ret []*record) {
	ret = make([]*record, len(rl.list))
	for i, record := range rl.list {
		ret[i] = record.make()
	}
	return
}

func (rl RecordListMaker) Init() {
	for _, record := range rl.list {
		record.Init()
	}
}

type RecordMaker struct {
	init     []featureToInit
	exporter []Exporter
	fields   []string
	make     func() *record
}

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
	export    bool
	variant   bool
}

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

	var currentFeature featureWithType
MAIN:
	for len(stack) > 0 {
		currentFeature, stack = stack[0], stack[1:]
		id := feature2id(currentFeature.feature, currentFeature.ret)
		seen := currentSelection.seen
		if _, ok := seen[id]; ok {
			continue MAIN
		}
		switch typedFeature := currentFeature.feature.(type) {
		case string:
			if feature, ok := getFeature(typedFeature, currentFeature.ret, 1); !ok {
				if composite, ok := compositeFeatures[typedFeature]; !ok {
					panic(fmt.Sprintf("Feature %s returning %s with input raw packet/flow not found", currentFeature.feature, currentFeature.ret))
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
				init = append(init, featureToInit{
					feature: feature,
					info: graphInfo{
						ret:   currentFeature.ret,
						apply: currentFeature.apply,
					},
					ie:        feature.ie, //this depends on function!
					arguments: currentSelection.argument,
					event:     currentSelection.argument == nil,
					export:    currentFeature.export,
				})
			}
		case bool, float64, int64, uint64, int:
			feature := newConstantMetaFeature(typedFeature)
			seen[id] = len(init)
			init = append(init, featureToInit{
				feature: feature,
				info: graphInfo{
					data: typedFeature,
				},
				ie:     feature.ie,
				export: currentFeature.export,
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
					/*
						if currentFeature.export {
							currentFeature.function = strings.Join(compositeToCall(typedFeature), "")
						}
						FIXME
					*/
					if s, ok := selections[sel]; ok {
						stack = append([]featureWithType{
							{
								feature: typedFeature[1],
								ret:     currentFeature.ret,
								export:  currentFeature.export,
								apply:   fun,
								reset:   true,
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
				ie := feature.ie
				if currentFeature.compositeID != "" {
					seen[currentFeature.compositeID] = len(init)
					ie = currentFeature.ie
				} else {
					if currentFeature.export {
						ie.Name = strings.Join(compositeToCall(typedFeature), "")
					}
					if feature.function {
						if feature.ie.Type != ipfix.IllegalType {
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
					export:    currentFeature.export,
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
	nexport := 0
	nvariants := 0

	for i, feature := range init {
		if len(feature.feature.variants) != 0 {
			nvariants++
			init[i].variant = true
		}
		if feature.export {
			nexport++
		}
		if feature.event {
			nevent++
		}
	}

	toexport := make([]featureToInit, 0, nexport)
	for _, feature := range init {
		if feature.export {
			toexport = append(toexport, feature)
		}
	}

	template, fields := makeTemplate(toexport, &rl.templates) //only exported ones!

	for _, feature := range init {
		fmt.Fprintf(os.Stderr, "%#v\n", feature)
	}

	rl.list = append(rl.list, RecordMaker{
		init,
		exporter,
		fields,
		func() *record {
			f := make([]Feature, len(init))
			event := make([]Feature, 0, nevent)
			export := make([]Feature, 0, nexport)
			variants := make([]Feature, 0, nvariants)
			for i, feature := range init {
				f[i] = feature.feature.make()
				if feature.event {
					event = append(event, f[i])
				}
				if feature.export {
					export = append(export, f[i])
				}
				if feature.variant {
					variants = append(variants, f[i])
				}
			}
			for i, feature := range init {
				if len(feature.call) > 0 {
					args := make([]Feature, len(feature.call))
					for i, call := range feature.call {
						args[i] = f[call]
					}
					f[i].setDependent(args)
				}
				if len(feature.arguments) > 0 {
					args := make([]Feature, len(feature.arguments))
					for i, arg := range feature.arguments {
						args[i] = f[arg]
					}
					f[i].SetArguments(args)
				}
			}
			return &record{
				startup:  f,
				event:    event,
				export:   export,
				exporter: exporter,
				variant:  variants,
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
func (r RecordListMaker) CallGraph(w io.Writer) {
	toId := func(id, i int) string {
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
	for listID, fl := range r.list {
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
			node.Name = toId(listID, i)
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
					node.HTML = fmt.Sprintf(`<TABLE BORDER="0" CELLBORDER="0" CELLSPACING="2"><TR>%s</TR><TR><TD COLSPAN="%d">%s</TD></TR></TABLE>`, strings.Join(args, ""), len(feature.arguments), node.Label)
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
				data.Edges = append(data.Edges, Edge{"event", "", toId(listID, i), ""})
			}
			if feature.export {
				data.Edges = append(data.Edges, Edge{toId(listID, i), "", fmt.Sprintf("export%d", listID), ""})
			}
			for index, j := range feature.arguments {
				index := strconv.Itoa(index)
				if len(feature.arguments) <= 1 {
					index = ""
				}
				data.Edges = append(data.Edges, Edge{toId(listID, j), "", toId(listID, i), index})
			}
		}
		data.Nodes = append(data.Nodes, Subgraph{nodes, export})
	}
	graphTemplate.Execute(w, data)
}
