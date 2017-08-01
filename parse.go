package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
)

func decodeFeatures(dec *json.Decoder) []interface{} {
	var ret []interface{}
	for {
		t, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if delim, ok := t.(json.Delim); ok {
			switch delim {
			case '{':
				ret = append(ret, decodeFeatures(dec))
			case '}':
				return ret
			}
		} else if delim, ok := t.(json.Number); ok {
			if t, err := delim.Int64(); err == nil {
				ret = append(ret, t)
			} else if t, err := delim.Float64(); err == nil {
				ret = append(ret, t)
			} else {
				log.Fatalf("Can't decode %s!\n", delim.String())
			}
		} else {
			ret = append(ret, t)
		}
	}
	log.Fatalln("File ended prematurely while decoding Features.")
	return nil
}

func decodeJSON(inputfile, key string, id int) []interface{} {
	f, err := os.Open(inputfile)
	if err != nil {
		log.Fatalln("Can't open ", inputfile)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.UseNumber()

	level := 0
	found := false
	discovered := 0

	for {
		t, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if delim, ok := t.(json.Delim); ok {
			switch delim {
			case '{', '[':
				level++
			case '}', ']':
				level--
			}
		}
		if field, ok := t.(string); ok {
			if found && level == 3 && field == "features" {
				if discovered == id {
					return decodeFeatures(dec)
				}
				discovered++
			} else if level == 1 && field == key {
				found = true
			}
		}
	}
	return nil
}
