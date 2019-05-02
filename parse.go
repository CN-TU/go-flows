package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/CN-TU/go-flows/flows"
)

func decodeOneFeature(feature interface{}) interface{} {
	switch feature := feature.(type) {
	case []interface{}:
		ret := make([]interface{}, len(feature))
		for i, elem := range feature {
			ret[i] = decodeOneFeature(elem)
		}
		return ret
	case map[string]interface{}:
		var k, v interface{}
		found := false
		for k, v = range feature {
			if !found {
				found = true
			} else {
				log.Fatalf("Only one key allowed in calls (unexpected %s)\n", k)
			}
		}
		if args, ok := v.([]interface{}); !ok {
			log.Fatalf("Call arguments must be an array (unexpected %s)\n", v)
		} else {
			return decodeOneFeature(append([]interface{}{k}, args...))
		}
	case json.Number:
		if i, err := feature.Int64(); err == nil {
			return i
		} else if f, err := feature.Float64(); err == nil {
			return f
		} else {
			log.Fatalf("Can't decode %s!\n", feature.String())
		}
	}
	return feature
}

func decodeFeatures(features interface{}) (ret []interface{}) {
	if features, ok := features.([]interface{}); ok {
		ret = make([]interface{}, len(features))
		for i, elem := range features {
			ret[i] = decodeOneFeature(elem)
		}
		return
	}
	log.Fatal("Feature list must be an array")
	return
}

type jsonType int

const (
	jsonAuto jsonType = iota
	jsonV2
	jsonSimple
)

func decodeV2(decoded featureJSONv2, id int) (features []interface{}, control, filter, key []string, bidirectional, allowZero bool, opt flows.FlowOptions) {
	flows := decoded.Preprocessing.Flows
	if id < 0 || id >= len(flows) {
		log.Fatalf("Only %d flows in the file ⇒ id must be between 0 and %d (is %d)\n", len(flows), len(flows)-1, id)
	}
	return decodeSimple(flows[id], id)
}

var requiredKeys = []string{"active_timeout", "idle_timeout", "bidirectional", "features", "key_features"}

func toStringArray(decoded featureJSONsimple, name string) []string {
	arr, ok := decoded[name].([]interface{})
	if !ok {
		log.Fatalf("%s must be array of strings", name)
	}
	ret := make([]string, len(arr))
	for i := range ret {
		ret[i], ok = arr[i].(string)
		if !ok {
			log.Fatalf("%s must be an array of strings", name)
		}
	}
	return ret
}

func toTimeout(decoded featureJSONsimple, name string) flows.DateTimeNanoseconds {
	val := decoded[name].(json.Number)
	if val, ok := val.Float64(); ok == nil {
		return flows.DateTimeNanoseconds(val * float64(flows.SecondsInNanoseconds))
	}
	if val, ok := val.Int64(); ok == nil {
		return flows.DateTimeNanoseconds(val) * flows.SecondsInNanoseconds
	}
	log.Fatalf("%s must be a number", name)
	return 0
}

func decodeSimple(decoded featureJSONsimple, _ int) (features []interface{}, control, filter, key []string, bidirectional, allowZero bool, opt flows.FlowOptions) {
	// Check if we have every required value
	for _, val := range requiredKeys {
		if _, ok := decoded[val]; !ok {
			log.Fatalf("Key %s is required in the flow description, but missing", val)
		}
	}
	features = decodeFeatures(decoded["features"])
	key = toStringArray(decoded, "key_features")
	bidirectional, ok := decoded["bidirectional"].(bool)
	if !ok {
		log.Fatalf("bidirectional must be a boolean")
	}

	if _, ok := decoded["_control_features"]; ok {
		control = toStringArray(decoded, "_control_features")
	}
	if _, ok := decoded["_filter_features"]; ok {
		filter = toStringArray(decoded, "_filter_features")
	}

	if zero, ok := decoded["_allow_zero"]; ok {
		if val, ok := zero.(bool); ok {
			allowZero = val
		} else {
			log.Fatal("_allow_zero must be a boolean")
		}
	}

	if packet, ok := decoded["_per_packet"]; ok {
		if val, ok := packet.(bool); ok {
			opt.PerPacket = val
		} else {
			log.Fatal("_per_packet must be a boolean")
		}
	}

	opt.ActiveTimeout = toTimeout(decoded, "active_timeout")
	opt.IdleTimeout = toTimeout(decoded, "idle_timeout")

	opt.TCPExpiry = true

	if expiry, ok := decoded["_expire_TCP"]; ok {
		if val, ok := expiry.(bool); ok {
			opt.TCPExpiry = val
		} else {
			log.Fatal("_expire_TCP must be a boolean")
		}
	}

	opt.CustomSettings = decoded

	return
}

/*	simple format:
	{
		"active_timeout": <Number>,
		"idle_timeout": <Number>,
		"bidirectional": <bool>,
		"features": [...],
		"key_features": [...],
		"_control_features": [...],
		"_filter_features": [...],
		"_per_packet": <bool>,
		"_allow_zero": <bool>,
		"_expire_TCP": <bool>
	}

	timeouts, features, key_features and bidirectional are required
	_per_packet, _allow_zero are assumed false if missing
	_expire_TCP is assumed true if missing (tcp expire works only if at least the five tuple is present in the key)
	further keys can be queried from features
*/

type featureJSONsimple map[string]interface{}

/*struct {
	Features      interface{}
	Control       []string `json:"_control_features"`
	Filter        []string `json:"_filter_features"`
	Bidirectional bool
	Key           []string `json:"key_features"`
}*/

/*	v2 format:
	{
		"version": "v2",
		"preprocessing": {
			"flows": [
				<simpleformat>
			]
		}
	}
*/

type featureJSONv2 struct {
	Version       string
	Preprocessing struct {
		Flows []featureJSONsimple
	}
}

func decodeJSON(inputfile string, format jsonType, id int) (features []interface{}, control, filter, key []string, bidirectional, allowZero bool, opt flows.FlowOptions) {
	f, err := os.Open(inputfile)
	if err != nil {
		log.Fatalln("Can't open ", inputfile)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.UseNumber()

	switch format {
	case jsonV2:
		var decoded featureJSONv2
		if err := dec.Decode(&decoded); err != nil {
			log.Fatalln("Couldn' parse feature spec:", err)
		}
		return decodeV2(decoded, id)
	case jsonSimple:
		var decoded featureJSONsimple
		if err := dec.Decode(&decoded); err != nil {
			log.Fatalln("Couldn' parse feature spec:", err)
		}
		return decodeSimple(decoded, id)
	case jsonAuto:
		//first see if we have a version in the file
		var decoded featureJSONv2
		if err := dec.Decode(&decoded); err != nil {
			log.Fatalln("Couldn' parse feature spec:", err)
		}
		if decoded.Version != "" {
			if strings.HasPrefix(decoded.Version, "v2") {
				return decodeV2(decoded, id)
			}
			log.Fatalf("Unknown file format version '%s'\n", decoded.Version)
		}
		f.Seek(0, 0)
		var decodedSimple featureJSONsimple
		if err := dec.Decode(&decodedSimple); err != nil {
			log.Fatalln("Couldn' parse feature spec:", err)
		}
		//should be simple - or something we don't know
		return decodeSimple(decodedSimple, id)
	}
	panic("Unknown format specification")
}
