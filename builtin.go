package main

import (
	_ "github.com/CN-TU/go-flows/modules/exporters/csv"
	_ "github.com/CN-TU/go-flows/modules/exporters/ipfix"
	_ "github.com/CN-TU/go-flows/modules/exporters/kafka"
	_ "github.com/CN-TU/go-flows/modules/filters/time"
	_ "github.com/CN-TU/go-flows/modules/labels/csv"
	_ "github.com/CN-TU/go-flows/modules/sources/libpcap"
)
