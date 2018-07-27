package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"reflect"
	"runtime/debug"
	"runtime/pprof"
	"sort"
	"strings"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/packet"
	"github.com/CN-TU/go-flows/util"
)

func tableUsage(cmd string, tableset *flag.FlagSet) {
	main := "features [featureargs] spec.json [features ...] [export type [exportargs]] [export ...] [...]"
	switch cmd {
	case "callgraph":
		cmdString(fmt.Sprintf("%s %s", cmd, main))
		cmdString(fmt.Sprintf("%s [args] -spec commands.json", cmd))
		fmt.Fprint(os.Stderr, "\nWrites the resulting callgraph in dot representation to stdout.")
	case "offline":
		cmdString(fmt.Sprintf("%s [args] %s input inputfile [...]", cmd, main))
		cmdString(fmt.Sprintf("%s [args] -spec commands.json inputfile [...]", cmd))
		fmt.Fprint(os.Stderr, `
Parse the packets from input file(s) and export the specified feature set to the specified exporters.

Packet labels might be provided as csv files. These files must start with a
header. If there is only one column in the csv file, every row is matched
up with a packet. If there are at least two columns, the first row must be
the packet number this label belongs to. The number of the first packet is
1!`)
	case "online":
		cmdString(fmt.Sprintf("%s [args] %s input interface", cmd, main))
		cmdString(fmt.Sprintf("%s [args] -spec commands.json interface", cmd))
		fmt.Fprint(os.Stderr, "\nParse the packets from a network interface and export the specified feature set to the specified exporters.")
	}
	fmt.Fprintf(os.Stderr, `

Featuresets (features), outputs (export), and, optionally, sources need to
be provided. It is possible to specify multiple feature statements and
multiple export statements. All the specified exporters always export the
features specified by the preceeding feature group.

At least one feature specification and one exporter is needed.

Identical exportes can be specified multiple times. Beware, that those will
share a common exporter instance, resulting in a field set specification
per specified featureset, and mixed field sets (depending on the feature
specification).

Instead of providing the commands on the command line, it is also possible
to use a json file.

A list of supported exporters and features can be seen with the list
command. See also %s %s features -h.

Examples:
  Export the feature set specified in example.json to example.csv
    %s %s features example.json export csv example.csv input [input ...]

  Export the feature sets a.json and b.json to a.csv and b.csv
    %s %s features a.json export csv a.csv features b.json export b.csv input [input ...]

  Export the feature sets a.json and b.json to a single common.csv (this
  results in a csv with features from a in the odd lines, and features
  from b in the even lines)
    %s %s features a.json features b.json export common.csv input [input ...]

  Execute the commands provided in commands.json
    %s %s -spec commands.json [...]

`, os.Args[0], cmd, os.Args[0], cmd, os.Args[0], cmd, os.Args[0], cmd, os.Args[0], cmd)
	flags()
	fmt.Fprintln(os.Stderr, "\nArgs:")
	tableset.PrintDefaults()
}

func init() {
	addCommand("run", "Extract flows", parseArguments)
	addCommand("callgraph", "Create a callgraph from a flowspecification", parseArguments)
}

func parseFeatures(cmd string, args []string) (arguments []string, features []interface{}, key []string, bidirectional bool) {
	set := flag.NewFlagSet("features", flag.ExitOnError)
	set.Usage = func() {
		fmt.Fprint(os.Stderr, `
Usage:
  features [args] spec.json

Reads feature list, as specified in spec.json.

Args:
`)
		set.PrintDefaults()
	}
	selection := set.Uint("select", 0, "Use nth flow selection (key:nth flow in specification)")
	v1 := set.Bool("v1", false, "Force v1 format")
	v2 := set.Bool("v2", false, "Force v2 format")
	simple := set.Bool("simple", false, "Treat file as if it only contains the flow specification")
	set.Parse(args)
	if (*v1 && *v2) || (*v1 && *simple) || (*v2 && *simple) {
		log.Fatalf("Only one of -v1, -v2, or -simple can be chosen\n")
	}
	if set.NArg() == 0 {
		log.Fatalln("features needs a json file as input.")
	}
	arguments = set.Args()[1:]

	format := jsonAuto
	switch {
	case *v1:
		format = jsonV1
	case *v2:
		format = jsonV2
	case *simple:
		format = jsonSimple
	}

	features, key, bidirectional = decodeJSON(set.Arg(0), format, int(*selection))
	if features == nil {
		log.Fatalf("Couldn't parse %s (%d) - features missing\n", set.Arg(0), *selection)
	}
	if key == nil {
		log.Fatalf("Couldn't parse %s (%d) - flow key missing\n", set.Arg(0), *selection)
	}
	return
}

func parseExport(args []string) ([]string, bool) {
	return args, true
}

type featureSpec struct {
	features      []interface{}
	key           []string
	bidirectional bool
}

type exportedFeatures struct {
	exporter   []flows.Exporter
	featureset []featureSpec
}

// TODO: fix this function to the new style
func parseCommandFile(cmd string, file string) (result []exportedFeatures, exporters map[string]flows.Exporter, filters packet.Filters, sources packet.Sources, labels packet.Labels) {
	f, err := os.Open(file)
	if err != nil {
		log.Fatalln("Can't open ", file)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.UseNumber()

	var decoded struct {
		Exporters map[string]struct {
			Type    string
			Options interface{}
		}
		Features []struct {
			Exporters []string
			Input     []map[string]interface{}
		}
	}

	if err := dec.Decode(&decoded); err != nil {
		log.Fatalln("Couldn' parse command spec:", err)
	}

	exporters = make(map[string]flows.Exporter)

	for name, options := range decoded.Exporters {
		if _, exporter, err := flows.MakeExporter(options.Type, name, options.Options, nil); err == nil {
			exporters[name] = exporter
		} else {
			log.Fatalf("Error initializing exporter '%s': %s\n", options.Type, err)
		}
	}

	reldir := path.Dir(file)

	for _, feature := range decoded.Features {
		var toexport exportedFeatures
		for _, exporter := range feature.Exporters {
			if e, ok := exporters[exporter]; ok {
				toexport.exporter = append(toexport.exporter, e)
			} else {
				log.Fatalf("Couldn' find exporter with name '%s'\n", exporter)
			}
		}
		for _, input := range feature.Input {
			if file, ok := input["file"].(string); ok {
				t := jsonAuto
				if v, ok := input["type"].(string); ok {
					switch v {
					case "v1":
						t = jsonV1
					case "v2":
						t = jsonV2
					case "simple":
						t = jsonSimple
					default:
						log.Fatalf("Don't know file type '%s' (I only know v1, v2, simple)\n", v)
					}
				}
				id := 0
				if i, ok := input["id"].(json.Number); ok {
					if i, err := i.Int64(); err == nil {
						id = int(i)
					} else {
						log.Fatalf("'%s' is not a valid id", input["id"])
					}
				}
				var f featureSpec
				if !path.IsAbs(file) {
					file = path.Join(reldir, file)
				}
				f.features, f.key, f.bidirectional = decodeJSON(file, t, id)
				toexport.featureset = append(toexport.featureset, f)
			} else {
				var f featureSpec
				var key []interface{}
				var ok bool
				if f.bidirectional, ok = input["bidirectional"].(bool); !ok {
					log.Fatalln("Bidirectional must be a bool")
				}
				if key, ok = input["key_features"].([]interface{}); !ok {
					log.Fatalln("key_features must be a list of strings")
				} else {
					f.key = make([]string, len(key))
					for i, elem := range key {
						if f.key[i], ok = elem.(string); !ok {
							log.Fatalln("key_features must be a list of strings")
						}
					}
				}
				f.features = decodeFeatures(input["features"])
				toexport.featureset = append(toexport.featureset, f)
			}
		}
		result = append(result, toexport)
	}

	return
}

func parseCommandLine(cmd string, args []string) (result []exportedFeatures, exporters map[string]flows.Exporter, filters packet.Filters, sources packet.Sources, labels packet.Labels) {
	var featureset []featureSpec
	clear := false
	var firstexporter []string
	exporters = make(map[string]flows.Exporter)
	var exportset []flows.Exporter
	var err error
	for len(args) >= 2 {
		typ := args[0]
		name := args[1]
		switch typ {
		case "features":
			if clear {
				if len(featureset) == 0 {
					log.Fatalf("At least one feature is needed for '%s'\n", strings.Join(firstexporter, " "))
				}
				firstexporter = nil
				clear = false
				result = append(result, exportedFeatures{exportset, featureset})
				featureset = nil
				exportset = nil
			}
			var f featureSpec
			args, f.features, f.key, f.bidirectional = parseFeatures(cmd, args[1:])
			featureset = append(featureset, f)
		case "export":
			if firstexporter == nil {
				firstexporter = args
			}
			if len(args) < 1 {
				log.Fatalln("Need an export type")
			}
			var e flows.Exporter
			args, e, err = flows.MakeExporter(name, "", util.UseStringOption{}, args[2:])
			if err != nil {
				log.Fatalf("Error creating exporter '%s': %s\n", name, err)
			}
			if existing, ok := exporters[e.ID()]; ok {
				e = existing
			} else {
				exporters[e.ID()] = e
			}
			exportset = append(exportset, e)
			clear = true
		case "source":
			if len(args) < 1 {
				log.Fatalln("Need a source type")
			}
			var s packet.Source
			args, s, err = packet.MakeSource(name, "", util.UseStringOption{}, args[2:])
			if err != nil {
				log.Fatalf("Error creating source '%s': %s\n", name, err)
			}
			sources.Append(s)
		case "filter":
			if len(args) < 1 {
				log.Fatalln("Need a filter type")
			}
			var s packet.Filter
			args, s, err = packet.MakeFilter(name, "", util.UseStringOption{}, args[2:])
			if err != nil {
				log.Fatalf("Error creating filter '%s': %s\n", name, err)
			}
			filters = append(filters, s)
		case "label":
			if len(args) < 1 {
				log.Fatalln("Need a label type")
			}
			var s packet.Label
			args, s, err = packet.MakeLabel(name, "", util.UseStringOption{}, args[2:])
			if err != nil {
				log.Fatalf("Error creating label '%s': %s\n", name, err)
			}
			labels = append(labels, s)
		default:
			log.Fatalf("Command (features, export, source, label, filter) missing, instead found '%s'\n", strings.Join(args, " "))
		}
	}
	if len(args) > 0 {
		log.Fatalf("Argument at end of input to '%s' is missing!\n", args[0])
	}

	if clear {
		if len(featureset) == 0 {
			log.Fatalf("At least one feature is needed for '%s'\n", strings.Join(firstexporter, " "))
		}
		result = append(result, exportedFeatures{exportset, featureset})
	}
	return
}

func parseArguments(cmd string, args []string) {
	set := flag.NewFlagSet("table", flag.ExitOnError)
	set.Usage = func() { tableUsage(cmd, set) }
	numProcessing := set.Uint("n", 4, "Number of parallel processing tables")
	activeTimeout := set.Uint("active", 1800, "Active timeout in seconds")
	idleTimeout := set.Uint("idle", 300, "Idle timeout in seconds")
	perPacket := set.Bool("perpacket", false, "Export one flow per Packet")
	flowExpire := set.Uint("expire", 100, "Check for expired timers with this period in seconds. expire↓ ⇒ memory↓, execution time↑")
	maxPacket := set.Uint("size", 9000, "Maximum packet size handled internally. 0 = automatic")
	commands := set.String("spec", "", "Load exporters and features from specified json file")
	printStats := set.Bool("stats", false, "Output statistics")

	set.Parse(args)
	if set.NArg() == 0 {
		set.Usage()
		os.Exit(-1)
	}

	if *numProcessing == 0 {
		log.Fatalln("Need at least one flow processing table!")
	}

	var result []exportedFeatures
	var exporters map[string]flows.Exporter
	var filters packet.Filters
	var sources packet.Sources
	var labels packet.Labels

	if *commands != "" {
		result, exporters, filters, sources, labels = parseCommandFile(cmd, *commands)
	} else {
		result, exporters, filters, sources, labels = parseCommandLine(cmd, set.Args())
	}

	if len(result) == 0 {
		log.Fatalf("At least one exporter is needed!\n")
	}

	var recordList flows.RecordListMaker

	var key []string
	var bidirectional bool
	first := true

	for _, featureset := range result {
		for _, feature := range featureset.featureset {
			sort.Strings(feature.key)
			if first {
				first = false
				key = feature.key
				bidirectional = feature.bidirectional
			} else {
				if !reflect.DeepEqual(key, feature.key) {
					log.Fatalln("key_features of every flowspec must match")
				}
				if bidirectional != feature.bidirectional {
					log.Fatalln("bidirectional of every flow must match")
				}
			}
			recordList.AppendRecord(feature.features, featureset.exporter, flows.FlowFeature)
		}
	}

	keyselector := packet.MakeDynamicKeySelector(key, bidirectional)

	if cmd == "callgraph" {
		recordList.CallGraph(os.Stdout)
		return
	}

	//	log.Fatalln(exporters, filters, sources, labels)

	for _, exporter := range exporters {
		exporter.Init()
	}

	recordList.Init()

	flows.CleanupFeatures()
	util.CleanupModules()

	debug.SetGCPercent(10000000) //We manually call gc after timing out flows; make that optional?

	flowtable := packet.NewParallelFlowTable(int(*numProcessing), recordList, packet.NewFlow,
		flows.FlowOptions{
			ActiveTimeout: flows.DateTimeNanoseconds(*activeTimeout) * flows.SecondsInNanoseconds,
			IdleTimeout:   flows.DateTimeNanoseconds(*idleTimeout) * flows.SecondsInNanoseconds,
			PerPacket:     *perPacket,
		},
		flows.DateTimeNanoseconds(*flowExpire)*flows.SecondsInNanoseconds, keyselector)

	engine := packet.NewEngine(int(*maxPacket), flowtable, filters, sources, labels)

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, os.Interrupt)

	go func() {
		<-cancel
		log.Println("Canceling...")
		engine.Stop()
	}()

	stopped := engine.Run()

	signal.Stop(cancel)

	engine.Finish()

	if *heapprofile != "" {
		f, err := os.Create(*heapprofile)
		if err != nil {
			log.Fatalln("could not create memory profile: ", err)
		}
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatalln("could not write memory profile: ", err)
		}
		f.Close()
	}

	flowtable.EOF(stopped)
	for _, exporter := range exporters {
		exporter.Finish()
	}
	if *printStats {
		engine.PrintStats(os.Stderr)
		flowtable.PrintStats(os.Stderr)
	}
}
