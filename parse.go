package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"
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
	jsonV1
	jsonV2
	jsonSimple
)

func fetchToken(dec *json.Decoder, fun string) json.Token {
	t, err := dec.Token()
	if err == io.EOF {
		log.Fatalf("File ended prematurely while decoding%s.\n", fun)
	}
	if err != nil {
		log.Fatalln(err)
	}
	return t
}

func decodeV1(decoded featureJSONv1, id int) (features []interface{}, control, filter, key []string, bidirectional bool) {
	flows := decoded.Flows
	if id < 0 || id >= len(flows) {
		log.Fatalf("Only %d flows in the file ⇒ id must be between 0 and %d (is %d)\n", len(flows), len(flows)-1, id)
	}
	flow := flows[id]
	features = decodeFeatures(flow.Features)
	key = flow.Key.Key
	bidirectional = flow.Key.Bidirectional
	control = flow.Control
	filter = flow.Filter
	return
}

func decodeV2(decoded featureJSONv2, id int) (features []interface{}, control, filter, key []string, bidirectional bool) {
	flows := decoded.Preprocessing.Flows
	if id < 0 || id >= len(flows) {
		log.Fatalf("Only %d flows in the file ⇒ id must be between 0 and %d (is %d)\n", len(flows), len(flows)-1, id)
	}
	return decodeSimple(flows[id], id)
}

func decodeSimple(decoded featureJSONsimple, _ int) (features []interface{}, control, filter, key []string, bidirectional bool) {
	features = decodeFeatures(decoded.Features)
	key = decoded.Key
	bidirectional = decoded.Bidirectional
	control = decoded.Control
	filter = decoded.Filter
	return
}

/*	simple format:
	{
		"features": [...],
		"key_features": [...],
		"_control_features": [...],
		"_filter_features": [...],
		"bidirectional": <bool>
	}
*/

type featureJSONsimple struct {
	Features      interface{}
	Control       []string `json:"_control_features"`
	Filter        []string `json:"_filter_features"`
	Bidirectional bool
	Key           []string `json:"key_features"`
}

/*	v1 format:
	{
		"flows": [
			{
				"features": [...],
				"_control_features": [...],
				"_filter_features": [...],
				"key": {
					"bidirectional": <bool>|"string",
					"key_features": [...]
				}
			}
		]
	}
*/

type featureJSONv1 struct {
	Flows []struct {
		Features interface{}
		Control  []string `json:"_control_features"`
		Filter   []string `json:"_filter_features"`
		Key      struct {
			Bidirectional bool
			Key           []string `json:"key_features"`
		}
	}
}

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

func decodeJSON(inputfile string, format jsonType, id int) (features []interface{}, control, filter, key []string, bidirectional bool) {
	f, err := os.Open(inputfile)
	if err != nil {
		log.Fatalln("Can't open ", inputfile)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.UseNumber()

	var decoded struct {
		featureJSONv1
		featureJSONv2
		featureJSONsimple
	}

	if err := dec.Decode(&decoded); err != nil {
		log.Fatalln("Couldn' parse feature spec:", err)
	}

	switch format {
	case jsonV1:
		return decodeV1(decoded.featureJSONv1, id)
	case jsonV2:
		return decodeV2(decoded.featureJSONv2, id)
	case jsonSimple:
		return decodeSimple(decoded.featureJSONsimple, id)
	case jsonAuto:
		//first see if we have a version in the file
		if decoded.Version != "" {
			if strings.HasPrefix(decoded.Version, "v2") {
				return decodeV2(decoded.featureJSONv2, id)
			}
			log.Fatalf("Unknown file format version '%s'\n", decoded.Version)
		}
		//no :( -> could be v1 or simple
		if decoded.Flows != nil {
			return decodeV1(decoded.featureJSONv1, id)
		}
		//should be simple - or something we don't know
		return decodeSimple(decoded.featureJSONsimple, id)
	}
	panic("Unknown format specification")
}
