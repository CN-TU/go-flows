package main

import (
	"flag"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/exporters"
	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
	"pm.cn.tuwien.ac.at/ipfix/go-flows/packet"
)

var (
	format        = flag.String("format", "text", "Output format")
	output        = flag.String("output", "-", "Output filename")
	cpuprofile    = flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile    = flag.String("memprofile", "", "write mem profile to file")
	blockprofile  = flag.String("blockprofile", "", "write block profile to file")
	numProcessing = flag.Uint("n", 4, "number of parallel processing queues")
	activeTimeout = flag.Uint("active", 1800, "active timeout in seconds")
	idleTimeout   = flag.Uint("idle", 300, "idle timeout in seconds")
)

func main() {
	flag.Parse()
	var exporter flows.Exporter
	if *format == "text" {
		exporter = exporters.NewPrintExporter(*output)
	} else {
		log.Fatal("Only text output supported for now!")
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

	packets := packet.ReadFiles(flag.Args())

	flowtable := packet.NewParallelFlowTable(int(*numProcessing), func(flow flows.Flow) flows.FeatureList {
		a := &packet.SrcIP{flows.NewBaseFeature(flow)}
		b := &packet.DstIP{flows.NewBaseFeature(flow)}
		//		c := &Proto{}
		//		c.flow = flow
		features := []flows.Feature{
			//			c,
			a,
			b}
		ret := flows.NewFeatureList(features, features, features, exporter)

		return ret
	}, packet.NewFlow, flows.Time(*activeTimeout)*flows.Seconds, flows.Time(*idleTimeout)*flows.Seconds, 100*flows.Seconds)

	time := packet.ParsePacket(packets, flowtable)

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		f.Close()
	}
	flowtable.EOF(time)
	exporter.Finish()
}
