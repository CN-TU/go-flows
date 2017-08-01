package main

import (
	"os"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
	_ "pm.cn.tuwien.ac.at/ipfix/go-flows/packet" //initialize features
)

func init() {
	addCommand("list", "List available features", listFeatures)
}

func listFeatures(string, []string) {
	//TODO add some kind of limit and filters
	flows.ListFeatures(os.Stdout)
}
