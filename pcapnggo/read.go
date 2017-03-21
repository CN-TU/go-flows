// Copyright 2014 Damjan Cvetko. All rights reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

package pcapnggo

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"bufio"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type iface struct {
	tsoffset        int64
	tzone           int32
	nanoSecsFactorM uint64
	nanoSecsFactorD uint64
	mul             bool
	linkType        layers.LinkType
	snaplen         uint32
}

// Reader wraps an underlying io.Reader to read packet data in PCAPNG
type Reader struct {
	r         *bufio.Reader
	byteOrder binary.ByteOrder
	linkType  layers.LinkType
	ifaces    []iface
	// reusable buffer
	buf [24]byte
}

const (
	byteOrderMagic = 0x1A2B3C4D

	versionMajor = 1
	versionMinor = 0

	blockTypeInterfaceDescriptor = 0x00000001 // Interface Description Block
	blockTypePacketBlock         = 0x00000002 // Packet Block (deprecated)
	blockTypeSimplePacketBlock   = 0x00000003 // Simple Packet Block
	blockTypeEnhancedPacketBlock = 0x00000006 // Enhanced Packet Block
	blockTypeSectionHeaderBlock  = 0x0A0D0D0A // Section Header Block
)

type optionCode uint16

const (
	optionEndOfOpt        optionCode = iota
	optionComment                    // comment
	optionHardware                   // description of the hardware
	optionOS                         // name of the operating system
	optionUserApplication            // name of the application
)

const (
	_                                     = iota
	_                                     = iota
	optionIFName               optionCode = iota // interface name
	optionIFDescription                          // interface description
	optionIFIPV4Address                          // IPv4 network address and netmask for the interface
	optionIFIPV6Address                          // IPv6 network address and prefix length for the interface
	optionIFMACAddress                           // interface hardware MAC address
	optionIFEUIAddress                           // interface hardware EUI address
	optionIFSpeed                                // interface speed in bits/s
	optionIFTimetampResolution                   // timestamp resolution
	optionIFTimezone                             // time zone
	optionIFFilter                               // capture filter
	optionIFOS                                   // operating system
	optionIFFCSLength                            // length of the Frame Check Sequence in bits
	optionIFTimestampOffset                      // offset (in seconds) that must be added to packet timestamp
)

type option struct {
	code  optionCode
	value []byte
}

// NewReader returns a new reader object, for reading packet data from
// the given reader. The reader must be open and header data is
// read from it at this point.
// If the file format is not supported an error is returned
//
//  // Create new reader:
//  f, _ := os.Open("/tmp/file.pcapng")
//  defer f.Close()
//  r, err := NewReader(f)
//  data, ci, err := r.ReadPacketData()
func NewReader(r io.Reader) (*Reader, error) {
	ret := &Reader{r: bufio.NewReader(r), byteOrder: binary.BigEndian}

	//pcapng _must_ start with a section header
	if t, err := ret.readBlockType(); err != nil {
		return nil, err
	} else if t != blockTypeSectionHeaderBlock {
		return nil, fmt.Errorf("Unknown magic %x", t)
	}

	if err := ret.readStartOfSection(); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *Reader) readStartOfSection() (err error) {
	if err = r.readSectionHeader(); err != nil {
		return
	}
	r.ifaces = make([]iface, 0, 1)
	//it must contain at least one interface description before the first packet
	for {
		var t uint32
		if t, err = r.readBlockType(); err != nil {
			return
		}
		switch t {
		case blockTypeInterfaceDescriptor:
			err = r.readInterfacesDescriptor()
			return
		case blockTypePacketBlock:
			return errors.New("Section without interface description")
		case blockTypeEnhancedPacketBlock:
			return errors.New("Section without interface description")
		case blockTypeSimplePacketBlock:
			return errors.New("Section without interface description")
		default:
			if err = r.skipBlock(); err != nil {
				return
			}
		}
	}
}

func (r *Reader) readBlockType() (uint32, error) {
	if _, err := io.ReadFull(r.r, r.buf[0:4]); err != nil {
		return 0, err
	}
	return r.byteOrder.Uint32(r.buf[0:4]), nil
}

func (r *Reader) skipBlock() error {
	if _, err := io.ReadFull(r.r, r.buf[0:4]); err != nil {
		return err
	}
	skip := r.byteOrder.Uint32(r.buf[0:4])
	if _, err := r.r.Discard(int(skip - 8)); err != nil {
		return err
	}
	return nil
}

func (r *Reader) readSectionHeader() error {
	// read everything except block header and set stuff
	if _, err := io.ReadFull(r.r, r.buf[:12]); err != nil {
		return err
	}
	if binary.BigEndian.Uint32(r.buf[4:8]) == byteOrderMagic {
		r.byteOrder = binary.BigEndian
	} else if binary.LittleEndian.Uint32(r.buf[4:8]) == byteOrderMagic {
		r.byteOrder = binary.LittleEndian
	} else {
		return errors.New("Wrong byte order value in Section Header")
	}

	totalLen := r.byteOrder.Uint32(r.buf[0:4])
	vMajor := r.byteOrder.Uint16(r.buf[8:10])
	vMinor := r.byteOrder.Uint16(r.buf[10:12])
	if vMajor != versionMajor || vMinor != versionMinor {
		return errors.New("Unknown pcapng Version in Section Header")
	}
	// skip rest of header (we don't need section length)
	if _, err := r.r.Discard(int(totalLen - 16)); err != nil {
		return err
	}

	return nil
}

func (r *Reader) readOption() (*option, uint16, error) {
	if _, err := io.ReadFull(r.r, r.buf[:4]); err != nil {
		return nil, 0, err
	}
	ret := &option{
		code: optionCode(r.byteOrder.Uint16(r.buf[0:2]))}
	length := r.byteOrder.Uint16(r.buf[2:4])
	if length != 0 {
		ret.value = make([]byte, length)
		if _, err := io.ReadFull(r.r, ret.value); err != nil {
			return nil, 0, err
		}
		//consume padding
		padding := 4 - length%4
		if padding > 0 {
			if _, err := r.r.Discard(int(padding)); err != nil {
				return nil, 0, err
			}
		}
		return ret, padding + length + 4, nil
	}
	return ret, 4, nil
}

func (r *Reader) readInterfacesDescriptor() error {
	if _, err := io.ReadFull(r.r, r.buf[:12]); err != nil {
		return err
	}
	totalLen := r.byteOrder.Uint32(r.buf[0:4])
	i := iface{}
	i.linkType = layers.LinkType(r.byteOrder.Uint16(r.buf[4:6]))
	i.snaplen = r.byteOrder.Uint32(r.buf[8:12])
	//parse options
	var res uint64 = 6
	for optionLength := totalLen - 20; optionLength > 0; {
		opt, consumed, err := r.readOption()
		if err != nil {
			return err
		}
		optionLength -= uint32(consumed)
		if opt.code == optionEndOfOpt && optionLength != 0 {
			return errors.New("End of option before last option")
		}
		switch opt.code {
		case optionIFTimestampOffset:
			i.tsoffset = int64(r.byteOrder.Uint64(opt.value)) * 1e9
		case optionIFTimetampResolution:
			res = uint64(opt.value[0])
		case optionIFTimezone:
			i.tzone = int32(r.byteOrder.Uint32(opt.value))
		default:
		}
	}
	if _, err := r.r.Discard(4); err != nil {
		return err
	}
	if (res & 0x80) != 0 {
		//negative power of 2
		res = 1 << (res & 0x7F)
	} else {
		//negative power of 10
		tmp := res
		res = 1
		for j := uint64(0); j < tmp; j++ {
			res *= 10
		}
	}
	switch {
	case res == 0:
		return errors.New("Wrong scaling")
	case res <= 1e9:
		i.nanoSecsFactorM = 1e9 / res
		i.mul = true
	default:
		i.nanoSecsFactorD = res / 1e9
		i.mul = false
	}
	r.ifaces = append(r.ifaces, i)
	return nil
}

/*
// Read next packet from file
func (r *Reader) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	if ci, err = r.readPacketHeader(); err != nil {
		return
	}
	if ci.CaptureLength > int(r.snaplen) {
		err = fmt.Errorf("capture length exceeds snap length: %d > %d", 16+ci.CaptureLength, r.snaplen)
		return
	}
	data = make([]byte, ci.CaptureLength)
	_, err = io.ReadFull(r.r, data)
	return data, ci, err
}*/

/*
func (r *Reader) readPacketHeader() (ci gopacket.CaptureInfo, err error) {
	if _, err = io.ReadFull(r.r, r.buf[:]); err != nil {
		return
	}
	ci.Timestamp = time.Unix(int64(r.byteOrder.Uint32(r.buf[0:4])), int64(r.byteOrder.Uint32(r.buf[4:8])*r.nanoSecsFactor)).UTC()
	ci.CaptureLength = int(r.byteOrder.Uint32(r.buf[8:12]))
	ci.Length = int(r.byteOrder.Uint32(r.buf[12:16]))
	ci.InterfaceIndex
	return
}*/

func (r *Reader) ReadPacketDataDirect(data *[]byte) (ci gopacket.CaptureInfo, layer layers.LinkType, err error) {
	var ifaceID, capLen, pLen uint32
	var ts uint64
	for {
		var t uint32
		t, err = r.readBlockType()
		if err != nil {
			return
		}
		switch t {
		case blockTypeInterfaceDescriptor:
			r.readInterfacesDescriptor()
		case blockTypeSectionHeaderBlock:
			r.readStartOfSection()
		case blockTypePacketBlock: //obsolete
			if _, err = io.ReadFull(r.r, r.buf[:24]); err != nil {
				return
			}
			blockLen := r.byteOrder.Uint32(r.buf[0:4])
			ifaceID = uint32(r.byteOrder.Uint16(r.buf[4:6]))
			ts = uint64(r.byteOrder.Uint32(r.buf[8:12]))<<32 | uint64(r.byteOrder.Uint32(r.buf[12:16]))
			capLen = r.byteOrder.Uint32(r.buf[16:20])
			pLen = r.byteOrder.Uint32(r.buf[20:24])
			capacity := cap(*data)
			if int(capLen) > capacity {
				capLen = uint32(capacity)
			}
			*data = (*data)[:capLen]
			if _, err = io.ReadFull(r.r, *data); err != nil {
				return
			}
			if _, err = r.r.Discard(int(blockLen - 28 - capLen)); err != nil {
				return
			}
			goto FoundPacket
		case blockTypeSimplePacketBlock:
			if _, err = io.ReadFull(r.r, r.buf[:8]); err != nil {
				return
			}
			blockLen := r.byteOrder.Uint32(r.buf[0:4])
			pLen = r.byteOrder.Uint32(r.buf[4:8])
			capLen = pLen
			if capLen > r.ifaces[0].snaplen {
				capLen = r.ifaces[0].snaplen
			}
			capacity := cap(*data)
			if int(capLen) > capacity {
				capLen = uint32(capacity)
			}
			*data = (*data)[:capLen]
			if _, err = io.ReadFull(r.r, *data); err != nil {
				return
			}
			if _, err = r.r.Discard(int(blockLen - 12 - capLen)); err != nil {
				return
			}
			goto FoundPacket
		case blockTypeEnhancedPacketBlock:
			if _, err = io.ReadFull(r.r, r.buf[:24]); err != nil {
				return
			}
			blockLen := r.byteOrder.Uint32(r.buf[0:4])
			ifaceID = r.byteOrder.Uint32(r.buf[4:8])
			ts = uint64(r.byteOrder.Uint32(r.buf[8:12]))<<32 | uint64(r.byteOrder.Uint32(r.buf[12:16]))
			capLen = r.byteOrder.Uint32(r.buf[16:20])
			pLen = r.byteOrder.Uint32(r.buf[20:24])
			capacity := cap(*data)
			if int(capLen) > capacity {
				capLen = uint32(capacity)
			}
			*data = (*data)[:capLen]
			if _, err = io.ReadFull(r.r, *data); err != nil {
				return
			}
			if _, err = r.r.Discard(int(blockLen - 28 - capLen)); err != nil {
				return
			}
			goto FoundPacket
		default:
			r.skipBlock()
		}
	}
FoundPacket:
	iface := r.ifaces[ifaceID]
	if iface.mul {
		ts *= iface.nanoSecsFactorM
	} else {
		ts /= iface.nanoSecsFactorD
	}
	ci.Timestamp = time.Unix(0, int64(ts)+iface.tsoffset).UTC()
	ci.CaptureLength = int(capLen)
	ci.InterfaceIndex = int(ifaceID)
	ci.Length = int(pLen)
	layer = iface.linkType
	return
}

// LinkType returns network, as a layers.LinkType.
func (r *Reader) LinkType() layers.LinkType {
	return r.ifaces[0].linkType //there is always at least one interface
}

// Reader formater
func (r *Reader) String() string {
	return "PcapFile"
}
