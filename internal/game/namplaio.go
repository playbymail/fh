package game

// Port of namplaio.c. Reads and writes the nampla (named planet) records
// inside the binary "sp##.dat" files (driven by speciesio.go).
//
// Skipped (JSON/S-expression exporters): namplaDataAsJson, namplaDataAsSExpr.

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// binary_nampla_data_t is 288 bytes:
//
//	offset   0: uint8 name[32]
//	offset  32: uint8 x, y, z, pn
//	offset  36: uint8 status
//	offset  37: uint8 reserved1
//	offset  38: uint8 hiding
//	offset  39: uint8 hidden
//	offset  40: int16 reserved2
//	offset  42: int16 planet_index
//	offset  44: int16 siege_eff
//	offset  46: int16 shipyards
//	offset  48: int32 reserved4
//	offset  52: int32 IUs_needed
//	offset  56: int32 AUs_needed
//	offset  60: int32 auto_IUs
//	offset  64: int32 auto_AUs
//	offset  68: int32 reserved5
//	offset  72: int32 IUs_to_install
//	offset  76: int32 AUs_to_install
//	offset  80: int32 mi_base
//	offset  84: int32 ma_base
//	offset  88: int32 pop_units
//	offset  92: int32 item_quantity[MAX_ITEMS] (38 items)
//	offset 244: int32 reserved6
//	offset 248: int32 use_on_ambush
//	offset 252: int32 message
//	offset 256: int32 special
//	offset 260: uint8 padding[28]
const binary_nampla_data_size = 288

// encodeNamplaData translates one nampla record into its on-disk form.
func encodeNamplaData(data []byte, nampla *nampla_data_t) {
	for i := range data[:binary_nampla_data_size] {
		data[i] = 0
	}
	copyName(data[0:32], nampla.name)
	data[32] = byte(nampla.x)
	data[33] = byte(nampla.y)
	data[34] = byte(nampla.z)
	data[35] = byte(nampla.pn)
	data[36] = byte(nampla.status)
	// reserved1 at 37 stays zero
	data[38] = byte(nampla.hiding)
	data[39] = byte(nampla.hidden)
	// reserved2 at 40 stays zero
	binary.LittleEndian.PutUint16(data[42:], uint16(int16(nampla.planet_index)))
	binary.LittleEndian.PutUint16(data[44:], uint16(int16(nampla.siege_eff)))
	binary.LittleEndian.PutUint16(data[46:], uint16(int16(nampla.shipyards)))
	// reserved4 at 48 stays zero
	binary.LittleEndian.PutUint32(data[52:], uint32(int32(nampla.IUs_needed)))
	binary.LittleEndian.PutUint32(data[56:], uint32(int32(nampla.AUs_needed)))
	binary.LittleEndian.PutUint32(data[60:], uint32(int32(nampla.auto_IUs)))
	binary.LittleEndian.PutUint32(data[64:], uint32(int32(nampla.auto_AUs)))
	// reserved5 at 68 stays zero
	binary.LittleEndian.PutUint32(data[72:], uint32(int32(nampla.IUs_to_install)))
	binary.LittleEndian.PutUint32(data[76:], uint32(int32(nampla.AUs_to_install)))
	binary.LittleEndian.PutUint32(data[80:], uint32(int32(nampla.mi_base)))
	binary.LittleEndian.PutUint32(data[84:], uint32(int32(nampla.ma_base)))
	binary.LittleEndian.PutUint32(data[88:], uint32(int32(nampla.pop_units)))
	for j := 0; j < MAX_ITEMS; j++ {
		binary.LittleEndian.PutUint32(data[92+4*j:], uint32(int32(nampla.item_quantity[j])))
	}
	// reserved6 at 244 stays zero
	binary.LittleEndian.PutUint32(data[248:], uint32(int32(nampla.use_on_ambush)))
	binary.LittleEndian.PutUint32(data[252:], uint32(int32(nampla.message)))
	binary.LittleEndian.PutUint32(data[256:], uint32(int32(nampla.special)))
	// padding at 260 stays zero
}

// decodeNamplaData translates one on-disk record into a nampla record.
func decodeNamplaData(nampla *nampla_data_t, data []byte) {
	nampla.name = nameString(data[0:32])
	nampla.x = int(data[32])
	nampla.y = int(data[33])
	nampla.z = int(data[34])
	nampla.pn = int(data[35])
	nampla.status = int(data[36])
	nampla.hiding = int(data[38])
	nampla.hidden = int(data[39])
	nampla.planet_index = int(int16(binary.LittleEndian.Uint16(data[42:])))
	nampla.siege_eff = int(int16(binary.LittleEndian.Uint16(data[44:])))
	nampla.shipyards = int(int16(binary.LittleEndian.Uint16(data[46:])))
	nampla.IUs_needed = int(int32(binary.LittleEndian.Uint32(data[52:])))
	nampla.AUs_needed = int(int32(binary.LittleEndian.Uint32(data[56:])))
	nampla.auto_IUs = int(int32(binary.LittleEndian.Uint32(data[60:])))
	nampla.auto_AUs = int(int32(binary.LittleEndian.Uint32(data[64:])))
	nampla.IUs_to_install = int(int32(binary.LittleEndian.Uint32(data[72:])))
	nampla.AUs_to_install = int(int32(binary.LittleEndian.Uint32(data[76:])))
	nampla.mi_base = int(int32(binary.LittleEndian.Uint32(data[80:])))
	nampla.ma_base = int(int32(binary.LittleEndian.Uint32(data[84:])))
	nampla.pop_units = int(int32(binary.LittleEndian.Uint32(data[88:])))
	for j := 0; j < MAX_ITEMS; j++ {
		nampla.item_quantity[j] = int(int32(binary.LittleEndian.Uint32(data[92+4*j:])))
	}
	nampla.use_on_ambush = int(int32(binary.LittleEndian.Uint32(data[248:])))
	nampla.message = int(int32(binary.LittleEndian.Uint32(data[252:])))
	nampla.special = int(int32(binary.LittleEndian.Uint32(data[256:])))
}

/* load named planet data from file and create empty slots for future use */
func get_nampla_data(numNamplas, extraNamplas int, fp *os.File) []*nampla_data_t {
	if planet_base == nil {
		fmt.Fprintf(os.Stderr, "get_nampla_data: assertion failed: planet_base != NULL\n")
		os.Exit(255)
	}
	_ = extraNamplas // C allocates headroom; Go grows the slice via append later

	/* Read it all into memory. */
	binData := make([]byte, numNamplas*binary_nampla_data_size)
	if numNamplas > 0 {
		if _, err := io.ReadFull(fp, binData); err != nil {
			fmt.Fprintf(os.Stderr, "get_nampla_data: %v\n", err)
			fmt.Fprintf(os.Stderr, "\nCannot read nampla data into memory!\n")
			fmt.Fprintf(os.Stderr, "\n\tattempted to read %d nampla entries\n\n", numNamplas)
			os.Exit(255)
		}
	}
	/* translate between the structures */
	namplaData := make([]*nampla_data_t, numNamplas)
	for i := 0; i < numNamplas; i++ {
		nampla := &nampla_data_t{}
		namplaData[i] = nampla
		decodeNamplaData(nampla, binData[i*binary_nampla_data_size:])

		// mdhender: added fields to help clean up code
		nampla.id = i + 1
		nampla.planet = planet_base[nampla.planet_index]
		nampla.star = nampla.planet.star
	}

	return namplaData
}

func save_nampla_data(namplaData []*nampla_data_t, numNamplas int, fp *os.File) {
	/* translate between the structures */
	binData := make([]byte, numNamplas*binary_nampla_data_size)
	for i := 0; i < numNamplas; i++ {
		encodeNamplaData(binData[i*binary_nampla_data_size:], namplaData[i])
	}
	/* Write nampla data. */
	if numNamplas > 0 {
		if _, err := fp.Write(binData); err != nil {
			fmt.Fprintf(os.Stderr, "save_nampla_data: %v\n", err)
			fmt.Fprintf(os.Stderr, "\nCannot write nampla data to file!\n")
			fmt.Fprintf(os.Stderr, "\n\tattempted to write %d nampla entries\n\n", numNamplas)
			os.Exit(255)
		}
	}
}
