package gbs

import (
	"bytes"
	"encoding/binary"
	"strings"
)

const (
	headerLength = 0x70
)

// Header describes the header of a GBS file.
//
//	0x00  0x03  Identifier string ("GBS")
//	0x03  0x01  Version (1)
//	0x04  0x01  Number of songs (1-255)
//	0x05  0x01  First song (usually 1)
//	0x06  0x02  Load address ($400-$7fff)
//	0x08  0x02  Init address ($400-$7fff)
//	0x0a  0x02  Play address ($400-$7fff)
//	0x0c  0x02  Stack pointer
//	0x0e  0x01  Timer modulo  (see TIMING)
//	0x0f  0x01  Timer control (see TIMING)
//	0x10  0x32  Title string
//	0x30  0x32  Author string
//	0x50  0x32  Copyright string
type Header struct {
	IdentifierBytes [3]byte
	Version         byte
	NumberOfSongs   byte
	FirstSong       byte
	LoadAddr        uint16
	InitAddr        uint16
	PlayAddr        uint16
	StackPtr        uint16
	TimerModulo     byte
	TimerControl    byte
	TitleBytes      [32]byte
	AuthorBytes     [32]byte
	CopyrightBytes  [32]byte
}

func NewHeader(data []byte) (Header, error) {
	header := Header{}
	headerBuf := bytes.NewBuffer(data)
	err := binary.Read(headerBuf, binary.LittleEndian, &header)
	if err != nil {
		return header, err
	}

	return header, nil
}

func trimHeaderString(b []byte) string {
	s := string(b)
	s = strings.Trim(s, "\u0000")
	s = strings.TrimSpace(s)
	return s
}

func (gbs Header) Identifier() string {
	return trimHeaderString(gbs.IdentifierBytes[:])
}

func (gbs Header) Title() string {
	return trimHeaderString(gbs.TitleBytes[:])
}

func (gbs Header) Author() string {
	return trimHeaderString(gbs.AuthorBytes[:])
}

func (gbs Header) Copyright() string {
	return trimHeaderString(gbs.CopyrightBytes[:])
}

// headerChecksum generates the cartridge header checksum according to
// https://gbdev.gg8.se/wiki/articles/The_Cartridge_Header#014D_-_Header_Checksum.
// Example Code: x=0:FOR i=0134h TO 014Ch:x=x-MEM[i]-1:NEXT
func headerChecksum(gbHeader []byte) byte {
	var cs int
	for _, b := range gbHeader {
		cs = (cs - int(b) - 1) & 0xff
	}
	return byte(cs)
}

// globalChecksum generates the cartridge header checksum according to
// https://gbdev.gg8.se/wiki/articles/The_Cartridge_Header#014E-014F_-_Global_Checksum.
// Produced by adding all bytes of the cartridge (except for the two checksum bytes).
func globalChecksum(gb []byte) uint16 {
	var cs int

	for i, b := range gb {
		if i == 0x14F || i == 0x150 {
			continue
		}

		cs = (cs + int(b)) & 0xffff
	}

	return uint16(cs)
}
