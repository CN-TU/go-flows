package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"runtime/debug"
	"runtime/pprof"
	"strings"

	_ "pm.cn.tuwien.ac.at/ipfix/go-flows/exporters"
	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
	"pm.cn.tuwien.ac.at/ipfix/go-flows/packet"
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
		fmt.Fprint(os.Stderr, "\nParse the packets from input file(s) and export the specified feature set to the specified exporters.")
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
	addCommand("callgraph", "Create a callgraph from a flowspecification", parseArguments)
	addCommand("offline", "Extract flows from pcaps", parseArguments)
	addCommand("online", "Extract flows from a network interface", parseArguments)
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
		log.Fatalf("Couldn't parse %s (%d) - key missing\n", set.Arg(0), *selection)
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

func parseCommandFile(cmd string, file string) (result []exportedFeatures, exporters map[string]flows.Exporter) {
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
		if _, exporter := flows.MakeExporter(options.Type, name, options.Options, nil); exporter != nil {
			exporters[name] = exporter
		} else {
			log.Fatalf("Couldn't find exporter with type '%s'\n", options.Type)
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

func parseCommandLine(cmd string, args []string) (result []exportedFeatures, exporters map[string]flows.Exporter, arguments []string) {
	arguments = args
	var featureset []featureSpec
	clear := false
	var firstexporter []string
	exporters = make(map[string]flows.Exporter)
	var exportset []flows.Exporter
MAIN:
	for {
		if len(arguments) == 0 {
			if cmd == "callgraph" {
				break MAIN
			} else {
				log.Fatalln("Command 'input' missing.")
			}
		}
		switch arguments[0] {
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
			arguments, f.features, f.key, f.bidirectional = parseFeatures(cmd, arguments[1:])
			featureset = append(featureset, f)
		case "export":
			if firstexporter == nil {
				firstexporter = arguments
			}
			if len(arguments) < 1 {
				log.Fatalln("Need an export type")
			}
			var e flows.Exporter
			arguments, e = flows.MakeExporter(arguments[1], "", flows.UseStringOption{}, arguments[2:])
			if e == nil {
				log.Fatalf("Exporter %s not found\n", arguments[1])
			}
			if existing, ok := exporters[e.ID()]; ok {
				e = existing
			} else {
				exporters[e.ID()] = e
			}
			exportset = append(exportset, e)
			clear = true
		case "input":
			arguments = arguments[1:]
			break MAIN
		default:
			log.Fatalf("Command (features, export, input) missing, instead found '%s'\n", strings.Join(arguments, " "))
		}
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
	flowExpire := set.Uint("expire", 100, "Check for expired timers with this period in seconds. expire↓ ⇒ memory↓, execution time↑")
	maxPacket := set.Uint("size", 9000, "Maximum packet size read from source. 0 = automatic")
	bpfFilter := set.String("filter", "", "Process only packets matching specified bpf filter")
	commands := set.String("spec", "", "Load exporters and features from specified json file")

	set.Parse(args)
	if set.NArg() == 0 {
		set.Usage()
		os.Exit(-1)
	}

	var result []exportedFeatures
	var exporters map[string]flows.Exporter
	var arguments []string

	if *commands != "" {
		result, exporters = parseCommandFile(cmd, *commands)
		arguments = set.Args()
	} else {
		result, exporters, arguments = parseCommandLine(cmd, set.Args())
	}

	if len(result) == 0 {
		log.Fatalf("At least one exporter is needed!\n")
	}

	var featureLists flows.FeatureListCreatorList

	for _, featureset := range result {
		for _, feature := range featureset.featureset {
			featureLists = append(featureLists, flows.NewFeatureListCreator(feature.features, featureset.exporter, flows.FeatureTypeFlow))
		}
	}

	switch cmd {
	case "callgraph":
		featureLists.CallGraph(os.Stdout)
		return
	case "online":
		if len(arguments) != 1 {
			log.Fatalln("Online mode needs extactly one interface!")
		}
	case "offline":
		if len(arguments) == 0 {
			log.Fatalln("Offline mode needs one or more pcap files as input!")
		}
	}

	for _, exporter := range exporters {
		exporter.Init()
	}
	featureLists.Fields()

	flows.CleanupFeatures()

	debug.SetGCPercent(10000000) //We manually call gc after timing out flows; make that optional?

	flowtable := packet.NewParallelFlowTable(int(*numProcessing), featureLists, packet.NewFlow,
		flows.Time(*activeTimeout)*flows.Seconds, flows.Time(*idleTimeout)*flows.Seconds,
		flows.Time(*flowExpire)*flows.Seconds)

	buffer := packet.NewPcapBuffer(int(*maxPacket), flowtable)
	buffer.SetFilter(*bpfFilter)

	var time flows.Time

	if cmd == "online" {
		time = buffer.ReadInterface(arguments[0])
	} else {
		for _, fname := range arguments {
			time = buffer.ReadFile(fname)
		}
	}

	buffer.Finish()

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatalln("could not create memory profile: ", err)
		}
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatalln("could not write memory profile: ", err)
		}
		f.Close()
	}

	flowtable.EOF(time)
	for _, exporter := range exporters {
		exporter.Finish()
	}
}
