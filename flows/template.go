package flows

import (
	"log"

	"github.com/CN-TU/go-ipfix"
)

// Template holds the information elements of a record
type Template interface {
	subTemplate(int) Template
	// InformationElements returns the list of information elements
	InformationElements() []ipfix.InformationElement
	// Unique template ID for this template
	ID() int
}

type leafTemplate struct {
	ies []ipfix.InformationElement
	id  int
}

func (m *leafTemplate) subTemplate(int) Template {
	log.Panic("Leaf template does not have a sub template")
	return nil
}

func (m *leafTemplate) InformationElements() []ipfix.InformationElement { return m.ies }
func (m *leafTemplate) ID() int                                         { return m.id }

type multiTemplate struct {
	templates []Template
}

func (m *multiTemplate) subTemplate(which int) Template { return m.templates[which] }

func (m *multiTemplate) InformationElements() []ipfix.InformationElement {
	log.Panic("Multi template does not have a sub template")
	return nil
}

func (m *multiTemplate) ID() int {
	log.Panic("Multi template does not have an id")
	return 0
}

type variant struct {
	ies   []ipfix.InformationElement
	index int
}

func makeSubTemplate(init []featureToInit, variants []variant, which []int, id *int) Template {
	if len(variants) != len(which) {
		j := len(which)
		ret := &multiTemplate{templates: make([]Template, len(variants[j].ies))}
		for i := range variants[j].ies {
			ret.templates[i] = makeSubTemplate(init, variants, append(which, i), id)
		}
		return ret
	}
	ies := make([]ipfix.InformationElement, 0, len(init))
	j := 0
	max := len(which)
	for i, feature := range init {
		if j != max && variants[j].index == i {
			ies = append(ies, variants[j].ies[which[j]])
			j++
		} else {
			ies = append(ies, feature.ie)
		}
	}
	ret := &leafTemplate{id: *id, ies: ies}
	*id++
	return ret
}

func makeTemplate(init []featureToInit, id *int) (Template, []string) {
	var variants []variant
	fields := make([]string, 0, len(init))
	for i, feature := range init {
		if feature.variant {
			variants = append(variants, variant{feature.feature.variants, i})
		}
		fields = append(fields, feature.ie.Name)
	}
	return makeSubTemplate(init, variants, []int{}, id), fields
}
