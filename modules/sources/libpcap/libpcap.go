package libpcap

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"

	"github.com/CN-TU/go-flows/packet"
	"github.com/CN-TU/go-flows/util"
	"github.com/google/gopacket"
)

type libpcapSource struct {
	id            string
	files         []string
	filter        string
	live          bool
	promisc       bool
	snaplen       int
	which         int
	lt            gopacket.LayerType
	currentHandle *pcap.Handle
	currentFilter *pcap.BPF
}

func (ps *libpcapSource) ID() string {
	return ps.id
}

func (ps *libpcapSource) Init() {
}

func (ps *libpcapSource) setLayerType() error {
	switch lt := ps.currentHandle.LinkType(); lt {
	case layers.LinkTypeEthernet:
		ps.lt = layers.LayerTypeEthernet
	case layers.LinkTypeRaw, layers.LinkType(12):
		ps.lt = packet.LayerTypeIPv46
	case layers.LinkTypeLinuxSLL:
		ps.lt = layers.LayerTypeLinuxSLL
	default:
		return fmt.Errorf("libpcap: unknown link type %s", lt)
	}
	return nil
}

func (ps *libpcapSource) openNext() error {
	var err error

	ps.which++
	if ps.live {
		if ps.which > 0 {
			return io.EOF
		}

		inactive, err := pcap.NewInactiveHandle(ps.files[0])
		if err != nil {
			return err
		}

		if err := inactive.SetTimeout(pcap.BlockForever); err != nil {
			return err
		}

		if ps.promisc {
			if err := inactive.SetPromisc(true); err != nil {
				return err
			}
		}

		if ps.snaplen != 0 {
			if err := inactive.SetSnapLen(ps.snaplen); err != nil {
				return err
			}
		}

		ps.currentHandle, err = inactive.Activate()
		if err != nil {
			return err
		}

		if ps.filter != "" {
			if err = ps.currentHandle.SetBPFFilter(ps.filter); err != nil {
				return err
			}
		}

		goto FINISHED
	}

	if ps.which > len(ps.files)-1 {
		return io.EOF
	}

	if ps.currentHandle != nil {
		ps.currentHandle.Close()
	}

	ps.currentHandle, err = pcap.OpenOffline(ps.files[ps.which])
	if err != nil {
		return fmt.Errorf("couldn't open file '%s': %s", ps.files[ps.which], err)
	}

	if ps.filter != "" {
		ps.currentFilter, err = ps.currentHandle.NewBPF(ps.filter)
		if err != nil {
			return err
		}
	}

FINISHED:
	return ps.setLayerType()
}

func (ps *libpcapSource) ReadPacket() (lt gopacket.LayerType, data []byte, ci gopacket.CaptureInfo, skipped uint64, filtered uint64, err error) {
	if ps.which == -1 {
		err = ps.openNext()
		if err != nil {
			return
		}
	}

RETRY:
	data, ci, err = ps.currentHandle.ZeroCopyReadPacketData()

	if err != nil {
		// report non-eof errors, but treat them as non-fatal
		if err != io.EOF {
			log.Printf("libpcap: read error in pcap file '%s': %s\n", ps.files[ps.which], err)
			skipped++
		}
		err = ps.openNext()
		if err != nil {
			return
		}
		goto RETRY
	}

	if ps.currentFilter != nil && !ps.currentFilter.Matches(ci, data) {
		filtered++
		goto RETRY
	}

	lt = ps.lt
	return
}

// Stop shuts down the source
func (ps *libpcapSource) Stop() {
	if ps.currentHandle != nil {
		ps.currentHandle.Close()
	}
}

func newLibpcapSource(name string, opts interface{}, args []string) (arguments []string, ret util.Module, err error) {
	var filter string
	var files []string
	var live bool
	var promisc bool
	var snaplen int

	if _, ok := opts.(util.UseStringOption); ok {
		set := flag.NewFlagSet("libpcap", flag.ExitOnError)
		set.Usage = func() { pcapHelp("libpcap") }

		online := set.Bool("live", false, "Life capture. Provided argument must be interface name")
		sl := set.Int("snaplen", 0, "Set non-default snaplen")
		pm := set.Bool("promisc", false, "Set interface to promiscous")
		f := set.String("filter", "", "Filter packets with this filter")

		set.Parse(args)

		filter = *f
		live = *online
		promisc = *pm
		snaplen = *sl

		arguments = set.Args()
		if set.NArg() > 0 {
			if *online {
				files = []string{arguments[0]}
				arguments = arguments[1:]
			} else {
				for len(arguments) > 0 {
					if arguments[0] == "--" {
						arguments = arguments[1:]
						break
					}
					files = append(files, arguments[0])
					arguments = arguments[1:]
				}
			}
		}
	} else {
		panic("FIXME: implement this")
	}

	if len(files) == 0 {
		return nil, nil, errors.New("libpcap needs at least one input file or interface")
	}

	if name == "" {
		name = fmt.Sprint("libpcap|", filter, "|", strings.Join(files, ";"))
	}

	ret = &libpcapSource{
		id:      name,
		files:   files,
		filter:  filter,
		live:    live,
		which:   -1,
		promisc: promisc,
		snaplen: snaplen,
	}
	return
}

func pcapHelp(name string) {
	fmt.Fprintf(os.Stderr, `
The %s source reads packets via libpcap. This can be either from a list of files
or online from an interface. If files are specified, and further commands need
to be provided, then "--" can be used to stop the file list.

Usage command line:
  source %s a.pcap [b.pcapng] [..] [--]

Flags:
  -live string
    Life capture. Provided argument must be interface name
  -snaplen int
    Set non-default snaplen
  -promisc
    Set interface to promiscous
  -filter string
    Filter packets with this filter

Usage json file (not working):
  {
    "type": "%s",
    "options": "file.csv"
  }
`, name, name, name)
}

func init() {
	packet.RegisterSource("libpcap", "Read packets from a libpcap source.", newLibpcapSource, pcapHelp)
}
