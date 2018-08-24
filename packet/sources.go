package packet

import (
	"io"
	"sync/atomic"

	"github.com/CN-TU/go-flows/util"
	"github.com/google/gopacket"
)

const sourceName = "source"

// Source represents a generic packet source
type Source interface {
	util.Module
	// ReadPacket reads the next packet from the source.
	// Must return layertype of base layer, binary data, capture info, skipped packets, filtered packets, error
	ReadPacket() (lt gopacket.LayerType, data []byte, ci gopacket.CaptureInfo, skipped uint64, filtered uint64, err error)
	// Stop shuts down the source
	Stop()
}

// Sources holds a collection of sources that are queried one after another
type Sources struct {
	stopped uint64
	sources []Source
}

// Append adds source to this source-collection
func (s *Sources) Append(a Source) {
	s.sources = append(s.sources, a)
}

// ReadPacket reads a single packet from the current packet source. In case the current source is empty, it switches to the next one.
func (s *Sources) ReadPacket() (lt gopacket.LayerType, data []byte, ci gopacket.CaptureInfo, skipped uint64, filtered uint64, err error) {
	for {
		lt, data, ci, skipped, filtered, err = s.sources[0].ReadPacket()
		if err == nil || err != io.EOF {
			return
		}
		if atomic.LoadUint64(&s.stopped) == 1 {
			err = io.EOF
			return
		}
		s.sources[0].Stop()
		if len(s.sources) == 1 {
			return
		}
		s.sources = s.sources[1:]
	}
}

// Stop all packet sources
func (s *Sources) Stop() {
	atomic.StoreUint64(&s.stopped, 1)
	s.sources[0].Stop()
}

// RegisterSource registers an source (see module system in util)
func RegisterSource(name, desc string, new util.ModuleCreator, help util.ModuleHelp) {
	util.RegisterModule(sourceName, name, desc, new, help)
}

// SourceHelp displays help for a specific source (see module system in util)
func SourceHelp(which string) error {
	return util.GetModuleHelp(sourceName, which)
}

// MakeSource creates an source instance (see module system in util)
func MakeSource(which string, args []string) ([]string, Source, error) {
	args, module, err := util.CreateModule(sourceName, which, args)
	if err != nil {
		return args, nil, err
	}
	return args, module.(Source), nil
}

// ListSources returns a list of sources (see module system in util)
func ListSources() ([]util.ModuleDescription, error) {
	return util.GetModules(sourceName)
}
