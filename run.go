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
	main := "features spec.json [featureargs] [features ...] [export type [exportargs]] [export ...] [...]"
	switch cmd {
	case "callgraph":
		cmdString(fmt.Sprintf("%s %s", cmd, main))
		fmt.Fprint(os.Stderr, "\nWrites the callgraph in dot representation to stdout.\n")
	case "offline":
		cmdString(fmt.Sprintf("%s [args] %s input inputfile [...]", cmd, main))
		fmt.Fprint(os.Stderr, "\nParses the packets from the input file(s) and export the specified feature set to the specified exporters.\n")
	case "online":
		cmdString(fmt.Sprintf("%s [args] %s input interface", cmd, main))
		fmt.Fprint(os.Stderr, "\nParses the packets from network interface and export the specified feature set to the specified exporters.\n")
	}
	flags()
	if cmd != "callgraph" {
		fmt.Fprintln(os.Stderr, "\nArgs:")
		tableset.PrintDefaults()
	}
	fmt.Fprintln(os.Stderr, "\nFeatureArgs:")
	fmt.Fprintln(os.Stderr, "\nExportArgs:")
}

func init() {
	addCommand("callgraph", "Create a callgraph from a flowspecification", parseArguments)
	addCommand("offline", "Extract flows from pcaps", parseArguments)
	addCommand("online", "Extract flows from a network interface", parseArguments)
}

// go-flows online -timeout 0 -n 4 features asd.json export csv -file - -- <interface>
// go-flows online -timeout 0 -n 4 features asd.json export csv -file - -- <interface>

func parseFeatures(args []string) ([]string, []interface{}) {
	set := flag.NewFlagSet("features", flag.ExitOnError)
	set.Usage = func() { tableUsage("", set) }
	selection := set.String("select", "flows:0", "flow selection (key:number in specification)")
	set.Parse(args)
	if set.NArg() == 0 {
		tableUsage("", set) //print feature usage
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
	numProcessing := set.Uint("n", 4, "Number of parallel processing queues")
	activeTimeout := set.Uint("active", 1800, "Active timeout in seconds")
	idleTimeout := set.Uint("idle", 300, "Idle timeout in seconds")
	flowExpire := set.Uint("expire", 100, "Check for expired Timers after this time. expire↓ = memory↓, execution time↑")
	maxPacket := set.Uint("size", 9000, "Maximum packet size")
	bpfFilter := set.String("filter", "", "Process only packets matching specified bpf filter")
	set.Parse(args)
	arguments := set.Args()
	var featureset [][]interface{}
	var result []exportedFeatures
	clear := false
	exporters := make(map[string]flows.Exporter)
	var exportset []flows.Exporter
MAIN:
	for {
		if len(arguments) == 0 {
			if cmd == "callgraph" {
				break MAIN
			} else {
				log.Fatal("Command 'input' missing")
			}
		}
		switch arguments[0] {
		case "features":
			if clear {
				clear = false
				result = append(result, exportedFeatures{exportset, featureset})
				featureset = nil
				exportset = nil
			}
			var features []interface{}
			arguments, features = parseFeatures(arguments[1:])
			featureset = append(featureset, features)
		case "export":
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
			log.Fatal("Command (features, export, input) missing")
		}
	}

	if clear {
		result = append(result, exportedFeatures{exportset, featureset})
	}
	if len(result) == 0 {
		tableUsage(cmd, set)
		os.Exit(-1)
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
