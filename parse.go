package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
)

func decodeOne(feature interface{}) interface{} {
	switch feature := feature.(type) {
	case []interface{}:
		ret := make([]interface{}, len(feature))
		for i, elem := range feature {
			ret[i] = decodeOne(elem)
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
			return decodeOne(append([]interface{}{k}, args...))
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
			ret[i] = decodeOne(elem)
		}
		return
	}
	log.Fatal("Feature list must be an array")
	return
}

func decodeKey(dec *json.Decoder) []string {
	var ret []string

	if err := dec.Decode(&ret); err != nil {
		log.Fatalln("Flow key must be a list of strings")
	}
	return ret
}

func decodeBidirectional(dec *json.Decoder) bool {
	if t, ok := fetchToken(dec, " bidirectional").(bool); ok {
		return t
	}
	log.Fatalln("bidirectional must be of type bool")
	return false
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

/*
	v1 format:
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
*/

func decodeV1(decoded map[string]interface{}, id int) (features []interface{}, key []string, bidirectional bool) {
	if flows, ok := decoded["flows"].([]interface{}); ok {
		if len(flows) <= id {
			log.Fatalf("Only %d flows in the file - can't use flow with id %d\n", len(flows), id)
		}
		if flow, ok := flows[id].(map[string]interface{}); ok {
			if featuresJSON, ok := flow["features"]; ok {
				features = decodeFeatures(featuresJSON)
			} else {
				log.Fatalln("features tag missing from features file")
			}
			if keyObject, ok := flow["key"].(map[string]interface{}); ok {
				if keyJSON, ok := keyObject["key_features"]; ok {
					if keyInterfaces, ok := keyJSON.([]interface{}); ok {
						key = make([]string, len(keyInterfaces))
						for i, elem := range keyInterfaces {
							if elem, ok := elem.(string); ok {
								key[i] = elem
							} else {
								log.Fatalln("Key specification must be array of strings")
							}
						}
					} else {
						log.Fatalln("Key specification must be array of strings")
					}
				} else {
					log.Fatalln("Key tag missing from features file")
				}
				if bidirectionalJSON, ok := keyObject["bidirectional"]; ok {
					if bidirectionalBool, ok := bidirectionalJSON.(bool); ok {
						bidirectional = bidirectionalBool
					} else {
						log.Fatalln("Bidirectional must be boolean")
					}
				} else {
					log.Fatalln("Bidirectional missing from specification")
				}
			} else {
				log.Fatalln("key must be object")
			}
			return
		}
		log.Fatalln("Flowspec must be JSON object")
	}
	log.Fatalln("flows must be list of objects")
	return
}

/*
	v2 format:
	{
		"version": "v2",
		"preprocessing": {
			"flows": [
				{
					"features": [...],
					"key_features": [...],
					"bidirectional": <bool>
				}
			]
		}
	}

	A single flowspec is bascially in simple format (look below)

*/

func decodeV2(decoded map[string]interface{}, id int) (features []interface{}, key []string, bidirectional bool) {
	if preprocessing, ok := decoded["preprocessing"].(map[string]interface{}); ok {
		if flows, ok := preprocessing["flows"].([]interface{}); ok {
			if len(flows) <= id {
				log.Fatalf("Only %d flows in the file - can't use flow with id %d\n", len(flows), id)
			}
			if flow, ok := flows[id].(map[string]interface{}); ok {
				return decodeSimple(flow, id)
			}
			log.Fatalln("Flowspec must be JSON object")
		}
		log.Fatalln("flows must be list of objects")
	}
	log.Fatalln("preprocessing must be object")
	return
}

/*
	simple format:
	{
		"features": [...],
		"key_features": [...],
		"bidirectional": <bool>
	}
*/

func decodeSimple(decoded map[string]interface{}, _ int) (features []interface{}, key []string, bidirectional bool) {
	if featuresJSON, ok := decoded["features"]; ok {
		features = decodeFeatures(featuresJSON)
	} else {
		log.Fatalln("features tag missing from features file")
	}
	if keyJSON, ok := decoded["key_features"]; ok {
		if keyInterfaces, ok := keyJSON.([]interface{}); ok {
			key = make([]string, len(keyInterfaces))
			for i, elem := range keyInterfaces {
				if elem, ok := elem.(string); ok {
					key[i] = elem
				} else {
					log.Fatalln("Key specification must be array of strings")
				}
			}
		} else {
			log.Fatalln("Key specification must be array of strings")
		}
	} else {
		log.Fatalln("Key tag missing from features file")
	}
	if bidirectionalJSON, ok := decoded["bidirectional"]; ok {
		if bidirectionalBool, ok := bidirectionalJSON.(bool); ok {
			bidirectional = bidirectionalBool
		} else {
			log.Fatalln("Bidirectional must be boolean")
		}
	} else {
		log.Fatalln("Bidirectional missing from specification")
	}
	return
}

func decodeJSON(inputfile string, format jsonType, id int) (features []interface{}, key []string, bidirectional bool) {
	f, err := os.Open(inputfile)
	if err != nil {
		log.Fatalln("Can't open ", inputfile)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.UseNumber()

	decoded := make(map[string]interface{})

	if err := dec.Decode(&decoded); err != nil {
		log.Fatalln("Couldn' parse feature spec:", err)
	}

	switch format {
	case jsonV1:
		return decodeV1(decoded, id)
	case jsonV2:
		return decodeV2(decoded, id)
	case jsonSimple:
		return decodeSimple(decoded, id)
	case jsonAuto:
		//first see if we have a version in the file
		if version, ok := decoded["version"]; ok {
			if version == "v2" {
				return decodeV2(decoded, id)
			}
			log.Fatalf("Unknown file format version '%s'\n", version)
		}
		//no :( -> could be v1 or simple
		if _, ok := decoded["flows"]; ok {
			return decodeV1(decoded, id)
		}
		//should be simple - or something we don't know
		return decodeSimple(decoded, id)
	}
	panic("Unknown format specification")
}
