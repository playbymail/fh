package game

// Port of shipio.c. Reads and writes the ship records inside the binary
// "sp##.dat" files (driven by speciesio.go).
//
// Skipped (JSON/S-expression exporters): shipDataAsJson, shipDataAsSExpr.

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// binary_ship_data_t (data.h) is 172 bytes:
//
//	offset   0: uint8 name[32]
//	offset  32: uint8 x, y, z, pn
//	offset  36: uint8 status
//	offset  37: uint8 type
//	offset  38: uint8 dest_x, dest_y
//	offset  40: uint8 dest_z
//	offset  41: uint8 just_jumped
//	offset  42: uint8 arrived_via_wormhole
//	offset  43: uint8 reserved1
//	offset  44: int16 reserved2
//	offset  46: int16 reserved3
//	offset  48: int16 class
//	offset  50: int16 tonnage
//	offset  52: int16 item_quantity[MAX_ITEMS] (38 items)
//	offset 128: int16 age
//	offset 130: int16 remaining_cost
//	offset 132: int16 reserved4
//	offset 134: int16 loading_point
//	offset 136: int16 unloading_point
//	offset 138: 2 bytes alignment padding
//	offset 140: int32 special
//	offset 144: uint8 padding[28]
const binary_ship_data_size = 172

// encodeShipData translates one ship record into its on-disk form.
func encodeShipData(data []byte, s *ship_data_t) {
	for i := range data[:binary_ship_data_size] {
		data[i] = 0
	}
	copyName(data[0:32], s.name)
	data[32] = byte(s.x)
	data[33] = byte(s.y)
	data[34] = byte(s.z)
	data[35] = byte(s.pn)
	data[36] = byte(s.status)
	data[37] = byte(s.ship_type)
	data[38] = byte(s.dest_x)
	data[39] = byte(s.dest_y)
	data[40] = byte(s.dest_z)
	data[41] = byte(s.just_jumped)
	data[42] = byte(s.arrived_via_wormhole)
	// reserved1 at 43, reserved2 at 44, reserved3 at 46 stay zero
	binary.LittleEndian.PutUint16(data[48:], uint16(int16(s.class)))
	binary.LittleEndian.PutUint16(data[50:], uint16(int16(s.tonnage)))
	for j := 0; j < MAX_ITEMS; j++ {
		binary.LittleEndian.PutUint16(data[52+2*j:], uint16(int16(s.item_quantity[j])))
	}
	binary.LittleEndian.PutUint16(data[128:], uint16(int16(s.age)))
	binary.LittleEndian.PutUint16(data[130:], uint16(int16(s.remaining_cost)))
	// reserved4 at 132 stays zero
	binary.LittleEndian.PutUint16(data[134:], uint16(int16(s.loading_point)))
	binary.LittleEndian.PutUint16(data[136:], uint16(int16(s.unloading_point)))
	// alignment padding at 138
	binary.LittleEndian.PutUint32(data[140:], uint32(int32(s.special)))
	// padding at 144 stays zero
}

// decodeShipData translates one on-disk record into a ship record.
func decodeShipData(s *ship_data_t, data []byte) {
	s.name = nameString(data[0:32])
	s.x = int(data[32])
	s.y = int(data[33])
	s.z = int(data[34])
	s.pn = int(data[35])
	s.status = int(data[36])
	s.ship_type = int(data[37])
	s.dest_x = int(data[38])
	s.dest_y = int(data[39])
	s.dest_z = int(data[40])
	s.just_jumped = int(data[41])
	s.arrived_via_wormhole = int(data[42])
	s.class = int(int16(binary.LittleEndian.Uint16(data[48:])))
	s.tonnage = int(int16(binary.LittleEndian.Uint16(data[50:])))
	for j := 0; j < MAX_ITEMS; j++ {
		s.item_quantity[j] = int(int16(binary.LittleEndian.Uint16(data[52+2*j:])))
	}
	s.age = int(int16(binary.LittleEndian.Uint16(data[128:])))
	s.remaining_cost = int(int16(binary.LittleEndian.Uint16(data[130:])))
	s.loading_point = int(int16(binary.LittleEndian.Uint16(data[134:])))
	s.unloading_point = int(int16(binary.LittleEndian.Uint16(data[136:])))
	s.special = int(int32(binary.LittleEndian.Uint32(data[140:])))
}

/* load ship data from file and create empty slots for future use */
func get_ship_data(numShips, extraShips int, fp *os.File) []*ship_data_t {
	_ = extraShips // C allocates headroom; Go grows the slice via append later

	/* Read it all into memory. */
	binData := make([]byte, numShips*binary_ship_data_size)
	if numShips > 0 {
		if _, err := io.ReadFull(fp, binData); err != nil {
			fmt.Fprintf(os.Stderr, "get_ship_data: %v\n", err)
			fmt.Fprintf(os.Stderr, "\nCannot read ship data into memory!\n")
			fmt.Fprintf(os.Stderr, "\n\tattempted to read %d ship entries\n\n", numShips)
			os.Exit(255)
		}
	}
	/* translate between the structures */
	shipData := make([]*ship_data_t, numShips)
	for i := 0; i < numShips; i++ {
		s := &ship_data_t{}
		shipData[i] = s
		decodeShipData(s, binData[i*binary_ship_data_size:])
	}

	return shipData
}

func save_ship_data(shipData []*ship_data_t, numShips int, fp *os.File) {
	/* translate between the structures */
	binData := make([]byte, numShips*binary_ship_data_size)
	for i := 0; i < numShips; i++ {
		encodeShipData(binData[i*binary_ship_data_size:], shipData[i])
	}
	/* Write ship data. */
	if numShips > 0 {
		if _, err := fp.Write(binData); err != nil {
			fmt.Fprintf(os.Stderr, "save_ship_data: %v\n", err)
			fmt.Fprintf(os.Stderr, "\nCannot write ship data to file!\n")
			fmt.Fprintf(os.Stderr, "\n\tattempted to write %d ship entries\n\n", numShips)
			os.Exit(255)
		}
	}
}
