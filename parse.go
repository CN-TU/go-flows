package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
)

func decodeFeatures(dec *json.Decoder) []interface{} {
	return _decodeFeatures(dec, 0)
}

func _decodeFeatures(dec *json.Decoder, level int) []interface{} {
	var ret []interface{}
	for {
		switch t := fetchToken(dec, " features").(type) {
		case json.Delim:
			switch t {
			case '{':
				ret = append(ret, _decodeFeatures(dec, level+1))
			case '}':
				if level == 0 {
					log.Fatalln("Unexpected object end in features")
				}
				return ret
			case ']':
				if level == 0 {
					return ret
				}
			}
		case json.Number:
			if i, err := t.Int64(); err == nil {
				ret = append(ret, i)
			} else if f, err := t.Float64(); err == nil {
				ret = append(ret, f)
			} else {
				log.Fatalf("Can't decode %s!\n", t.String())
			}
		default:
			ret = append(ret, t)
		}
	}
}

func decodeKey(dec *json.Decoder) []interface{} {
	var ret []interface{}
	open := false
	for {
		switch t := fetchToken(dec, " key").(type) {
		case json.Delim:
			switch t {
			case '[':
				if !open {
					open = true
					continue
				}
				log.Fatalln("Lists are not allowed in flow key specification.")
			case '{', '}':
				log.Fatalln("Objects are not allowed in flow key specification.")
			case ']':
				return ret
			}
		case string:
			ret = append(ret, t)
		default:
			log.Fatalln("Only strings are allowed in flow key specification.")
		}
	}
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

func decodeSpec(dec *json.Decoder, format jsonType, id int) (features []interface{}, key []interface{}, bidirectional bool) {
	level := 0
	found := false
	discovered := 0
	inFlows := false
	inKey := false
	inPreprocessing := false

	bidirectional = true //default to true, if not specified

	/*
		v1 format:
		{
			"flows": [
				{
					"features": [...],
					"key": {
						"bidirectional": <bool>,
						"key_features": [...]
					}
				}
			]
		}
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
		simple format:
		{
			"features": [...],
			"key_features": [...],
			"bidirectional": <bool>
		}
	*/

	for {
		switch t := fetchToken(dec, "").(type) {
		case json.Delim:
			switch t {
			case '{', '[':
				level++
				switch format {
				case jsonV1:
					if inFlows && level == 3 {
						if discovered == id {
							found = true
						}
						discovered++
					}
				case jsonV2:
					if inFlows && level == 4 {
						if discovered == id {
							found = true
						}
						discovered++
					}
				}
			case '}', ']':
				switch format {
				case jsonV1:
					if found && level == 3 {
						return
					}
				case jsonV2:
					if found && level == 4 {
						return
					}
				case jsonSimple:
					if level == 1 {
						return
					}
				}
				level--
			}
		case string:
			switch format {
			case jsonV1:
				if found {
					if !inKey && level == 3 {
						if t == "features" {
							features = decodeFeatures(dec)
						} else if t == "key" {
							inKey = true
						}
					} else if inKey && level == 4 {
						if t == "bidirectional" {
							bidirectional = decodeBidirectional(dec)
						} else if t == "key_features" {
							key = decodeKey(dec)
						}
					}
				} else if level == 1 && t == "flows" {
					inFlows = true
				}
			case jsonV2:
				if found && level == 4 {
					switch t {
					case "features":
						features = decodeFeatures(dec)
					case "bidirectional":
						bidirectional = decodeBidirectional(dec)
					case "key_features":
						key = decodeKey(dec)
					}
				} else if inPreprocessing && level == 2 && t == "flows" {
					inFlows = true
				} else if !inPreprocessing && level == 1 && t == "preprocessing" {
					inPreprocessing = true
				}
			case jsonSimple:
				if level == 1 {
					switch t {
					case "features":
						features = decodeFeatures(dec)
					case "bidirectional":
						bidirectional = decodeBidirectional(dec)
					case "key_features":
						key = decodeKey(dec)
					}
				}
			case jsonAuto:
				/* treat it like:
				simple: if we find "features", "bidirectional", "key_features" at level 1
				v2: if we find "version": "v2" at level 1
				v1: if we find "flows" at level 1
				*/
				if level == 1 {
					switch t {
					case "features":
						features = decodeFeatures(dec)
						format = jsonSimple
					case "bidirectional":
						bidirectional = decodeBidirectional(dec)
						format = jsonSimple
					case "key_features":
						key = decodeKey(dec)
						format = jsonSimple
					case "flows":
						inFlows = true
						format = jsonV1
					case "version":
						if t, ok := fetchToken(dec, " version").(string); !ok {
							log.Fatalln("version needs to be a string")
						} else {
							if t == "v2" {
								format = jsonV2
							} else {
								log.Fatalln("Unknown version")
							}
						}
					}
				}
			}
		}
	}
}

func decodeJSON(inputfile string, format jsonType, id int) (features []interface{}, key []interface{}, bidirectional bool) {
	f, err := os.Open(inputfile)
	if err != nil {
		log.Fatalln("Can't open ", inputfile)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.UseNumber()

	return decodeSpec(dec, format, id)
}
