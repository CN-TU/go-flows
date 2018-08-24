package packet

import (
	"io"

	"github.com/CN-TU/go-flows/util"
)

const labelName = "label"

// Label represents a generic packet label
type Label interface {
	util.Module
	// GetLabel returns the label for the provided packet
	GetLabel(packet Buffer) (interface{}, error)
}

// Labels holds a collection of labels that are tried one after another
type Labels []Label

// GetLabel returns the label of the provided packet
func (l *Labels) GetLabel(packet Buffer) interface{} {
RETRY:
	if len(*l) == 0 {
		return nil
	}
	ret, err := (*l)[0].GetLabel(packet)
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
func MakeLabel(which string, args []string) ([]string, Label, error) {
	args, module, err := util.CreateModule(labelName, which, args)
	if err != nil {
		return args, nil, err
	}
	return args, module.(Label), nil
}

// ListLabels returns a list of labels (see module system in util)
func ListLabels() ([]util.ModuleDescription, error) {
	return util.GetModules(labelName)
}
