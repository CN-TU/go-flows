package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/packet"
	"github.com/CN-TU/go-flows/util"
)

type moduleDefinition struct {
	name, arghelp string
	help          func(string) error
	list          func() ([]util.ModuleDescription, error)
}

var modules = []moduleDefinition{
	moduleDefinition{
		"exporter", "List available exporters and options",
		flows.ExporterHelp,
		flows.ListExporters,
	},
	moduleDefinition{
		"source", "List available packet sources and options",
		packet.SourceHelp,
		packet.ListSources,
	},
	moduleDefinition{
		"filter", "List available packet filters and options",
		packet.FilterHelp,
		packet.ListFilters,
	},
	moduleDefinition{
		"label", "List available packet labels and options",
		packet.LabelHelp,
		packet.ListLabels,
	},
}

func init() {
	for _, def := range modules {
		addCommand(def.name+"s", def.arghelp, func(def moduleDefinition) func(string, []string) {
			return func(cmd string, args []string) {
				if len(args) == 1 {
					if err := def.help(args[0]); err != nil {
						log.Fatalln(err)
					} else {
						return
					}
				}

				descs, err := def.list()
				if err != nil {
					fmt.Fprintf(os.Stderr, "No %ss registered.\n", def.name)
					return
				}
				fmt.Fprintf(os.Stderr, "List of %ss:\n\n", def.name)

				t := tabwriter.NewWriter(os.Stderr, 3, 4, 5, ' ', 0)
				for _, desc := range descs {
					line := new(bytes.Buffer)
					fmt.Fprintf(line, "%s\t%s\n", desc.Name(), desc.Description())
					t.Write(line.Bytes())
				}
				t.Flush()

				fmt.Fprintf(os.Stderr, "\nTo query the options of a %s use:\n%s %s <%s>\n", def.name, os.Args[0], cmd, def.name)
			}
		}(def))
	}
}
