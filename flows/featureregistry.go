package flows

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/CN-TU/go-ipfix"
)

// TypeResolver is a resolution function. It must return an ipfix information element for a givent list of feature argument types.
type TypeResolver func([]ipfix.InformationElement) ipfix.InformationElement

// MakeFeature is a function that returns an instantiated Feature
type MakeFeature func() Feature

// featureMake is used internally to hold information about how to create a specific feature and the needed metadata
type featureMaker struct {
	ret       FeatureType
	make      MakeFeature
	arguments []FeatureType
	ie        ipfix.InformationElement
	variants  []ipfix.InformationElement
	resolver  TypeResolver
	function  bool
}

func (f featureMaker) String() string {
	return fmt.Sprintf("<%s>%s(%s)", f.ret, f.ie, f.arguments)
}

// getArguments returns the argument types needed for a given return type and number of arguments of a single feature
func (f featureMaker) getArguments(ret FeatureType, nargs int) []FeatureType {
	if f.arguments[len(f.arguments)-1] == Ellipsis {
		r := make([]FeatureType, nargs)
		last := len(f.arguments) - 2
		variadic := f.arguments[last]
		if variadic == MatchType {
			variadic = ret
		}
		for i := 0; i < nargs; i++ {
			if i > last {
				r[i] = variadic
			} else {
				if f.arguments[i] == MatchType {
					r[i] = ret
				} else {
					r[i] = f.arguments[i]
				}
			}
		}
		return r
	}
	if f.ret == MatchType {
		r := make([]FeatureType, nargs)
		for i := range r {
			if f.arguments[i] == MatchType {
				r[i] = ret
			} else {
				r[i] = f.arguments[i]
			}
		}
		return r
	}
	return f.arguments
}

var featureRegistry = make([]map[string][]featureMaker, featureTypeMax) // variable holding all registered features
var compositeFeatures = make(map[string]compositeFeatureMaker)          // variable holding all registered composite features

func init() {
	for i := range featureRegistry {
		featureRegistry[i] = make(map[string][]featureMaker)
	}
	ipfix.LoadIANASpec() //for RegisterStandardFeature
}

// RegisterFeature registers a new feature with the given IE.
func RegisterFeature(ie ipfix.InformationElement, ret FeatureType, make MakeFeature, arguments ...FeatureType) {
	featureRegistry[ret][ie.Name] = append(featureRegistry[ret][ie.Name],
		featureMaker{
			ret:       ret,
			make:      make,
			arguments: arguments,
			ie:        ie,
		})
}

// RegisterFunction registers a function (feature with arguments - e.g. min()), which data type can
// be resolved from the arguments
//
// This is the case for one-argument functions, where return type (e.g. min()) is the same as argument
// type, and n-argument functions, where the return type is the maximum numeric type resolved from the
// arguments (e.g. add)
func RegisterFunction(name string, ret FeatureType, make MakeFeature, arguments ...FeatureType) {
	ie := ipfix.NewInformationElement(name, 0, 0, ipfix.IllegalType, 0)
	featureRegistry[ret][ie.Name] = append(featureRegistry[ret][ie.Name],
		featureMaker{
			ret:       ret,
			make:      make,
			arguments: arguments,
			ie:        ie,
			function:  true,
		})
}

// RegisterTypedFunction registers a function that has a specific return type.
func RegisterTypedFunction(name string, t ipfix.Type, tl uint16, ret FeatureType, make MakeFeature, arguments ...FeatureType) {
	ie := ipfix.NewInformationElement(name, 0, 0, t, tl)
	featureRegistry[ret][ie.Name] = append(featureRegistry[ret][ie.Name],
		featureMaker{
			ret:       ret,
			make:      make,
			arguments: arguments,
			ie:        ie,
			function:  true,
		})
}

// RegisterCustomFunction registers a function that needs custom type resolution to get the return type
func RegisterCustomFunction(name string, resolver TypeResolver, ret FeatureType, make MakeFeature, arguments ...FeatureType) {
	featureRegistry[ret][name] = append(featureRegistry[ret][name],
		featureMaker{
			ret:       ret,
			make:      make,
			arguments: arguments,
			resolver:  resolver,
			function:  true,
		})
}

// RegisterVariantFeature registers a feature that represents more than one information element depending on the data (e.g. sourceIpv4Address/sourceIpv6Address)
func RegisterVariantFeature(name string, ies []ipfix.InformationElement, ret FeatureType, make MakeFeature, arguments ...FeatureType) {
	ie := ipfix.NewInformationElement(name, 0, 0, ipfix.IllegalType, 0)
	featureRegistry[ret][ie.Name] = append(featureRegistry[ret][ie.Name],
		featureMaker{
			ret:       ret,
			make:      make,
			arguments: arguments,
			ie:        ie,
			variants:  ies,
		})
}

// RegisterStandardFeature registers a feature from the iana ipfix list
func RegisterStandardFeature(name string, ret FeatureType, make MakeFeature, arguments ...FeatureType) {
	ie := ipfix.GetInformationElement(name)
	RegisterFeature(ie, ret, make, arguments...)
}

// RegisterTemporaryFeature registers a feature that is not part of the iana ipfix list. It gets assigned a number upon exporting.
func RegisterTemporaryFeature(name string, t ipfix.Type, tl uint16, ret FeatureType, make MakeFeature, arguments ...FeatureType) {
	ie := ipfix.NewInformationElement(name, 0, 0, t, tl)
	RegisterFeature(ie, ret, make, arguments...)
}

// RegisterControlFeature registers a control feature (i.e. a feature that can manipulate flow behaviour)
func RegisterControlFeature(name string, make MakeFeature) {
	RegisterFunction(name, ControlFeature, make, RawPacket)
}

// RegisterFilterFeature registers a filter feature (i.e. a feature that can skip events for a flow)
func RegisterFilterFeature(name string, make MakeFeature) {
	RegisterFunction(name, RawPacket, make, RawPacket)
}

// RegisterCompositeFeature registers a new composite feature with the given name. Composite features are features that depend on other features and need to be
// represented in the form ["featurea", ["featureb", "featurec"]]
func RegisterCompositeFeature(ie ipfix.InformationElement, definition ...interface{}) {
	if _, ok := compositeFeatures[ie.Name]; ok {
		panic(fmt.Sprintf("Feature %s already registered", ie.Name))
	}
	compositeFeatures[ie.Name] = compositeFeatureMaker{
		definition: definition,
		ie:         ie,
	}
}

// RegisterStandardCompositeFeature registers a composite feature (see RegisterCompositeFeature) that is part of the iana ipfix list
func RegisterStandardCompositeFeature(name string, definition ...interface{}) {
	ie := ipfix.GetInformationElement(name)
	RegisterCompositeFeature(ie, definition...)
}

// RegisterTemporaryCompositeFeature registers a composite feature (see RegisterCompositeFeature) that is not part of the iana ipfix list
func RegisterTemporaryCompositeFeature(name string, t ipfix.Type, tl uint16, definition ...interface{}) {
	ie := ipfix.NewInformationElement(name, 0, 0, t, tl)
	RegisterCompositeFeature(ie, definition...)
}

// getFeature returns the feature metadata for a feature with the given name, return type, and number of arguments, and true if such a feature exists
func getFeature(feature string, ret FeatureType, nargs int) (featureMaker, bool) {
	variadicFound := false
	var variadic featureMaker
	for _, t := range []FeatureType{ret, MatchType} {
		for _, f := range featureRegistry[t][feature] {
			if len(f.arguments) >= 1 && f.arguments[len(f.arguments)-1] == Ellipsis {
				variadicFound = true
				variadic = f
			} else if len(f.arguments) == nargs {
				return f, true
			}
		}
	}
	if variadicFound {
		return variadic, true
	}
	return featureMaker{}, false
}

// ListFeatures creates a table of available features and outputs it to w.
func ListFeatures(w io.Writer) {
	t := tabwriter.NewWriter(w, 0, 1, 1, ' ', 0)
	pf := make(map[string]string)
	ff := make(map[string]string)
	args := make(map[string]string)
	impl := make(map[string]string)
	var base, functions, filters, control []string
	for ret, features := range featureRegistry {
		for name, featurelist := range features {
			for _, feature := range featurelist {
				if feature.ret == ControlFeature {
					control = append(control, name)
				} else if feature.ret == RawPacket || feature.ret == RawFlow {
					filters = append(filters, name)
					tmp := make([]string, len(feature.arguments))
					for i := range feature.arguments {
						switch feature.arguments[i] {
						case RawFlow, FlowFeature:
							tmp[i] = "F"
						case RawPacket, PacketFeature:
							tmp[i] = "P"
						case Ellipsis:
							tmp[i] = "..."
						case MatchType:
							tmp[i] = "X"
						case Selection:
							tmp[i] = "S"
						case Const:
							tmp[i] = "C"
						}
					}
					args[name] = strings.Join(tmp, ",")
				} else if len(feature.arguments) == 1 &&
					(feature.arguments[0] == RawPacket || feature.arguments[0] == RawFlow) {
					base = append(base, name)
				} else {
					tmp := make([]string, len(feature.arguments))
					for i := range feature.arguments {
						switch feature.arguments[i] {
						case FlowFeature:
							tmp[i] = "F"
						case PacketFeature:
							tmp[i] = "P"
						case Ellipsis:
							tmp[i] = "..."
						case MatchType:
							tmp[i] = "X"
						case Selection:
							tmp[i] = "S"
						case Const:
							tmp[i] = "C"
						}
					}
					args[name] = strings.Join(tmp, ",")
					functions = append(functions, name)
				}
				switch FeatureType(ret) {
				case RawPacket, PacketFeature:
					pf[name] = "X"
				case RawFlow, FlowFeature:
					ff[name] = "X"
				case MatchType:
					pf[name] = "X"
					ff[name] = "X"
				}

			}
		}
	}
	for name, feature := range compositeFeatures {
		impl[name] = fmt.Sprint(" = ", strings.Join(compositeToCall(feature.definition), ""))
		fun := feature.definition[0].(string)
		if _, ok := featureRegistry[FlowFeature][fun]; ok {
			ff[name] = "X"
		}
		if _, ok := featureRegistry[PacketFeature][fun]; ok {
			pf[name] = "X"
		}
		if _, ok := featureRegistry[MatchType][fun]; ok {
			ff[name] = "X"
			pf[name] = "X"
		}
		base = append(base, name)
	}
	sort.Strings(base)
	sort.Strings(functions)
	sort.Strings(filters)
	fmt.Fprintln(w, "P ... Packet Feature")
	fmt.Fprintln(w, "F ... Flow Feature")
	fmt.Fprintln(w, "S ... Selection")
	fmt.Fprintln(w, "C ... Constant")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Base Features:")
	fmt.Fprintln(w, "  P F Name")
	var last string
	for _, name := range base {
		if name == last {
			continue
		}
		last = name
		line := new(bytes.Buffer)
		fmt.Fprintf(line, "  %1s\t%1s\t%s%s\n", pf[name], ff[name], name, impl[name])
		t.Write(line.Bytes())
	}
	t.Flush()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Functions:")
	fmt.Fprintln(w, "  P F Name")
	for _, name := range functions {
		if name == last {
			continue
		}
		last = name
		line := new(bytes.Buffer)
		fmt.Fprintf(line, "  %1s\t%1s\t%s(%s)\n", pf[name], ff[name], name, args[name])
		t.Write(line.Bytes())
	}
	t.Flush()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Filters:")
	fmt.Fprintln(w, "  P F Name")
	for _, name := range filters {
		if name == last {
			continue
		}
		last = name
		line := new(bytes.Buffer)
		fmt.Fprintf(line, "  %1s\t%1s\t%s(%s)\n", pf[name], ff[name], name, args[name])
		t.Write(line.Bytes())
	}
	t.Flush()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Control:")
	fmt.Fprintln(w, "      Name")
	for _, name := range control {
		if name == last {
			continue
		}
		last = name
		line := new(bytes.Buffer)
		fmt.Fprintf(line, "  %1s\t%1s\t%s\n", " ", " ", name)
		t.Write(line.Bytes())
	}
	t.Flush()
}

// CleanupFeatures deletes _all_ feature definitions for conserving memory. Call this after you've finished creating all feature lists with NewFeatureListCreator.
func CleanupFeatures() {
	featureRegistry = nil
	compositeFeatures = nil
}
