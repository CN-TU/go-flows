package flows

type Exporter interface {
	Export([]Feature, Time)
	Fields([]string)
	Finish()
}
