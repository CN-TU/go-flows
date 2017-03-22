package flows

type Exporter interface {
	Export([]Feature, string, Time)
	Finish()
}
