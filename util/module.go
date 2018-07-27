package util

import (
	"fmt"
	"sort"
)

// UseStringOption is a special marker, which redirects the module to use the string options
type UseStringOption struct{}

// Module interface.
// Modules need at least an ID() function returning suitable string representation
// and an Init() function, which is called after all the arguments were successfully parsed and all the modules created.
type Module interface {
	ID() string
	Init()
}

// ModuleCreator is a function, which creates a module. It is provided a name,
// structured arguments (or UseStringOption), or a list of string options. It needs to
// return the not used string options and the created Module.
type ModuleCreator func(string, interface{}, []string) ([]string, Module, error)

// ModuleHelp is provided the name of the module and must produce a help description on stderr.
type ModuleHelp func(string)

// ModuleDescription contains name and description of a module
type ModuleDescription struct {
	name, desc string
}

// Name returns the name of this module
func (m ModuleDescription) Name() string {
	return m.name
}

// Description returns the description of this module
func (m ModuleDescription) Description() string {
	return m.desc
}

type moduleDefinition struct {
	ModuleDescription
	new  ModuleCreator
	help ModuleHelp
}

var modules = make(map[string]map[string]moduleDefinition)

// RegisterModule registers a module with given type, name, description, module creator, and help function.
// Existing modules are overwritten.
func RegisterModule(typ, name, desc string, new ModuleCreator, help ModuleHelp) {
	submodule, found := modules[typ]
	if !found {
		modules[typ] = make(map[string]moduleDefinition)
		submodule = modules[typ]
	}
	submodule[name] = moduleDefinition{
		ModuleDescription: ModuleDescription{
			name: name,
			desc: desc,
		},
		new:  new,
		help: help,
	}
}

// GetModuleHelp calls the help function of the module identified by typ, name
func GetModuleHelp(typ, name string) error {
	if submodules, ok := modules[typ]; ok {
		if module, ok := submodules[name]; ok {
			module.help(name)
			return nil
		}
	}
	return fmt.Errorf("couldn't find module of type %s with name %s", typ, name)
}

// GetModules returns the descriptions of the registered modules ordered by name
func GetModules(typ string) (descriptions []ModuleDescription, err error) {
	submodules, ok := modules[typ]
	if !ok {
		err = fmt.Errorf("no modules with typ %s registered", typ)
		return
	}
	for _, module := range submodules {
		descriptions = append(descriptions, module.ModuleDescription)
	}
	sort.Slice(descriptions, func(i, j int) bool { return descriptions[i].name < descriptions[j].name })
	return
}

// CreateModule creates the module with the given type, name, and the provided options
func CreateModule(typ, which, name string, options interface{}, args []string) ([]string, Module, error) {
	if submodules, ok := modules[typ]; ok {
		if module, ok := submodules[which]; ok {
			return module.new(name, options, args)
		}
	}
	return nil, nil, fmt.Errorf("couldn't find module of type %s with name %s", typ, which)
}
