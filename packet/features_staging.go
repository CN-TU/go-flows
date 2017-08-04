package packet

import (
	"bytes"
	"strings"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

/*

Features in here are subject to change. Use them with caution.

*/

////////////////////////////////////////////////////////////////////////////////

type _characters struct {
	flows.BaseFeature
	time int64
	src  []byte
}

func (f *_characters) Event(new interface{}, context flows.EventContext, src interface{}) {
	var time int64
	if f.time != 0 {
		time = int64(context.When) - f.time
	}
	if len(f.src) == 0 {
		tmp_src, _ := new.(PacketBuffer).NetworkLayer().NetworkFlow().Endpoints()
		f.src = tmp_src.Raw()
	}
	f.time = int64(context.When)
	new_time := int(float64(time) / 100000000.) // time is now in deciseconds

	srcIP, _ := new.(PacketBuffer).NetworkLayer().NetworkFlow().Endpoints()

	var buffer bytes.Buffer
	if bytes.Equal(f.src, srcIP.Raw()) {
		buffer.WriteString("A")
	} else {
		buffer.WriteString("B")
	}

	buffer.WriteString(strings.Repeat("_", new_time))

	f.SetValue(buffer.String(), context, f)
}

func init() {
	flows.RegisterFeature("_characters", []flows.FeatureCreator{
		{flows.FeatureTypePacket, func() flows.Feature { return &_characters{} }, []flows.FeatureType{flows.RawPacket}},
	})
}

////////////////////////////////////////////////////////////////////////////////

type _characters2 struct {
	flows.BaseFeature
	time int64
	src  []byte
}

func (f *_characters2) Event(new interface{}, context flows.EventContext, src interface{}) {
	var time int64
	if f.time != 0 {
		time = int64(context.When) - f.time
	}
	if len(f.src) == 0 {
		tmp_src, _ := new.(PacketBuffer).NetworkLayer().NetworkFlow().Endpoints()
		f.src = tmp_src.Raw()
	}
	f.time = int64(context.When)
	new_time := int(float64(time) / 100000000.) // time is now in deciseconds

	srcIP, _ := new.(PacketBuffer).NetworkLayer().NetworkFlow().Endpoints()
	tcp := getTcp(new.(PacketBuffer))
	if tcp == nil {
		return
	}

	var buffer bytes.Buffer
	if bytes.Equal(f.src, srcIP.Raw()) {
		// A->B
		if tcp.FIN && tcp.ACK {
			buffer.WriteString("n")
		} else if tcp.FIN {
			buffer.WriteString("f")
		} else if tcp.SYN && tcp.ACK {
			buffer.WriteString("k")
		} else if tcp.SYN {
			buffer.WriteString("s")
		} else if tcp.RST && tcp.ACK {
			buffer.WriteString("t")
		} else if tcp.RST {
			buffer.WriteString("r")
		} else if tcp.PSH && tcp.ACK {
			buffer.WriteString("h")
		} else if tcp.PSH {
			buffer.WriteString("p")
		} else if tcp.ACK {
			buffer.WriteString("a")
		} else if tcp.URG {
			buffer.WriteString("u")
		} else {
			buffer.WriteString("o")
		}
	} else {
		// B->A
		if tcp.FIN && tcp.ACK {
			buffer.WriteString("N")
		} else if tcp.FIN {
			buffer.WriteString("F")
		} else if tcp.SYN && tcp.ACK {
			buffer.WriteString("K")
		} else if tcp.SYN {
			buffer.WriteString("S")
		} else if tcp.RST && tcp.ACK {
			buffer.WriteString("T")
		} else if tcp.RST {
			buffer.WriteString("R")
		} else if tcp.PSH && tcp.ACK {
			buffer.WriteString("H")
		} else if tcp.PSH {
			buffer.WriteString("P")
		} else if tcp.ACK {
			buffer.WriteString("A")
		} else if tcp.URG {
			buffer.WriteString("U")
		} else {
			buffer.WriteString("O")
		}
	}

	buffer.WriteString(strings.Repeat("-", new_time))

	f.SetValue(buffer.String(), context, f)
}

func init() {
	flows.RegisterFeature("_characters2", []flows.FeatureCreator{
		{flows.FeatureTypePacket, func() flows.Feature { return &_characters2{} }, []flows.FeatureType{flows.RawPacket}},
	})
}
