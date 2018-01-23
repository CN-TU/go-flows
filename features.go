package main

import (
	"os"

	"github.com/CN-TU/go-flows/flows"
	_ "github.com/CN-TU/go-flows/packet" //initialize features
)

func init() {
	addCommand("features", "List available features", listFeatures)
}

func listFeatures(string, []string) {
	//TODO add some kind of limit and filters
	flows.ListFeatures(os.Stdout)
}
