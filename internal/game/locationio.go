package game

// Port of locationio.c. Reads and writes the binary file "locations.dat".
//
// NOTE: the C file defines a file-local struct confusingly named
// binary_ship_data_t (4 bytes: uint8 s, x, y, z). That local layout is
// what is ported here, not the ship record from data.h.
//
// Skipped (JSON/S-expression exporters): locationDataAsJson,
// locationDataAsSExpr.

import (
	"fmt"
	"os"
)

// binary_location_data is 4 bytes:
//
//	offset 0: uint8 s (species number)
//	offset 1: uint8 x
//	offset 2: uint8 y
//	offset 3: uint8 z
const binary_location_data_size = 4

func get_location_data() {
	/* Get size of file. */
	sb, err := os.Stat("locations.dat")
	if err != nil {
		num_locs = 0
		return
	}

	// get number of records in the file
	num_locs = int(sb.Size()) / binary_location_data_size
	if sb.Size() != int64(num_locs*binary_location_data_size) {
		fmt.Fprintf(os.Stderr, "\nFile locations.dat contains extra bytes (%d > %d)!\n\n",
			sb.Size(), num_locs*binary_location_data_size)
		os.Exit(255)
	} else if num_locs == 0 {
		// nothing to do
		return
	} else if num_locs > MAX_LOCATIONS {
		fmt.Fprintf(os.Stderr, "\nFile locations.dat contains too many records (%d > %d)!\n\n", num_locs, MAX_LOCATIONS)
		os.Exit(255)
	}

	/* Open locations file and read it all into memory. */
	binData, err := os.ReadFile("locations.dat")
	if err != nil {
		fmt.Fprintf(os.Stderr, "get_location_data: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nCannot open file 'locations.dat' for reading!\n\n")
		os.Exit(255)
	}
	if len(binData) < num_locs*binary_location_data_size {
		fmt.Fprintf(os.Stderr, "\nCannot read file 'locations.dat' into memory!\n")
		fmt.Fprintf(os.Stderr, "\n\tattempted to read %d location entries\n\n", num_locs)
		os.Exit(255)
	}

	/* translate data */
	for i := 0; i < num_locs; i++ {
		rec := binData[i*binary_location_data_size:]
		loc[i].s = int(rec[0])
		loc[i].x = int(rec[1])
		loc[i].y = int(rec[2])
		loc[i].z = int(rec[3])
	}
}

func save_location_data() {
	/* Open file 'locations.dat' for writing. */
	fp, err := os.Create("locations.dat")
	if err != nil {
		fmt.Fprintf(os.Stderr, "save_location_data: %v\n", err)
		fmt.Fprintf(os.Stderr, "\n\tCannot create file 'locations.dat'!\n\n")
		os.Exit(255)
	}

	if num_locs > 0 {
		binData := make([]byte, num_locs*binary_location_data_size)

		/* translate data */
		for i := 0; i < num_locs; i++ {
			rec := binData[i*binary_location_data_size:]
			rec[0] = byte(loc[i].s)
			rec[1] = byte(loc[i].x)
			rec[2] = byte(loc[i].y)
			rec[3] = byte(loc[i].z)
		}

		/* Write array to disk. */
		if _, err := fp.Write(binData); err != nil {
			fmt.Fprintf(os.Stderr, "save_location_data: %v\n", err)
			fmt.Fprintf(os.Stderr, "\n\n\tCannot write to 'locations.dat'!\n\n")
			os.Exit(255)
		}
	}
	fp.Close()
}
