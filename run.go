package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
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
	main := "features [featureargs] spec.json [features ...] [export type [exportargs]] [filter type [filterargs]] [label type [filterargs]] input type [inputargs] [...]"
	switch cmd {
	case "callgraph":
		cmdString(fmt.Sprintf("%s %s", cmd, main))
		fmt.Fprint(os.Stderr, "\nWrites the resulting callgraph in dot representation to stdout.")
	case "run":
		cmdString(fmt.Sprintf("%s [args] %s input inputfile [...]", cmd, main))
		fmt.Fprint(os.Stderr, `
Parse the packets from input source(s), apply filter(s) and/or label(s) and
export the specified feature set to the specified exporters.`)
	}
	fmt.Fprintf(os.Stderr, `

Featuresets (features), outputs (export), and, optionally, sources need to
be provided. It is possible to specify multiple feature statements and
multiple export statements. All the specified exporters always export the
features specified by the preceeding feature group.

If multiple sources are specified, those are queried in order.

If multiple filters are specified, those are tried in order.

If multiple labels are specified, those are queried in order.

At least one feature specification and one exporter is needed.

Identical exportes can be specified multiple times. Beware, that those will
share a common exporter instance, resulting in a field set specification
per specified featureset, and mixed field sets (depending on the feature
specification).

A list of supported exporters and features can be seen with the list
command. See also %s %s features -h.

Examples:
  Export the feature set specified in example.json to example.csv
    %s %s features example.json export csv example.csv source [sourcetype ...]

  Export the feature sets a.json and b.json to a.csv and b.csv
    %s %s features a.json export csv a.csv features b.json export b.csv source [sourcetype ...]

  Export the feature sets a.json and b.json to a single common.csv (this
  results in a csv with features from a in the odd lines, and features
  from b in the even lines)
    %s %s features a.json features b.json export common.csv source [sourcetype ...]

`, os.Args[0], cmd, os.Args[0], cmd, os.Args[0], cmd, os.Args[0], cmd)
	flags()
	fmt.Fprintln(os.Stderr, "\nArgs:")
	tableset.PrintDefaults()
}

func init() {
	addCommand("run", "Extract flows", parseArguments)
	addCommand("callgraph", "Create a callgraph from a flowspecification", parseArguments)
}

func parseFeatures(cmd string, args []string) (arguments []string, features []interface{}, control, filter, key []string, bidirectional, allowZero bool, opt flows.FlowOptions) {
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
	v2 := set.Bool("v2", false, "Force v2 format")
	simple := set.Bool("simple", false, "Treat file as if it only contains the flow specification")
	set.Parse(args)
	if *v2 && *simple {
		log.Fatalf("Only one of -v2, or -simple can be chosen\n")
	}
	if set.NArg() == 0 {
		log.Fatalln("features needs a json file as input.")
	}
	arguments = set.Args()[1:]

	format := jsonAuto
	switch {
	case *v2:
		format = jsonV2
	case *simple:
		format = jsonSimple
	}

	features, control, filter, key, bidirectional, allowZero, opt = decodeJSON(set.Arg(0), format, int(*selection))
	if features == nil {
		log.Fatalf("Couldn't parse %s (%d) - features missing\n", set.Arg(0), *selection)
	}
	if key == nil {
		log.Fatalf("Couldn't parse %s (%d) - flow key missing\n", set.Arg(0), *selection)
	}
	return
}

type featureSpec struct {
	features      []interface{}
	control       []string
	filter        []string
	key           []string
	bidirectional bool
	allowZero     bool
	opt           flows.FlowOptions
}

type exportedFeatures struct {
	exporter   []flows.Exporter
	featureset []featureSpec
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
			args, f.features, f.control, f.filter, f.key, f.bidirectional, f.allowZero, f.opt = parseFeatures(cmd, args[1:])
			featureset = append(featureset, f)
		case "export":
			if firstexporter == nil {
				firstexporter = args
			}
			if len(args) < 1 {
				log.Fatalln("Need an export type")
			}
			var e flows.Exporter
			args, e, err = flows.MakeExporter(name, args[2:])
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
			args, s, err = packet.MakeSource(name, args[2:])
			if err != nil {
				log.Fatalf("Error creating source '%s': %s\n", name, err)
			}
			sources.Append(s)
		case "filter":
			if len(args) < 1 {
				log.Fatalln("Need a filter type")
			}
			var s packet.Filter
			args, s, err = packet.MakeFilter(name, args[2:])
			if err != nil {
				log.Fatalf("Error creating filter '%s': %s\n", name, err)
			}
			filters = append(filters, s)
		case "label":
			if len(args) < 1 {
				log.Fatalln("Need a label type")
			}
			var s packet.Label
			args, s, err = packet.MakeLabel(name, args[2:])
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
	expireWindow := set.Bool("expireWindow", false, "Expire all flows after every window. Useful if flow key contains a window function")
	flowExpire := set.Uint("expire", 100, "Check for expired timers with this period in seconds. expire↓ ⇒ memory↓, execution time↑")
	maxPacket := set.Uint("size", 9000, "Maximum packet size handled internally. 0 = automatic")
	printStats := set.Bool("stats", false, "Output statistics")
	autoGC := set.Bool("scantFlows", false, "If you not have many flows setting this speeds up processing speed, but might cause a huge increase in memory usage.")
	sortOrderStr := set.String("sort", "stop", `Sort output by "start" time, "stop" time, "expiry" time, or unsorted ("none")
Beware: Worst case performance of start/stop sorting is O(1), while expiry is O(flow log(flow)) (with average O(flow)).
Both need an additional O(flow) merge part if multiple tables are used.
Additionally, stop might lead to very high memory usage (and longer execution times) in case one long lasting flow keeps all other flows from expiring (active/idle timeout!).`)
	verbose := set.Bool("verbose", false, "Verbose output")

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

	result, exporters, filters, sources, labels = parseCommandLine(cmd, set.Args())

	if len(result) == 0 {
		log.Fatalf("At least one exporter is needed!\n")
	}

	var recordList flows.RecordListMaker

	var key []string
	var bidirectional, allowZero bool
	var opts flows.FlowOptions
	first := true

	sortOrder, err := flows.AtoSort(*sortOrderStr)
	if err != nil {
		log.Fatalln(err)
	}

	for _, featureset := range result {
		pipeline, err := flows.MakeExportPipeline(featureset.exporter, sortOrder, *numProcessing)
		if err != nil {
			log.Fatalln(err)
		}
		for _, feature := range featureset.featureset {
			sort.Strings(feature.key)
			if first {
				first = false
				key = feature.key
				bidirectional = feature.bidirectional
				allowZero = feature.allowZero
				opts = feature.opt
			} else {
				if !reflect.DeepEqual(key, feature.key) {
					log.Fatalln("key_features of every flowspec must match")
				}
				if bidirectional != feature.bidirectional {
					log.Fatalln("bidirectional of every flow must match")
				}
				if allowZero != feature.allowZero {
					log.Fatalln("allowZero of every flow must match")
				}
				if !reflect.DeepEqual(opts, feature.opt) {
					log.Fatalln("timeouts and per packet of every flow must match")
				}
			}
			if err := recordList.AppendRecord(feature.features, feature.control, feature.filter, pipeline, *verbose); err != nil {
				log.Fatalf("Couldn't parse feature specification: %s\n", err)
			}
		}
	}

	keyselector := packet.MakeDynamicKeySelector(key, bidirectional, allowZero)

	if cmd == "callgraph" {
		recordList.CallGraph(os.Stdout)
		return
	}

	for _, exporter := range exporters {
		exporter.Init()
	}

	recordList.Init()

	flows.CleanupFeatures()
	util.CleanupModules()
	recordList.Clean()

	if !*autoGC {
		debug.SetGCPercent(10000000) //We manually call gc after timing out flows; make that optional?
	}

	opts.WindowExpiry = *expireWindow
	opts.SortOutput = sortOrder

	flowtable := packet.NewFlowTable(int(*numProcessing), recordList, packet.NewFlow, opts,
		flows.DateTimeNanoseconds(*flowExpire)*flows.SecondsInNanoseconds, keyselector, *autoGC)

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

	recordList.Flush()

	for _, exporter := range exporters {
		exporter.Finish()
	}
	if *printStats {
		engine.PrintStats(os.Stderr)
		flowtable.PrintStats(os.Stderr)
	}
}
