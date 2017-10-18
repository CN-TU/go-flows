package flows

import "log"
import "pm.cn.tuwien.ac.at/ipfix/go-ipfix"

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

func (m *leafTemplate) subTemplate() Template {
	log.Panic("Leaf template does not have a sub template")
	return nil
}

func (m *leafTemplate) InformationElements() []ipfix.InformationElement {
	return m.ies
}

func (m *leafTemplate) ID() int {
	return m.id
}

type multiTemplate struct {
	templates []Template
}

func (m *multiTemplate) subTemplate(which int) Template {
	return m.templates[which]
}

func (m *multiTemplate) InformationElements() []ipfix.InformationElement {
	log.Panic("Multi template does not have a sub template")
	return nil
}

func (m *multiTemplate) ID() int {
	log.Panic("Multi template does not have an id")
	return 0
}
