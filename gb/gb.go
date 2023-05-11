package gb

import "strings"

const (
	gbHeaderStart   = 0x100
	GBSHeaderLength = 0x70
)

var (
	romSizes = map[int]int{
		0: 1 << 15, // 32 KiB (no ROM banking)
		1: 1 << 16, // 64 KiB (4 banks)
		2: 1 << 17, // 128 KiB (8 banks)
		3: 1 << 18, // 256 KiB (16 banks)
		4: 1 << 19, // 512 KiB (32 banks)
		5: 1 << 20, // 1 MiB (64 banks)
		6: 1 << 21, // 2 MiB (128 banks)
		7: 1 << 22, // 4 MiB (256 banks)
		8: 1 << 23, // 8 MiB (512 banks)
	}
)

func GetROMSize(fileSize int) (romSize int, usesMBC bool) {
	for i := 0; i < 9; i++ {
		if fileSize <= (0x8000 << i) {
			romSize = romSizes[i]
			return romSize, usesMBC
		}

		// only set to 1 if a second loop happens
		usesMBC = true
	}

	return -1, false
}

// GBSHeader describes the header of a GBS file.
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
type GBSHeader struct {
	Identifier     [3]byte
	Version        byte
	NumberOfSongs  byte
	FirstSong      byte
	LoadAddr       uint16
	InitAddr       uint16
	PlayAddr       uint16
	StackPtr       uint16
	TimerModulo    byte
	TimerControl   byte
	TitleBytes     [32]byte
	AuthorBytes    [32]byte
	CopyrightBytes [32]byte
}

func trimHeaderString(b []byte) string {
	s := string(b)
	s = strings.Trim(s, "\u0000")
	s = strings.TrimSpace(s)
	return s
}

func (gbs GBSHeader) Title() string {
	return trimHeaderString(gbs.TitleBytes[:])
}

func (gbs GBSHeader) Author() string {
	return trimHeaderString(gbs.AuthorBytes[:])
}

func (gbs GBSHeader) Copyright() string {
	return trimHeaderString(gbs.CopyrightBytes[:])
}

// HeaderChecksum generates the cartridge header checksum according to
// https://gbdev.gg8.se/wiki/articles/The_Cartridge_Header#014D_-_Header_Checksum.
// Example Code: x=0:FOR i=0134h TO 014Ch:x=x-MEM[i]-1:NEXT
func HeaderChecksum(gbHeader []byte) byte {
	var cs int
	for _, b := range gbHeader {
		cs = (cs - int(b) - 1) & 0xff
	}
	return byte(cs)
}

// GlobalChecksum generates the cartridge header checksum according to
// https://gbdev.gg8.se/wiki/articles/The_Cartridge_Header#014E-014F_-_Global_Checksum.
// Produced by adding all bytes of the cartridge (except for the two checksum bytes).
func GlobalChecksum(gb []byte) uint16 {
	var cs int

	for i, b := range gb {
		if i == 0x14F || i == 0x150 {
			continue
		}

		cs = (cs + int(b)) & 0xffff
	}

	return uint16(cs)
}
