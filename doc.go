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

Specification

Specification files are JSON files based on the NTARC format (https://nta-meta-analysis.readthedocs.io/en/latest/).
Both version 1 and version 2 files can be used. It is also possible to use a simpler format, if a paper specification
is not needed.

V1-formated file:

	{
	  "flows": [
	    {
	      "features": [...],
	      "key": {
	        "bidirectional": <bool>|"string",
	        "key_features": [...]
	      }
	    }
	  ]
	}

Simpleformat specification:

	{
	  "features": [...],
	  "key_features": [...],
	  "bidirectional": <bool>
	}

V2-formated file:

	{
	  "version": "v2",
	  "preprocessing": {
	    "flows": [
	      <simpleformat>
	    ]
	  }
	}

key featurs give a list of features, which are used to compute a flow key. features is a formated list
of features to export. This list can also contain combinations of features and operations
(https://nta-meta-analysis.readthedocs.io/en/latest/features.html).
Only single pass operations can ever be supported due to design restrictions in the flow exporter.

In addition to the features specified in the nta-meta-analysis, two addional types of features are present:
Filter features which can exclude packets from a whole flow, and control features which can change flow
behaviour like exporting the flow before the end, restarting the flow, or discarding the flow.

A list of supported features can be queried with "./go-flows features"

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

Features most follow the conventions in https://nta-meta-analysis.readthedocs.io/en/latest/features.html,
which states that names must follow the ipfix iana assignments (https://www.iana.org/assignments/ipfix/ipfix.xhtml),
or start with an _ for common features or __ for uncommon ones. Feature names must be camelCase. The flow exporter
has the full list of ipfix iana assignments already builtin which means that for these features one needs to only
specifiy the name - all type information is automatically added by the flow extractor.

For implementing features most of the time flows.BaseFeature is a good start point. Features need to override
the needed methods:

  * Start(*EventContext) gets called when a flow starts. Do cleanup here (features might be reused!). MUST call flows.BaseFeature.Start from this function!
  * Event(interface{}, *EventContext, interface{}) gets called for every packet belonging to the current flow
  * Stop(FlowEndReason, *EventContext) gets called when a flow finishes (before export)

  * SetValue(new interface{}, when *EventContext, self interface{}) Call this one for setting a value. It stores the new value and forwards it to all dependent features.

Less commonly used functions

  * Variant() gets called to determine which variant to use, if a feature can different types depending on data (see sourceIPAddress for an example)
  * SetArguments([]Feature) gets called during instantiation if a feature expects constand arguments (see select_slice for an example)



A simple example is the protocolIdentifier:

	type protocolIdentifier struct {
		flows.BaseFeature
	}

	func (f *protocolIdentifier) Event(new interface{}, context *flows.EventContext, src interface{}) {
		if f.Value() == nil {
			f.SetValue(new.(packet.Buffer).Proto(), context, f)
		}
	}

This feature doesn't need a Start or Stop (since both functions don't provide a packet).
For every packet, it checks, if the protocolIdentifier has already been set, and if it hasn't been, it sets a new value.
The new value provided to Event will always be a packet.Buffer for features that expect a raw packet.
For other features, this will be the actual value emitted from other features. E.g. for the specification

	{"eq": ["protocolIdentifier", 17]}

the minfeature will receive the uint8 emitted by this feature.

The final component missing from the code is the feature registration. This has to be done in init with one of the
Register* functions from the flows packet. For the protocolIdentifier this looks like the following:

	func init() {
		flows.RegisterStandardFeature("protocolIdentifier", flows.FlowFeature, func() flows.Feature { return &protocolIdentifier{} }, flows.RawPacket)
	}

Since protocolIdentifier is one of the iana assigned ipfix features, RegisterStandardFeature can be used, which automatically adds the rest of the
ipfix information element specification. The second argument is what this feature implementation returns which in this case is a single value per
flow - a FlowFeature. The third argument must be a function that returns a new feature instance. The last argument specifies the input to this
features, which is a raw packet. The flows package contains a list of implemented types and Register functions.

For more examples have a look at the provided features.
*/
package main