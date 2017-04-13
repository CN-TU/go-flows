package flows

// Exporter represents a genric exporter
type Exporter interface {
	// Export gets called upon flow export with a list of features and the export time.
	Export([]Feature, Time)
	// Fields gets called during flow list creation. A list of flow base types is supplied.
	Fields([]string)
	// Finish gets called before program exit. Eventual flushing needs to be implemented here.
	Finish()
}
