package gb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"gbs2gb/utils"
	"log"
)

const (
	playerEntrypoint = 0x150
)

func MakeGB(gbsBytes []byte, outFile string) error {
	header := GBSHeader{}
	headerBuf := bytes.NewBuffer(gbsBytes[:gbsHeaderLength])
	err := binary.Read(headerBuf, binary.LittleEndian, &header)
	if err != nil {
		panic(err)
	}

	if header.Identifier() != "GBS" || header.Version != 1 {
		return errors.New("file does not have a valid GBS header")
	}

	log.Printf("Title:        %s", header.Title())
	log.Printf("Author:       %s", header.Author())
	log.Printf("Copyright:    %s", header.Copyright())
	log.Printf("Load Address: 0x%[1]X (%[1]d)", header.LoadAddr)

	if header.LoadAddr < 0x470 {
		return errors.New(fmt.Sprintf("incompatible load address 0x%X", header.LoadAddr))
	}

	gbsSize := len(gbsBytes)
	fileLength := gbsSize + int(header.LoadAddr) - gbsHeaderLength
	cartType := 0

	// https://gbdev.gg8.se/wiki/articles/The_Cartridge_Header#0148_-_ROM_Size
	log.Printf("GB file size: %d", fileLength)
	romSize, romSizeBytes, usesMBC := getROMSize(fileLength)
	if romSize == -1 {
		return errors.New(fmt.Sprintf("final gb file is too large: %d", fileLength))
	}

	if usesMBC {
		cartType = 1
	}

	log.Printf("CartridgeType: %d", cartType)
	log.Printf("ROMSize: %d", romSize)

	gbBytes := make([]byte, romSizeBytes)
	for i := range gbBytes {
		// empty rom space needs to be filled with 0xFF for checksum purposes
		gbBytes[i] = 0xFF
	}

	log.Printf("rom length: %d", len(gbBytes))

	gbsPlayer := GBSPlayROM()

	// patch header
	log.Printf("title bytes copied: %d", copy(gbsPlayer[0x134:0x143], header.TitleBytes[:15]))
	log.Printf("mbc/size bytes copied: %d", copy(gbsPlayer[0x147:0x149], []byte{byte(cartType), byte(romSize)}))

	// regenerate header checksum
	headerChecksum := headerChecksum(gbsPlayer[0x134:0x14d])
	log.Printf("checksum bytes copied: %d", copy(gbsPlayer[0x14d:0x14e], []byte{headerChecksum}))

	// insert gbs player into gb rom
	gbsPlayerBytesCopied := copy(gbBytes[:0x400], gbsPlayer[:])
	log.Printf("gbs player inserted: %d", gbsPlayerBytesCopied)

	// insert gbs file
	gbsStart := int(header.LoadAddr - gbsHeaderLength)
	gbsBytesCopied := copy(gbBytes[gbsStart:gbsStart+gbsSize], gbsBytes)
	log.Printf("gbs file inserted: %d", gbsBytesCopied)

	// refer to GBS2GB.txt for the following:

	staticData := gbBytes[:0x3B]

	// RST Table
	rstBase := gbsStart + gbsHeaderLength
	for i := 0; i < 8; i++ {
		rstInst := 0x01 + i*8
		rstAddr := rstBase + i*8

		rst := make([]byte, 2)
		binary.LittleEndian.PutUint16(rst, uint16(rstAddr&0xFFFF))
		rstBytesCopied := copy(staticData[rstInst:rstInst+2], rst)

		log.Printf("rst 0x%02X: 0x%04X [%d]", rstInst, rstAddr, rstBytesCopied)
	}

	// Text Data
	textBase := gbsStart + 0x10 // offset of header.TitleBytes
	for i := 0; i < 6; i++ {
		textEntry := 0xC + i*8
		textOffset := textBase + i*16

		text := make([]byte, 2)
		binary.LittleEndian.PutUint16(text, uint16(textOffset&0xFFFF))
		textBytesCopied := copy(staticData[textEntry:textEntry+2], text)

		log.Printf("text 0x%02X: 0x%04X [%d]", textEntry, textOffset, textBytesCopied)
	}

	// Other Stuff
	otherStuff := gbBytes[playerEntrypoint+0x07 : playerEntrypoint+0x07+0x7B]
	stuffOffsets := []int{0x00, 0x21, 0x2A, 0x39, 0x44, 0x58, 0x72}
	stuffPointers := []int{
		0x0C, // header.StackPtr
		0x04, // header.NumberOfSongs
		0x0F, // header.TimerControl
		0x0F, // header.TimerControl
		0x0E, // header.TimerModulo
		0x05, // header.FirstSong
		0x0F, // header.TimerControl
	}
	for i := 0; i < len(stuffOffsets); i++ {
		offset := stuffOffsets[i]
		ptr := gbsStart + stuffPointers[i]

		stuff := make([]byte, 2)
		binary.LittleEndian.PutUint16(stuff, uint16(ptr&0xFFFF))
		stuffBytesCopied := copy(otherStuff[offset:offset+2], stuff)

		log.Printf("stuff 0x%02X: 0x%04X [%d]", offset, ptr, stuffBytesCopied)
	}

	// EI, Halt
	eiHaltBytesCopied := copy(gbBytes[playerEntrypoint+0x92:playerEntrypoint+0x94], []byte{0xFB, 0x76})
	log.Printf("ei, halt copied: %d", eiHaltBytesCopied)

	var g int

	log.Printf("gbsStart: 0x%04X", gbsStart)
	log.Printf("pe: 0x%04X:0x%04X", playerEntrypoint, playerEntrypoint+0x106)

	// shifted data (see gbsplay.asm for the GBS player specifics)
	shiftedData := gbBytes[playerEntrypoint : playerEntrypoint+0x106]
	shiftOffsets := []int{0x99, 0xB8, 0xC7, 0x104}
	shiftPointers := []int{
		0x0A, // header.PlayAddr
		0x04, // header.NumberOfSongs
		0x04, // header.NumberOfSongs
		0x08, // header.InitAddr
	}
	for i := 0; i < len(shiftOffsets); i++ {
		offset := shiftOffsets[i]
		ptr := gbsStart + shiftPointers[i]

		shift := make([]byte, 2)
		binary.LittleEndian.PutUint16(shift, uint16(ptr&0xFFFF))
		shiftBytesCopied := copy(shiftedData[offset:offset+2], shift)

		log.Printf("shift 0x%02X: 0x%04X [%d]", offset, ptr, shiftBytesCopied)
	}

	// fix off-by-one error
	gbBytes[playerEntrypoint+0x65] = 0x64

	// player check subroutine
	g = gbsStart + gbsHeaderLength
	pcs1 := make([]byte, 2)
	pcs2 := make([]byte, 2)
	binary.LittleEndian.PutUint16(pcs1, uint16(g+0x40))
	binary.LittleEndian.PutUint16(pcs2, uint16(g+0x48))
	pcs1BytesCopied := copy(gbBytes[0x41:0x43], pcs1)
	pcs2BytesCopied := copy(gbBytes[0x51:0x53], pcs2)
	log.Printf("pcs copied: %d, %d", pcs1BytesCopied, pcs2BytesCopied)

	// rewrite reserved interrupt info
	tacByte := gbBytes[gbsStart+0x0F]
	if tacByte&0x40 == 0x40 {
		gbBytes[0x40] = 0xC3
		gbBytes[0x50] = 0xC3
		gbBytes[playerEntrypoint+0x38] = 0x05
		gbBytes[playerEntrypoint+0x5A] = 0x05
	}

	// compensating for 4 extra bytes in the newer gbs player
	newBin := 4

	// 150 - 400 recoding

	// (FE11 'CC????' 11???? 210082)
	pe := make([]byte, 2)
	binary.LittleEndian.PutUint16(pe, uint16(playerEntrypoint+0x66))
	log.Printf("pe copied: %d", copy(gbBytes[0x9D+newBin:0x9F+newBin], pe))

	// (FE11 CC???? '11????' 210082):
	binary.LittleEndian.PutUint16(pe, uint16(playerEntrypoint+0x10B+1))
	log.Printf("pe copied: %d", copy(gbBytes[0xA0+newBin:0xA2+newBin], pe))

	// 0x101 jump to the 0x150-0x400 code
	binary.LittleEndian.PutUint16(pe, uint16(playerEntrypoint))
	log.Printf("pe copied: %d", copy(gbBytes[0x102:0x104], pe))

	binary.LittleEndian.PutUint16(pe, uint16(playerEntrypoint+0x9E+1))
	log.Printf("pe copied: %d", copy(gbBytes[playerEntrypoint+0x95:playerEntrypoint+0x97], pe))
	binary.LittleEndian.PutUint16(pe, uint16(playerEntrypoint+0x92))
	log.Printf("pe copied: %d", copy(gbBytes[playerEntrypoint+0x100:playerEntrypoint+0x102], pe))

	gcs := make([]byte, 2)
	binary.BigEndian.PutUint16(gcs, globalChecksum(gbBytes))
	log.Printf("global checksum copied: %d", copy(gbBytes[0x14E:0x150], gcs))

	written, err := utils.WriteAllBytes(outFile, gbBytes)
	if err != nil {
		return err
	}

	log.Printf("gbBytes written to file: %d/%d", written, romSizeBytes)

	return nil
}
