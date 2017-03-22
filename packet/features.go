package packet

import (
	"net"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

type SrcIP struct {
	flows.BaseFeature
}

func (f *SrcIP) Event(new interface{}, when flows.Time) {
	if f.Value() == nil {
		f.SetValue(net.IP(f.Key().SrcIP()), when)
	}
}

type DstIP struct {
	flows.BaseFeature
}

func (f *DstIP) Event(new interface{}, when flows.Time) {
	if f.Value() == nil {
		f.SetValue(net.IP(f.Key().DstIP()), when)
	}
}

type Proto struct {
	flows.BaseFeature
}

func (f *Proto) Event(new interface{}, when flows.Time) {
	if f.Value() == nil {
		f.SetValue(f.Key().Proto(), when)
	}
}
