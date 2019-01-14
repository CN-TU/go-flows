package flows

import (
	"fmt"
	"strings"

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
	panic("Leaf template does not have a sub template")
}

func (m *leafTemplate) InformationElements() []ipfix.InformationElement { return m.ies }
func (m *leafTemplate) ID() int                                         { return m.id }

func (m *leafTemplate) String() string {
	ies := make([]string, len(m.ies))
	for i := range m.ies {
		ies[i] = fmt.Sprint(m.ies[i])
	}
	return fmt.Sprintf("{(%d) [%s]}", m.id, strings.Join(ies, ", "))
}

type multiTemplate struct {
	templates []Template
}

func (m *multiTemplate) subTemplate(which int) Template { return m.templates[which] }

func (m *multiTemplate) InformationElements() []ipfix.InformationElement {
	panic("Multi template does not have a sub template")
}

func (m *multiTemplate) ID() int {
	panic("Multi template does not have an id")
}

func (m *multiTemplate) String() string {
	templates := make([]string, len(m.templates))
	for i := range templates {
		templates[i] = fmt.Sprintf("%d: %s", i, m.templates[i])
	}
	return fmt.Sprintf("[%s]", strings.Join(templates, ", "))
}

type emptyTemplate struct{}

func (e *emptyTemplate) subTemplate(int) Template { panic("Impossible type combination in template") }
func (e *emptyTemplate) InformationElements() []ipfix.InformationElement {
	panic("Impossible type combination in template")
}
func (e *emptyTemplate) ID() int { panic("Impossible type combination in template") }
func (e *emptyTemplate) String() string {
	return "<illegal>"
}

func makeLeafTemplate(ies []ipfix.InformationElement, id *int) Template {
	ret := &leafTemplate{id: *id, ies: ies}
	*id++
	return ret
}

func (a *ast) makeSubTemplateRec(variants, max []int, choose map[int]int, which int, id *int) Template {
	if which == len(variants) {
		var ies []ipfix.InformationElement
		for _, fragment := range a.Fragments {
			if fragment.Export() {
				ie := fragment.Type()
				if ie == nil {
					return &emptyTemplate{}
				}
				ies = append(ies, ie.SpecificIE(choose))
			}
		}
		return makeLeafTemplate(ies, id)
	}

	var ret []Template
	for i := 0; i < max[which]; i++ {
		choose[variants[which]] = i
		ret = append(ret, a.makeSubTemplateRec(variants, max, choose, which+1, id))
	}
	return &multiTemplate{ret}
}

func (a *ast) makeSubTemplate(variants, max []int, id *int) Template {
	if len(variants) == 0 {
		var ies []ipfix.InformationElement
		for _, fragment := range a.Fragments {
			if fragment.Export() {
				ie := fragment.Type()
				if ie == nil {
					return &emptyTemplate{}
				}
				ies = append(ies, ie.IE())
			}
		}
		return makeLeafTemplate(ies, id)
	}
	return a.makeSubTemplateRec(variants, max, make(map[int]int), 0, id)
}

func (a *ast) template(id *int) (Template, []string) {
	var variants, max []int
	var fields []string
	for _, fragment := range a.Fragments {
		maker := fragment.FeatureMaker()
		if len(maker.variants) > 0 {
			variants = append(variants, fragment.Register())
			max = append(max, len(maker.variants))
		}
		if fragment.Export() {
			fields = append(fields, fragment.ExportName())
		}
	}
	return a.makeSubTemplate(variants, max, id), fields
}
