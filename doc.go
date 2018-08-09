/*
This contains a highly customizable general-purpose flow exporter.

Building

For building either "go build", "go install", or the program provided in go-flows-build can be used.
The latter allows for customizing builtin modules and can help with building modules as plugins.

Overview

This flow exporter can convert network packets into flows, extract features, and export those via
various file formats.

In general the flow exporter reads the feature definitions (i.e. a description of which features
to extract, and the flow key), and builds an execution graph (this can be viewed with the option
callgraph) from this definition.

Packets are then read from a source and processed in the following pipeline:

	source -> filter -> (parse) -> [key] -> label -> (table) -> (flow) -> (record) -> {feature} -> export

() parts in the pipeline are fixed, [] parts can be configured via the specification, {} can be configured
via the specification and provided via modules, and everything else can be provided from a module and
configured from the command line.

source is a packet source, which must provide single packets as []byte sequences and metadata like
capture time, and dropped/filtered packets. The []byte-buffer can be reused for the next packet.
For examples look at modules/sources.

filter is a packet filter, which must return true for a given packet if it should be filtered out.
For examples look at modules/filters.

parse is a fixed step that parses the packet with gopacket.

label is an optional step, that can provide an arbitrary label for every packet. For examples look at
modules/labels.

key is a fixed step that calculates the flow key. Key parameters can be configured via the specification.

table, flow, record are fixed steps that are described in more detail in the flows package.

The feature step calculates the actual feature values. Features can be provided via modules, and the
selection of which features to calculate must be provided via the specification. Features are described
in more detail in the flows package.

If a flow ends (e.g. because of timeout, or tcp-rst) it gets exported via the exported, which must
be provided as a module and configured via the command line. For examples look at modules/exporters.


The whole pipeline is executed concurrently with the following four subpipelines running concurrently:

	source -> filter

	parse -> key -> label

	n times table -> flow -> record -> feature

	export

The "table"-pipeline exists n times, where n can be configured on the command line. Packets are divided
onto the different "table"-pipelines according to flow-key.

Example usage

The examples directory contains several example flow specifications that can be used. The general
syntax on the command line is "go-flows run <commands>" where <commands> is a list of "<verb> <which>
[options] [--]" sequences. <verb> can be one of features, export, source filter, or label, and
<which> is the actual module. The options can be queried from the help of the different modules
(e.g. go-flows <verb>s <which>; e.g. go-flows exporters ipfix).

Example:

	go-flows run features examples/complex_simple.json export ipfix out.ipfix source libpcap input.pcap

Contents

The following list describes all the different things contained in the subdirectories.

 * examples: example specifications
 * flows: flows package; Contains base flow functionality, which is not dependent on packets
 * packet: packet package; Packet-part of the flow implementation
 * modules: implementation of exporters, filters, labels, sources, and features
 * util: package wit utility functions
 * go-flows-build: build script for customizing binaries and compiling plugins

Implementing Features

Todo; For now have a look at the currently implemented features and descriptions in packet and flow
packages.
*/
package main
