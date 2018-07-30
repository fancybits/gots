/*
MIT License

Copyright 2016 Comcast Cable Communications Management, LLC

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package ebp

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"
)

type cableLabsEbp struct {
	DataFieldTag     uint8
	DataFieldLength  uint8
	FormatIdentifier uint32
	DataFlags        uint8
	ExtensionFlags   uint8
	SapType          uint8
	Grouping         []uint8
	TimeSeconds      uint32
	TimeFraction     uint32
	PartitionFlags   uint8
	ReservedBytes    []byte
	SuccessReadTime  time.Time
}

func CreateCableLabsEbp() cableLabsEbp {
	return cableLabsEbp{
		DataFieldTag:     CableLabsEbpTag,
		DataFieldLength:  1, // not empty by default
		FormatIdentifier: CableLabsFormatIdentifier,
	}
}

func (ebp *cableLabsEbp) EBPType() byte {
	return ebp.DataFieldTag
}

func (ebp *cableLabsEbp) FragmentFlag() bool {
	return ebp.DataFlags&0x80 != 0
}

func (ebp *cableLabsEbp) SetFragmentFlag(value bool) {
	if ebp.DataFieldLength != 0 && value {
		ebp.DataFlags |= 0x80
	}
}

func (ebp *cableLabsEbp) SegmentFlag() bool {
	return ebp.DataFlags&0x40 != 0
}

func (ebp *cableLabsEbp) SetSegmentFlag(value bool) {
	if ebp.DataFieldLength != 0 && value {
		ebp.DataFlags |= 0x40
	}
}

func (ebp *cableLabsEbp) SapFlag() bool {
	return ebp.DataFlags&0x20 != 0
}

func (ebp *cableLabsEbp) SetSapFlag(value bool) {
	if ebp.DataFieldLength != 0 && value {
		ebp.DataFlags |= 0x20
	}
}

func (ebp *cableLabsEbp) GroupingFlag() bool {
	return ebp.DataFlags&0x10 != 0
}

func (ebp *cableLabsEbp) SetGroupingFlag(value bool) {
	if ebp.DataFieldLength != 0 && value {
		ebp.DataFlags |= 0x10
	}
}

func (ebp *cableLabsEbp) GroupingIDs() []uint8 {
	return ebp.Grouping
}

func (ebp *cableLabsEbp) SetGroupingIDs(values []uint8) {
	ebp.Grouping = make([]uint8, len(values))
	for i := range ebp.Grouping {
		ebp.Grouping[i] = values[i]
		ebp.Grouping[i] |= 0x80
		if i == len(ebp.Grouping)-1 {
			ebp.Grouping[i] &= 0x7F // last index does not have this flag set
		}
	}
}

func (ebp *cableLabsEbp) TimeFlag() bool {
	return ebp.DataFlags&0x08 != 0
}

func (ebp *cableLabsEbp) SetTimeFlag(value bool) {
	if ebp.DataFieldLength != 0 && value {
		ebp.DataFlags |= 0x08
	}
}

func (ebp *cableLabsEbp) ConcealmentFlag() bool {
	return ebp.DataFlags&0x04 != 0
}

func (ebp *cableLabsEbp) SetConcealmentFlag(value bool) {
	if ebp.DataFieldLength != 0 && value {
		ebp.DataFlags |= 0x04
	}
}

func (ebp *cableLabsEbp) ExtensionFlag() bool {
	return ebp.DataFlags&0x01 != 0
}

func (ebp *cableLabsEbp) SetExtensionFlag(value bool) {
	if ebp.DataFieldLength != 0 && value {
		ebp.DataFlags |= 0x01
	}
}

func (ebp *cableLabsEbp) EBPTime() time.Time {
	return extractUtcTime(ebp.TimeSeconds, ebp.TimeFraction)
}

func (ebp *cableLabsEbp) SetEBPTime(t time.Time) {
	insertUtcTime(t, &ebp.TimeSeconds, &ebp.TimeFraction)
}

func (ebp *cableLabsEbp) Sap() byte {
	return ebp.SapType
}

func (ebp *cableLabsEbp) SetSap(sapType byte) {
	ebp.SapType = sapType
}

func (ebp *cableLabsEbp) PartitionFlag() bool {
	return ebp.ExtensionFlag() && ebp.ExtensionFlags&0x80 != 0
}

// Defines when the EBP was read successfully
func (ebp *cableLabsEbp) EBPSuccessReadTime() time.Time {
	return ebp.SuccessReadTime
}

func readCableLabsEbp(data io.Reader) (ebp *cableLabsEbp, err error) {
	ebp = &cableLabsEbp{DataFieldTag: CableLabsEbpTag}

	if err = binary.Read(data, ebpEncoding, &ebp.DataFieldLength); err != nil {
		return nil, err
	}

	remaining := ebp.DataFieldLength

	if err = binary.Read(data, ebpEncoding, &ebp.FormatIdentifier); err != nil {
		return nil, err
	}

	if err = binary.Read(data, ebpEncoding, &ebp.DataFlags); err != nil {
		return nil, err
	}

	remaining -= uint8(5)

	if ebp.ExtensionFlag() {
		if err = binary.Read(data, ebpEncoding, &ebp.ExtensionFlags); err != nil {
			return nil, err
		}
		remaining -= uint8(1)
	}

	if ebp.SapFlag() {
		if err = binary.Read(data, ebpEncoding, &ebp.SapType); err != nil {
			return nil, err
		}
		remaining -= uint8(1)
	}

	if ebp.GroupingFlag() {
		var group byte
		if err = binary.Read(data, ebpEncoding, &group); err != nil {
			return nil, err
		}
		ebp.Grouping = append(ebp.Grouping, group)

		remaining -= uint8(1)
		for group&0x80 != 0 {
			if err = binary.Read(data, ebpEncoding, &group); err != nil {
				return nil, err
			}
			ebp.Grouping = append(ebp.Grouping, group)
			remaining -= uint8(1)
		}
	}

	if ebp.TimeFlag() {
		if err = binary.Read(data, ebpEncoding, &ebp.TimeSeconds); err != nil {
			return nil, err
		}
		if err = binary.Read(data, ebpEncoding, &ebp.TimeFraction); err != nil {
			return nil, err
		}
		remaining -= uint8(8)
	}

	if ebp.PartitionFlag() {
		if err = binary.Read(data, ebpEncoding, &ebp.PartitionFlags); err != nil {
			return nil, err
		}
		remaining -= uint8(1)
	}

	if remaining > 0 {
		ebp.ReservedBytes = make([]byte, remaining)
		if err = binary.Read(data, ebpEncoding, &ebp.ReservedBytes); err != nil {
			return nil, err
		}

	}

	// update the successful read time
	ebp.SuccessReadTime = time.Now()

	return ebp, nil
}

// Data will return the raw bytes of the EBP
func (ebp *cableLabsEbp) Data() []byte {
	requiredFields := new(bytes.Buffer)
	data := new(bytes.Buffer)
	binary.Write(requiredFields, ebpEncoding, ebp.DataFieldTag)

	if ebp.DataFieldLength == 0 {
		return data.Bytes()
	}

	binary.Write(data, ebpEncoding, ebp.FormatIdentifier)

	binary.Write(data, ebpEncoding, ebp.DataFlags)
	if ebp.ExtensionFlag() {
		binary.Write(data, ebpEncoding, ebp.ExtensionFlags)
	}

	if ebp.SapFlag() {
		binary.Write(data, ebpEncoding, ebp.SapType)
	}

	if ebp.GroupingFlag() {
		for i := range ebp.Grouping {
			binary.Write(data, ebpEncoding, ebp.Grouping[i])
		}
	}

	if ebp.TimeFlag() {
		binary.Write(data, ebpEncoding, ebp.TimeSeconds)
		binary.Write(data, ebpEncoding, ebp.TimeFraction)
	}

	if ebp.PartitionFlag() {
		binary.Write(data, ebpEncoding, ebp.PartitionFlags)
	}

	binary.Write(data, ebpEncoding, ebp.ReservedBytes)

	ebp.DataFieldLength = uint8(data.Len())

	binary.Write(requiredFields, ebpEncoding, ebp.DataFieldLength)
	binary.Write(requiredFields, ebpEncoding, data.Bytes())

	return requiredFields.Bytes()
}
