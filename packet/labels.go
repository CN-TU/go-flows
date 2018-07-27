package packet

import (
	"io"

	"github.com/CN-TU/go-flows/util"
	"github.com/google/gopacket"
)

const labelName = "label"

// Label represents a generic packet label
type Label interface {
	util.Module
	// GetLabel returns the label for the provided packet
	GetLabel(ci gopacket.CaptureInfo, packetnr uint64, data []byte) (interface{}, error)
}

// Labels holds a collection of labels that are tried one after another
type Labels []Label

// GetLabel returns the label of the provided packet
func (l *Labels) GetLabel(ci gopacket.CaptureInfo, packetnr uint64, data []byte) interface{} {
RETRY:
	if len(*l) == 0 {
		return nil
	}
	ret, err := (*l)[0].GetLabel(ci, packetnr, data)
	if err != io.EOF {
		return ret
	}
	(*l) = (*l)[1:]
	goto RETRY
}

// RegisterLabel registers an label (see module system in util)
func RegisterLabel(name, desc string, new util.ModuleCreator, help util.ModuleHelp) {
	util.RegisterModule(labelName, name, desc, new, help)
}

// LabelHelp displays help for a specific label (see module system in util)
func LabelHelp(which string) error {
	return util.GetModuleHelp(labelName, which)
}

// MakeLabel creates an label instance (see module system in util)
func MakeLabel(which, name string, options interface{}, args []string) ([]string, Label, error) {
	args, module, err := util.CreateModule(labelName, which, name, options, args)
	if err != nil {
		return args, nil, err
	}
	return args, module.(Label), nil
}

// ListLabels returns a list of labels (see module system in util)
func ListLabels() ([]util.ModuleDescription, error) {
	return util.GetModules(labelName)
}
