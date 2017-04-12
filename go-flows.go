package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"runtime/trace"
	"strings"

	"strconv"

	"encoding/json"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/exporters"
	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
	"pm.cn.tuwien.ac.at/ipfix/go-flows/packet"
)

var (
	list          = flag.Bool("list", false, "List available Features")
	callgraph     = flag.String("callgraph", "", "Write callgraph")
	format        = flag.String("format", "text", "Output format (text|csv)")
	featurefile   = flag.String("features", "", "Features specification (json)")
	selection     = flag.String("select", "flows:0", "flow selection (key:number in specification)")
	output        = flag.String("output", "-", "Output filename")
	cpuprofile    = flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile    = flag.String("memprofile", "", "write mem profile to file")
	blockprofile  = flag.String("blockprofile", "", "write block profile to file")
	tracefile     = flag.String("trace", "", "set tracing file")
	numProcessing = flag.Uint("n", 4, "number of parallel processing queues")
	activeTimeout = flag.Uint("active", 1800, "active timeout in seconds")
	idleTimeout   = flag.Uint("idle", 300, "idle timeout in seconds")
	maxPacket     = flag.Uint("size", 9000, "Maximum packet size")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [args] file1 [file2] [...] \n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(-1)
}

func decodeFeatures(dec *json.Decoder) []interface{} {
	var ret []interface{}
	for {
		t, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if delim, ok := t.(json.Delim); ok {
			switch delim {
			case '{':
				ret = append(ret, decodeFeatures(dec))
			case '}':
				return ret
			}
		} else if delim, ok := t.(json.Number); ok {
			if t, err := delim.Int64(); err == nil {
				ret = append(ret, t)
			} else if t, err := delim.Float64(); err == nil {
				ret = append(ret, t)
			} else {
				log.Fatalf("Can't decode %s!\n", delim.String())
			}
		} else {
			ret = append(ret, t)
		}
	}
	log.Fatalln("File ended prematurely while decoding Features.")
	return nil
}

func decodeJSON(inputfile, key string, id int) []interface{} {
	f, err := os.Open(inputfile)
	if err != nil {
		log.Fatalln("Can't open ", inputfile)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.UseNumber()

	level := 0
	found := false
	discovered := 0

	for {
		t, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if delim, ok := t.(json.Delim); ok {
			switch delim {
			case '{', '[':
				level++
			case '}', ']':
				level--
			}
		}
		if field, ok := t.(string); ok {
			if found && level == 3 && field == "features" {
				if discovered == id {
					return decodeFeatures(dec)
				}
				discovered++
			} else if level == 1 && field == key {
				found = true
			}
		}
	}
	return nil
}

func main() {
	flag.Parse()
	if *list {
		flows.ListFeatures()
		return
	}
	var exporter flows.Exporter
	switch *format {
	case "text":
		exporter = exporters.NewPrintExporter(*output)
	case "csv":
		exporter = exporters.NewCSVExporter(*output)
	case "none":
		exporter = nil
	default:
		usage()
	}
	if exporter != nil && flag.NArg() == 0 {
		usage()
	}
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if *blockprofile != "" {
		f, err := os.Create(*blockprofile)
		if err != nil {
			log.Fatal(err)
		}
		runtime.SetBlockProfileRate(1)
		defer pprof.Lookup("block").WriteTo(f, 0)
	}
	if *tracefile != "" {
		f, err := os.Create(*tracefile)
		if err != nil {
			log.Fatal(err)
		}
		trace.Start(f)
		defer trace.Stop()
	}

	if *featurefile == "" {
		log.Fatalln("Need a feature input file!")
	}

	selector := strings.Split(*selection, ":")
	if len(selector) != 2 {
		log.Fatalln("select must be of form 'key:id'!")
	}

	selectorID, err := strconv.Atoi(selector[1])
	if err != nil {
		log.Fatalln("select must be of form 'key:id'!")
	}

	features := decodeJSON(*featurefile, selector[0], selectorID)
	if features == nil {
		log.Fatalln("Features ", *selection, " not found in ", *featurefile)
	}

	featurelist := flows.NewFeatureListCreator(features, exporter, flows.FeatureTypeFlow)

	if *callgraph != "" {
		featurelist.CallGraph(os.Stdout)
	}

	if exporter == nil {
		return
	}

	flows.CleanupFeatures()

	debug.SetGCPercent(-1) //We manually call gc after timing out flows; make that optional?

	flowtable := packet.NewParallelFlowTable(int(*numProcessing), featurelist, packet.NewFlow, flows.Time(*activeTimeout)*flows.Seconds, flows.Time(*idleTimeout)*flows.Seconds, 100*flows.Seconds)

	time := packet.ReadFiles(flag.Args(), int(*maxPacket), flowtable)

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
	exporter.Finish()
}
