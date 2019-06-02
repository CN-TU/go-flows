package custom

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/google/gopacket/layers"

	"github.com/google/gopacket/pcapgo"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/packet"
	ipfix "github.com/CN-TU/go-ipfix"
)

type exportPackets struct {
	flows.NoopFeature
	b bytes.Buffer
	w *pcapgo.Writer
}

func (e *exportPackets) Start(context *flows.EventContext) {
	e.w = pcapgo.NewWriter(&e.b)
	e.w.WriteFileHeader(65535, layers.LinkTypeEthernet)
}

func (e *exportPackets) Event(new interface{}, context *flows.EventContext, src interface{}) {
	buf := new.(packet.Buffer)
	e.w.WritePacket(buf.Metadata().CaptureInfo, buf.Data())
}

func (e *exportPackets) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	flow := context.Flow()
	id := flow.ID()
	tid := flow.Table().ID()
	p := fmt.Sprintf("%d/%d/%d", tid, id/1000000, id/1000%1000)
	os.MkdirAll(p, 0755)
	p += fmt.Sprintf("/%d.pcap", id%1000)
	ioutil.WriteFile(p, e.b.Bytes(), 0644)
}

func init() {
	flows.RegisterTemporaryFeature("__exportPackets", "Writes one pcap per flow containing the flow's packets", ipfix.Unsigned8Type, 0, flows.FlowFeature, func() flows.Feature { return &exportPackets{} }, flows.RawPacket)
}
