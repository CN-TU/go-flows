package flows

import (
	"fmt"

	ipfix "github.com/CN-TU/go-ipfix"
)

type compositeFeatureMaker struct {
	definition []interface{}
	ie         ipfix.InformationElement
	iana       bool
}

// compositeToCall generates a textual representation of a composite feature (e.g. fun(arg1,arg2,..))
func compositeToCall(features []interface{}) (ret []string) {
	flen := len(features) - 1
	for i, feature := range features {
		if list, ok := feature.([]interface{}); ok {
			ret = append(ret, compositeToCall(list)...)
		} else {
			ret = append(ret, fmt.Sprint(feature))
		}
		if i == 0 {
			ret = append(ret, "(")
		} else if i < flen {
			ret = append(ret, ",")
		} else {
			ret = append(ret, ")")
		}
	}
	return
}
