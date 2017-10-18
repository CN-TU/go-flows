package flows

import (
	"fmt"
	"html/template"
	"io"
	"strconv"
	"strings"
)

type record struct {
	event    []Feature
	export   []Feature
	startup  []Feature
	exporter []Exporter
	variant  []Feature
	template *multiTemplate
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
	var template Template = r.template
	for _, variant := range r.variant {
		template = template.subTemplate(variant.Variant())
	}
	for _, exporter := range r.exporter {
		exporter.Export(template, r.export, context.When)
	}
}

type RecordListMaker []RecordMaker

func (rl RecordListMaker) make() (ret []*record) {
	ret = make([]*record, len(rl))
	for i, record := range rl {
		ret[i] = record.make()
	}
	return
}

type RecordMaker struct {
	init     []featureToInit
	exporter []Exporter
	make     func() *record
}

type graphInfo struct {
	ret      FeatureType
	constant interface{}
}

type featureToInit struct {
	feature   featureMaker
	info      graphInfo
	arguments []int
	call      []int
	event     bool
	export    bool
}

func NewRecordMaker(features []interface{}, exporter []Exporter, base FeatureType) RecordMaker {
	type featureWithType struct {
		feature     interface{}
		ret         FeatureType
		export      bool
		reset       bool
		selection   string
		compositeID string
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
							feature:     composite, //FIXME
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
						ret: currentFeature.ret,
					},
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
					constant: typedFeature,
				},
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
								reset:   true,
							},
						}, stack...)
					}
					continue MAIN
				} else {
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
					if currentFeature.compositeID != "" {
						seen[currentFeature.compositeID] = len(init)
					}
					if currentFeature.selection != "" {
						currentSelection = &selection{[]int{len(init)}, make(map[string]int, len(features))}
						selections[currentFeature.selection] = currentSelection
					}
					/*
						FIXME:
						if currentFeature.export {
							currentFeature.function = strings.Join(compositeToCall(typedFeature), "")
						}
					*/
					//select: set event true (event from logic + event from base)
					init = append(init, featureToInit{
						feature: feature,
						info: graphInfo{
							ret: currentFeature.ret,
						},
						arguments: argumentPos,
						event:     fun == "select" || (fun == "select_slice" && len(argumentPos) == 2),
						export:    currentFeature.export,
					})
				}
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
	for _, feature := range init {
		if feature.export {
			/*			var basetype string
						if feature.composite != "" && feature.composite != "apply" && feature.composite != "map" {
							basetype = feature.composite
						} else if feature.function != "" {
							basetype = feature.function
						} else {
							basetype = feature.feature.BaseType()
						}
						basetypes = append(basetypes, basetype)*/

			nexport++
		}
		if feature.event {
			nevent++
		}
	}

	return RecordMaker{
		init,
		exporter,
		func() *record {
			f := make([]Feature, len(init))
			event := make([]Feature, 0, nevent)
			export := make([]Feature, 0, nexport)
			for i, feature := range init {
				f[i] = feature.feature.make()
				if feature.event {
					event = append(event, f[i])
				}
				if feature.export {
					export = append(export, f[i])
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
			}
		},
	}
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
	for listID, fl := range r {
		var nodes []Node
		export := make([]Node, len(fl.exporter))
		for i, exporter := range fl.exporter {
			export[i] = Node{Name: fmt.Sprintf("%p", exporter), Label: exporter.ID()} //FIXME: better style
		}
		for i, feature := range fl.init {
			var node Node
			node.Label = feature.feature.ie.Name
			/*
				FIXME:
				if feature.composite == "apply" || feature.composite == "map" {
					node.Label = fmt.Sprintf("%s\\n%s", feature.composite, node.Label)
				} else if feature.composite != "" {
					node.Label = fmt.Sprintf("%s\\n%s", node.Label, feature.composite)
				}
			*/
			node.Name = toId(listID, i)
			if feature.info.constant == nil {
				if len(feature.arguments) == 0 {
					node.Style = append(styles[feature.info.ret], []string{"fillcolor", "green"})
				} else if len(feature.arguments) == 1 {
					/*
						FIXME: if feature.composite == "" {
							node.Style = append(styles[feature.info.ret], []string{"fillcolor", "orange"})
						} else {
					*/
					node.Style = append(styles[feature.info.ret], []string{"fillcolor", "green:orange"})
					//}
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
				node.Label = fmt.Sprint(feature.info.constant)
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
