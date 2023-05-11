package main

import (
	"bytes"
	"encoding/binary"
	"gbs2gb/gb"
	"io"
	"log"
	"os"
	"path"
)

const (
	GBSPlay = "GBSPlay103_Mod.gb"

	PlayerEntrypoint = 0x150
)

func main() {
	const input = "Pokemon Red.gbs"
	const output = "Pokemon Red (music).gb"

	inputFile, err := os.Open(input)
	if err != nil {
		panic(err)
	}

	gbsBytes, err := io.ReadAll(inputFile)
	if err != nil {
		panic(err)
	}

	header := gb.GBSHeader{}
	headerBuf := bytes.NewBuffer(gbsBytes[:gb.GBSHeaderLength])
	err = binary.Read(headerBuf, binary.LittleEndian, &header)
	if err != nil {
		panic(err)
	}

	log.Printf("%s:\n", path.Base(input))
	log.Printf("Title:     %s\n", header.Title())
	log.Printf("Author:    %s\n", header.Author())
	log.Printf("Copyright: %s\n", header.Copyright())

	log.Printf("Load Address: 0x%[1]X (%[1]d)", header.LoadAddr)

	if header.LoadAddr < 0x470 {
		log.Panicln("Wrong load address")
	}

	gbsSize := len(gbsBytes)
	fileLength := gbsSize + int(header.LoadAddr) - gb.GBSHeaderLength
	fileMBC := 0

	// https://gbdev.gg8.se/wiki/articles/The_Cartridge_Header#0148_-_ROM_Size
	log.Printf("file length: %d\n", fileLength)
	romSize, usesMBC := gb.GetROMSize(fileLength)
	if usesMBC {
		fileMBC = 1
	}

	log.Printf("mbc: %d - rom size: %d", fileMBC, romSize)

	gbBytes := make([]byte, romSize)
	for i := range gbBytes {
		// empty rom space needs to be filled with 0xFF for checksum purposes
		gbBytes[i] = 0xFF
	}

	log.Printf("rom length: %d\n", len(gbBytes))

	gbsPlayer := gb.GBSPlayROM()

	// patch header
	log.Printf("title bytes copied: %d", copy(gbsPlayer[0x134:0x143], header.TitleBytes[:15]))
	log.Printf("mbc/size bytes copied: %d", copy(gbsPlayer[0x147:0x149], []byte{byte(fileMBC), byte(romSize)}))

	// regenerate header checksum
	headerChecksum := gb.HeaderChecksum(gbsPlayer[0x134:0x14d])
	log.Printf("checksum bytes copied: %d", copy(gbsPlayer[0x14d:0x14e], []byte{headerChecksum}))

	// insert gbs player into gb rom
	gbsPlayerBytesCopied := copy(gbBytes[:0x400], gbsPlayer[:])
	log.Printf("gbs player inserted: %d", gbsPlayerBytesCopied)

	// insert gbs file
	gbsStart := int(header.LoadAddr - gb.GBSHeaderLength)
	gbsBytesCopied := copy(gbBytes[gbsStart:gbsStart+gbsSize], gbsBytes)
	log.Printf("gbs file inserted: %d", gbsBytesCopied)

	staticData := gbBytes[:59]

	// RST vectors
	g := gbsStart + gb.GBSHeaderLength
	for i := 0; i < 8; i++ {
		gi := 1 + i*8
		staticData[gi] = byte(g & 0xff)
		staticData[gi+1] = byte(g >> 8)
		if i < 7 {
			g += 8
		}
	}

	// text data
	g = gbsStart + 16
	for i := 0; i < 6; i++ {
		gi := 12 + i*8
		staticData[gi] = byte(g & 0xff)
		staticData[gi+1] = byte(g >> 8)
		if i < 5 {
			g += 8
		}
	}

	// other stuff (whatever this is)
	otherStuff := gbBytes[PlayerEntrypoint : PlayerEntrypoint+123]
	gOffsets := []int{12, 4, 15, 15, 14, 5, 15}
	stuffOffsets := []int{7, 40, 49, 64, 75, 95, 121}
	for i := 0; i < len(gOffsets); i++ {
		g = gbsStart + gOffsets[i]
		s := stuffOffsets[i]
		otherStuff[s] = byte(g & 0xff)
		otherStuff[s+1] = byte(g >> 8)
	}

	// ei, halt
	eiHaltBytesCopied := copy(gbBytes[PlayerEntrypoint+146:PlayerEntrypoint+148], []byte{0xFB, 0x76})
	log.Printf("ei, halt copied: %d", eiHaltBytesCopied)

	// shifted data (see gbsplay.asm for the GBS player specifics)
	shiftedData := gbBytes[PlayerEntrypoint : PlayerEntrypoint+262]
	gOffsets = []int{10, 4, 4, 8}
	dataOffsets := []int{153, 184, 199, 260}
	for i := 0; i < len(gOffsets); i++ {
		g := gbsStart + gOffsets[i]
		d := dataOffsets[i]
		shiftedData[d] = byte(g & 0xff)
		shiftedData[d+1] = byte(g >> 8)
	}

	// fix off-by-one error
	offByOneCopied := copy(gbBytes[PlayerEntrypoint+101:PlayerEntrypoint+102], []byte{0x64})
	log.Printf("off by one copied: %d", offByOneCopied)

	// player check subroutine
	g = gbsStart + gb.GBSHeaderLength
	pcs1 := make([]byte, 2)
	pcs2 := make([]byte, 2)
	binary.LittleEndian.PutUint16(pcs1, uint16(g+64))
	binary.LittleEndian.PutUint16(pcs2, uint16(g+72))
	pcs1BytesCopied := copy(gbBytes[65:67], pcs1)
	pcs2BytesCopied := copy(gbBytes[81:83], pcs2)
	log.Printf("pcs copied: %d, %d", pcs1BytesCopied, pcs2BytesCopied)

	// rewrite reserved interrupt info
	tacByte := gbBytes[gbsStart+0x0F]
	if tacByte&0x40 == 0x40 {
		gbBytes[64] = 0xC3
		gbBytes[80] = 0xC3
		gbBytes[PlayerEntrypoint+56] = 0x05
		gbBytes[PlayerEntrypoint+90] = 0x05
	}

	// compensating for 4 extra bytes in the newer gbs player
	newBin := 4

	// player entrypoint stuff
	// (FE11 'CC????' 11???? 210082)
	pe := make([]byte, 2)
	binary.LittleEndian.PutUint16(pe, uint16(PlayerEntrypoint+102))
	log.Printf("pe copied: %d", copy(gbBytes[157+newBin:159+newBin], pe))

	// (FE11 CC???? '11????' 210082):
	binary.LittleEndian.PutUint16(pe, uint16(PlayerEntrypoint+267+1))
	log.Printf("pe copied: %d", copy(gbBytes[160+newBin:162+newBin], pe))

	// 0x101 jump to the 0x150-0x400 code
	binary.LittleEndian.PutUint16(pe, uint16(PlayerEntrypoint))
	log.Printf("pe copied: %d", copy(gbBytes[258:260], pe))

	binary.LittleEndian.PutUint16(pe, uint16(PlayerEntrypoint+158+1))
	log.Printf("pe copied: %d", copy(gbBytes[PlayerEntrypoint+149:PlayerEntrypoint+151], pe))
	binary.LittleEndian.PutUint16(pe, uint16(PlayerEntrypoint+146))
	log.Printf("pe copied: %d", copy(gbBytes[PlayerEntrypoint+256:PlayerEntrypoint+258], pe))

	globalChecksum := make([]byte, 2)
	binary.BigEndian.PutUint16(globalChecksum, gb.GlobalChecksum(gbBytes))
	log.Printf("global checksum copied: %d", copy(gbBytes[0x14E:0x150], globalChecksum))

	outputFile, err := os.Create(output)
	if err != nil {
		panic(err)
	}

	defer outputFile.Close()

	written, err := outputFile.Write(gbBytes)
	if err != nil {
		panic(err)
	}

	log.Printf("gbBytes written to file: %d/%d", written, romSize)
}
