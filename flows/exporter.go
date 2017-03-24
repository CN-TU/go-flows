package flows

type Exporter interface {
	Export([]Feature, FlowEndReason, Time)
	Fields([]string)
	Finish()
}
