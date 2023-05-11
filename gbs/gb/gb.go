package gb

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

// GetROMSize returns the cartridge type and ROM size header values according to
// https://gbdev.gg8.se/wiki/articles/The_Cartridge_Header.
func GetROMSize(fileSize int) (romSize int, trueSize int, usesMBC bool) {
	for i := 0; i < 9; i++ {
		if fileSize <= (0x8000 << i) {
			romSize = i
			trueSize = romSizes[i]
			return romSize, trueSize, usesMBC
		}

		// only set to true if a second loop happens
		usesMBC = true
	}

	return -1, -1, false
}
