package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/exporters"
	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
	"pm.cn.tuwien.ac.at/ipfix/go-flows/packet"
)

var (
	format        = flag.String("format", "text", "Output format (text|csv)")
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

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		usage()
	}
	var exporter flows.Exporter
	switch *format {
	case "text":
		exporter = exporters.NewPrintExporter(*output)
	case "csv":
		exporter = exporters.NewCSVExporter(*output)
	default:
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

	features := []interface{}{
		"sourceIPAddress",
		"destinationIPAddress",
		"protocolIdentifier",
		"flowEndReason",
		"flowEndNanoSeconds",
	}

	flowtable := packet.NewParallelFlowTable(int(*numProcessing), flows.NewFeatureListCreator(features, exporter), packet.NewFlow, flows.Time(*activeTimeout)*flows.Seconds, flows.Time(*idleTimeout)*flows.Seconds, 100*flows.Seconds)

	time := packet.ReadFiles(flag.Args(), int(*maxPacket), flowtable)

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		f.Close()
	}
	flowtable.EOF(time)
	//_ = time
	exporter.Finish()
}
