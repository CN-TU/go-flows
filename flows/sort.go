package flows

// SortType specifies output sorting order
type SortType int

const (
	// SortTypeNone does no sorting (output order will be indeterministic)
	SortTypeNone SortType = iota
	// SortTypeStartTime sorts flows by packet number of the first packet in the flow
	SortTypeStartTime
	// SortTypeStopTime sorts flows by packet number of the last packet in the flow
	SortTypeStopTime
	// SortTypeExportTime sorts flows by export time and in case of ties packet number of the last packet in the flow; last packet in case of eof
	SortTypeExportTime
)
