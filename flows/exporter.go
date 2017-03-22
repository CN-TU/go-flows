package flows

type Exporter interface {
	Export([]Feature, string, int64)
	Finish()
}
