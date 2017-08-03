package main

import (
	"fmt"
	"log"
	"os"

	_ "pm.cn.tuwien.ac.at/ipfix/go-flows/exporters" //initialize exporters
	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

func init() {
	addCommand("exporters", "List available exporters and options", listExporters)
}

func listExporters(cmd string, args []string) {
	if len(args) == 1 {
		if err := flows.ExporterHelp(args[0]); err != nil {
			log.Fatalln(err)
		} else {
			return
		}
	}
	fmt.Fprintln(os.Stderr, `List of exporters:
`)
	flows.ListExporters(os.Stdout)
	fmt.Fprintf(os.Stderr, `
To query the options of an exporter use:
  %s %s exporter
`, os.Args[0], cmd)
}
