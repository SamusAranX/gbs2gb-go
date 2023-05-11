package gbs

import (
	"encoding/binary"
	"errors"
	"fmt"
	"gbs2gb/gbs/gb"
	"gbs2gb/utils"
	"log"
)

const (
	playerEntrypoint = 0x150
)

// validateHeader parses the GBS file's header, validates it, and returns it upon success.
func validateHeader(headerBytes []byte) (Header, error) {
	header, err := NewHeader(headerBytes)
	if err != nil {
		return header, err
	}

	if header.Identifier() != "GBS" || header.Version != 1 {
		return header, errors.New("file does not have a valid GBS header")
	}

	if header.LoadAddr < 0x470 {
		return header, errors.New(fmt.Sprintf("incompatible load address 0x%X", header.LoadAddr))
	}

	return header, nil
}

// calculateROMTypeAndSize returns the cartridge type and ROM size according to
// https://gbdev.gg8.se/wiki/articles/The_Cartridge_Header#0147_-_Cartridge_Type.
func calculateROMTypeAndSize(fileLength int) (cartType, romSize, romSizeBytes int, err error) {
	romSize, romSizeBytes, usesMBC := gb.GetROMSize(fileLength)
	if romSize == -1 {
		return cartType,
			romSize,
			romSizeBytes,
			errors.New(fmt.Sprintf("final gb file is too large: %d", fileLength))
	}

	if usesMBC {
		cartType = 1
	}

	return cartType, romSize, romSizeBytes, nil
}

// insertPlayerAndGBS inserts GBSPlay103_Mod.gb and a GBS file into gbBytes.
func insertPlayerAndGBS(header Header, cartType, romSize int, gbBytes, gbsBytes []byte) {
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
	gbsStart := int(header.LoadAddr - headerLength)
	gbsBytesCopied := copy(gbBytes[gbsStart:gbsStart+len(gbsBytes)], gbsBytes)
	log.Printf("gbs file inserted: %d", gbsBytesCopied)
}

func fillRSTTable(gbsStart int, staticData []byte) {
	rstBase := gbsStart + headerLength
	for i := 0; i < 8; i++ {
		rstInst := 0x01 + i*8
		rstAddr := rstBase + i*8

		rst := make([]byte, 2)
		binary.LittleEndian.PutUint16(rst, uint16(rstAddr&0xFFFF))
		_ = copy(staticData[rstInst:rstInst+2], rst)

		//log.Printf("rst 0x%02X: 0x%04X [%d]", rstInst, rstAddr, rstBytesCopied)
	}
}

func fillTextTable(gbsStart int, staticData []byte) {
	textBase := gbsStart + 0x10 // offset of header.TitleBytes
	for i := 0; i < 6; i++ {
		textEntry := 0xC + i*8
		textOffset := textBase + i*16

		text := make([]byte, 2)
		binary.LittleEndian.PutUint16(text, uint16(textOffset&0xFFFF))
		_ = copy(staticData[textEntry:textEntry+2], text)

		//log.Printf("text 0x%02X: 0x%04X [%d]", textEntry, textOffset, textBytesCopied)
	}
}

func doOtherStuff(gbsStart int, otherStuff []byte) {
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
		_ = copy(otherStuff[offset:offset+2], stuff)

		//log.Printf("stuff 0x%02X: 0x%04X [%d]", offset, ptr, stuffBytesCopied)
	}
}

func doShiftedData(gbsStart int, shiftedData []byte) {
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
		_ = copy(shiftedData[offset:offset+2], shift)

		//log.Printf("shift 0x%02X: 0x%04X [%d]", offset, ptr, shiftBytesCopied)
	}
}

func playerCheckSubroutine(gbsStart int, gbBytes []byte) {
	pcs1 := make([]byte, 2)
	pcs2 := make([]byte, 2)
	binary.LittleEndian.PutUint16(pcs1, uint16(gbsStart+headerLength+0x40))
	binary.LittleEndian.PutUint16(pcs2, uint16(gbsStart+headerLength+0x48))
	_ = copy(gbBytes[0x41:0x43], pcs1)
	_ = copy(gbBytes[0x51:0x53], pcs2)
	//log.Printf("pcs copied: %d, %d", pcs1BytesCopied, pcs2BytesCopied)
}

func rewriteInterruptInfo(gbsStart int, gbBytes []byte) {
	tacByte := gbBytes[gbsStart+0x0F] // GBS timer control byte
	if tacByte&0x40 == 0x40 {
		gbBytes[0x40] = 0xC3
		gbBytes[0x50] = 0xC3
		gbBytes[playerEntrypoint+0x38] = 0x05
		gbBytes[playerEntrypoint+0x5A] = 0x05
	}
}

func apply150400Recoding(gbBytes []byte, binaryOffset int) {
	// (FE11 'CC????' 11???? 210082)
	pe := make([]byte, 2)
	binary.LittleEndian.PutUint16(pe, uint16(playerEntrypoint+0x66))
	log.Printf("pe copied: %d", copy(gbBytes[0x9D+binaryOffset:0x9F+binaryOffset], pe))

	// (FE11 CC???? '11????' 210082):
	binary.LittleEndian.PutUint16(pe, uint16(playerEntrypoint+0x10B+1))
	log.Printf("pe copied: %d", copy(gbBytes[0xA0+binaryOffset:0xA2+binaryOffset], pe))

	// 0x101 jump to the 0x150-0x400 code
	binary.LittleEndian.PutUint16(pe, uint16(playerEntrypoint))
	log.Printf("pe copied: %d", copy(gbBytes[0x102:0x104], pe))

	binary.LittleEndian.PutUint16(pe, uint16(playerEntrypoint+0x9E+1))
	log.Printf("pe copied: %d", copy(gbBytes[playerEntrypoint+0x95:playerEntrypoint+0x97], pe))
	binary.LittleEndian.PutUint16(pe, uint16(playerEntrypoint+0x92))
	log.Printf("pe copied: %d", copy(gbBytes[playerEntrypoint+0x100:playerEntrypoint+0x102], pe))
}

func MakeGB(gbsBytes []byte, outFile string, debug bool) error {
	header, err := validateHeader(gbsBytes[:headerLength])
	if err != nil {
		return err
	}

	log.Printf("Title:        %s", header.Title())
	log.Printf("Author:       %s", header.Author())
	log.Printf("Copyright:    %s", header.Copyright())
	log.Printf("Load Address: 0x%X", header.LoadAddr)
	log.Println("--------------------")

	fileLength := len(gbsBytes) + int(header.LoadAddr) - headerLength
	cartType, romSize, romSizeBytes, err := calculateROMTypeAndSize(fileLength)
	log.Printf("CartridgeType/ROMSize: %d/%d", cartType, romSize)
	log.Printf("ROM Size in bytes: %d", romSizeBytes)

	gbBytes := make([]byte, romSizeBytes)
	for i := range gbBytes {
		// empty rom space needs to be filled with 0xFF for checksum purposes
		gbBytes[i] = 0xFF
	}

	insertPlayerAndGBS(header, cartType, romSize, gbBytes, gbsBytes)

	// refer to GBS2GB.txt for the following:

	gbsStart := int(header.LoadAddr - headerLength)
	staticData := gbBytes[:0x3B]

	// RST Table
	fillRSTTable(gbsStart, staticData)

	// Text Data
	fillTextTable(gbsStart, staticData)

	// Other Stuff
	otherStuff := gbBytes[playerEntrypoint+0x07 : playerEntrypoint+0x07+0x7B]
	doOtherStuff(gbsStart, otherStuff)

	// EI, Halt
	_ = copy(gbBytes[playerEntrypoint+0x92:playerEntrypoint+0x94], []byte{0xFB, 0x76})
	//log.Printf("ei, halt copied: %d", eiHaltBytesCopied)

	//log.Printf("gbsStart: 0x%04X", gbsStart)
	//log.Printf("pe: 0x%04X:0x%04X", playerEntrypoint, playerEntrypoint+0x106)

	// shifted data (see gbsplay.asm for the GBS player specifics)
	shiftedData := gbBytes[playerEntrypoint : playerEntrypoint+0x106]
	doShiftedData(gbsStart, shiftedData)

	// fix off-by-one error
	gbBytes[playerEntrypoint+0x65] = 0x64

	// player check subroutine
	playerCheckSubroutine(gbsStart, gbBytes)

	// rewrite reserved interrupt info
	rewriteInterruptInfo(gbsStart, gbBytes)

	// 150 - 400 recoding
	// compensating for 4 extra bytes in the newer gbs player
	apply150400Recoding(gbBytes, 4)

	// recompute global checksum (no game boy checks this but hey why not)
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
