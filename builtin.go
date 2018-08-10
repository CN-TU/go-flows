package main

import (
	_ "github.com/CN-TU/go-flows/modules/exporters/csv"
	_ "github.com/CN-TU/go-flows/modules/exporters/ipfix"
	_ "github.com/CN-TU/go-flows/modules/exporters/kafka"
	_ "github.com/CN-TU/go-flows/modules/features/custom"
	_ "github.com/CN-TU/go-flows/modules/features/iana"
	_ "github.com/CN-TU/go-flows/modules/features/operations"
	_ "github.com/CN-TU/go-flows/modules/features/staging"
	_ "github.com/CN-TU/go-flows/modules/filters/time"
	_ "github.com/CN-TU/go-flows/modules/labels/csv"
	_ "github.com/CN-TU/go-flows/modules/sources/libpcap"
)
