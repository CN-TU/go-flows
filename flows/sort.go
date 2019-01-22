package flows

import (
	"errors"
	"strings"
)

// SortType specifies output sorting order
type SortType int

const (
	// SortTypeNone does no sorting (output order will be indeterministic)
	SortTypeNone SortType = iota
	// SortTypeStartTime sorts flows by packet number of the first packet in the flow
	SortTypeStartTime
	// SortTypeStopTime sorts flows by packet number of the last packet in the flow
	SortTypeStopTime
	// SortTypeExpiryTime sorts flows by expiry time and in case of ties packet number of the last packet in the flow; last packet in case of eof
	SortTypeExpiryTime
)

// AtoSort converts a string to a sort type
func AtoSort(s string) (SortType, error) {
	switch strings.ToLower(s) {
	case "none":
		return SortTypeNone, nil
	case "starttime", "start":
		return SortTypeStartTime, nil
	case "stoptime", "stop":
		return SortTypeStopTime, nil
	case "expirytime", "expiry":
		return SortTypeExpiryTime, nil
	}
	return 0, errors.New(`sort order must be either "none", "start", "stop", or "expiry"`)
}
