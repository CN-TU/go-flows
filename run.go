package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
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
		fmt.Fprint(os.Stderr, "\nWrites the resulting callgraph in dot representation to stdout.")
	case "offline":
		cmdString(fmt.Sprintf("%s [args] %s input inputfile [...]", cmd, main))
		fmt.Fprint(os.Stderr, "\nParse the packets from input file(s) and export the specified feature set to the specified exporters.")
	case "online":
		cmdString(fmt.Sprintf("%s [args] %s input interface", cmd, main))
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

A list of supported exporters and features can be seen with the list
command. See also %s %s features -h.

Examples:
  Export the feature set specified in example.json to example.csv
    %s %s features example.json export csv example.csv [input ...]

  Export the feature sets a.json and b.json to a.csv and b.csv
    %s %s features a.json export csv a.csv features b.json export b.csv [input ...]

  Export the feature sets a.json and b.json to a single common.csv (this
  results in a csv with features from a in the odd lines, and features
  from b in the even lines)
    %s %s features a.json features b.json export common.csv [input ...]
`, os.Args[0], cmd, os.Args[0], cmd, os.Args[0], cmd, os.Args[0], cmd)
	flags()
	if cmd != "callgraph" {
		fmt.Fprintln(os.Stderr, "\nArgs:")
		tableset.PrintDefaults()
	}
}

func init() {
	addCommand("callgraph", "Create a callgraph from a flowspecification", parseArguments)
	addCommand("offline", "Extract flows from pcaps", parseArguments)
	addCommand("online", "Extract flows from a network interface", parseArguments)
}

func parseFeatures(cmd string, args []string) ([]string, []interface{}) {
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
	selection := set.String("select", "flows:0", "Flow selection (key:nth flow in specification)")
	set.Parse(args)
	if set.NArg() == 0 {
		log.Fatal("features needs a json file as input.")
	}
	selector := strings.Split(*selection, ":")
	if len(selector) != 2 {
		log.Fatalln("select must be of form 'key:id'!")
	}

	selectorID, err := strconv.Atoi(selector[1])
	if err != nil {
		log.Fatalln("select must be of form 'key:id'!")
	}

	features := decodeJSON(set.Arg(0), selector[0], selectorID)
	if features == nil {
		log.Fatalf("Couldn't parse %s (%s)", set.Arg(0), *selection)
	}
	_ = selection
	return set.Args()[1:], features
}

func parseExport(args []string) ([]string, bool) {
	return args, true
}

func parseArguments(cmd string, args []string) {
	type exportedFeatures struct {
		exporter   []flows.Exporter
		featureset [][]interface{}
	}

	set := flag.NewFlagSet("table", flag.ExitOnError)
	set.Usage = func() { tableUsage(cmd, set) }
	numProcessing := set.Uint("n", 4, "Number of parallel processing tables")
	activeTimeout := set.Uint("active", 1800, "Active timeout in seconds")
	idleTimeout := set.Uint("idle", 300, "Idle timeout in seconds")
	flowExpire := set.Uint("expire", 100, "Check for expired timers with this period in seconds. expire↓ ⇒ memory↓, execution time↑")
	maxPacket := set.Uint("size", 9000, "Maximum packet size read from source. 0 = automatic")
	bpfFilter := set.String("filter", "", "Process only packets matching specified bpf filter")

	set.Parse(args)
	if set.NArg() == 0 {
		set.Usage()
		os.Exit(-1)
	}

	arguments := set.Args()
	var featureset [][]interface{}
	var result []exportedFeatures
	clear := false
	var firstexporter []string
	exporters := make(map[string]flows.Exporter)
	var exportset []flows.Exporter
MAIN:
	for {
		if len(arguments) == 0 {
			if cmd == "callgraph" {
				break MAIN
			} else {
				log.Fatal("Command 'input' missing.")
			}
		}
		switch arguments[0] {
		case "features":
			if clear {
				if len(featureset) == 0 {
					log.Fatalf("At least one feature is needed for '%s'", strings.Join(firstexporter, " "))
				}
				firstexporter = nil
				clear = false
				result = append(result, exportedFeatures{exportset, featureset})
				featureset = nil
				exportset = nil
			}
			var features []interface{}
			arguments, features = parseFeatures(cmd, arguments[1:])
			featureset = append(featureset, features)
		case "export":
			if firstexporter == nil {
				firstexporter = arguments
			}
			if len(arguments) < 1 {
				log.Fatal("Need an export type")
			}
			var e flows.Exporter
			arguments, e = flows.MakeExporter(arguments[1], arguments[2:])
			if e == nil {
				log.Fatalf("Exporter %s not found", arguments[1])
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
			log.Fatalf("Command (features, export, input) missing, instead found '%s'", strings.Join(arguments, " "))
		}
	}

	if clear {
		if len(featureset) == 0 {
			log.Fatalf("At least one feature is needed for '%s'", strings.Join(firstexporter, " "))
		}
		result = append(result, exportedFeatures{exportset, featureset})
	}
	if len(result) == 0 {
		log.Fatalf("At least one exporter is needed!")
	}

	var featureLists flows.FeatureListCreatorList

	for _, featureset := range result {
		for _, feature := range featureset.featureset {
			featureLists = append(featureLists, flows.NewFeatureListCreator(feature, featureset.exporter, flows.FeatureTypeFlow))
		}
	}

	switch cmd {
	case "callgraph":
		featureLists.CallGraph(os.Stdout)
		return
	case "online":
		if len(arguments) != 1 {
			log.Fatal("Online mode needs extactly one interface!")
		}
	case "offline":
		if len(arguments) == 0 {
			log.Fatal("Offline mode needs one or more pcap files as input!")
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