package game

// Port of export.c — converts the binary .dat files to JSON using the
// marshal.go tree builders. The exportSpecies debug dump reads the raw
// binary_species_data_t/binary_nampla_data_t/binary_ship_data_t records
// (layouts documented in speciesio.go, namplaio.go and shipio.go) and
// prints every field, reserved and padding bytes included.

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

// exportCommand mirrors the C entry point; args[0] is the command name.
func exportCommand(args []string) int {
	for i := 1; i < len(args); i++ {
		opt := args[i]
		var val string
		hasVal := false
		if idx := strings.IndexByte(opt, '='); idx >= 0 {
			val = opt[idx+1:]
			opt = opt[:idx]
			hasVal = val != ""
		}

		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr, "usage: export json\n")
			return 2
		} else if opt == "--species" && hasVal && val == "05" {
			exportSpecies(5)
		} else if opt == "json" && !hasVal {
			return exportToJson()
		} else {
			fmt.Fprintf(os.Stderr, "fh: export: unknown option '%s'\n", opt)
			return 2
		}
	}

	return 0
}

func exportSpecies(spNo int) int {
	filename := fmt.Sprintf("sp%02d.dat", spNo)
	fp, err := os.Open(filename)
	if err != nil {
		jsonPerror(filename, err)
		os.Exit(2)
	}
	bytesRead := 0
	sp := make([]byte, binary_species_data_size)
	if _, err := io.ReadFull(fp, sp); err != nil {
		jsonPerror(filename, err)
		os.Exit(2)
	} else {
		bytesRead += binary_species_data_size
	}
	rd16 := func(data []byte, off int) int { return int(int16(binary.LittleEndian.Uint16(data[off:]))) }
	rd32 := func(data []byte, off int) int { return int(int32(binary.LittleEndian.Uint32(data[off:]))) }
	rdu32 := func(data []byte, off int) uint32 { return binary.LittleEndian.Uint32(data[off:]) }
	// hex16/hex32 reproduce C's default argument promotion for the
	// 0x%04x / 0x%08x conversions of signed fields.
	hex16 := func(data []byte, off int) uint32 { return uint32(int32(int16(binary.LittleEndian.Uint16(data[off:])))) }
	hex32 := func(data []byte, off int) uint32 { return binary.LittleEndian.Uint32(data[off:]) }

	fmt.Printf("              name: '%s'\n", nameString(sp[0:32]))
	fmt.Printf("         govt_name: '%s'\n", nameString(sp[32:64]))
	fmt.Printf("         govt_type: '%s'\n", nameString(sp[64:96]))
	fmt.Printf("                 x: %3d\n", sp[96])
	fmt.Printf("                 y: %3d\n", sp[97])
	fmt.Printf("                 z: %3d\n", sp[98])
	fmt.Printf("                pn: %3d\n", sp[99])
	fmt.Printf("      required_gas: %3d\n", sp[100])
	fmt.Printf("  required_gas_min: %3d\n", sp[101])
	fmt.Printf("  required_gas_max: %3d\n", sp[102])
	fmt.Printf("         reserved5: 0x%02x\n", sp[103])
	fmt.Printf("       neutral_gas: [%3d, %3d, %3d, %3d, %3d, %3d]\n",
		sp[104], sp[105], sp[106],
		sp[107], sp[108], sp[109])
	fmt.Printf("        poison_gas: [%3d, %3d, %3d, %3d, %3d, %3d]\n",
		sp[110], sp[111], sp[112],
		sp[113], sp[114], sp[115])
	fmt.Printf("       auto_orders: 0x%02x\n", sp[116])
	fmt.Printf("         reserved3: 0x%02x\n", sp[117])
	fmt.Printf("         reserved4: 0x%04x\n", hex16(sp, 118))
	fmt.Printf("        tech_level: [%4d, %4d, %4d, %4d, %4d, %4d]\n",
		rd16(sp, 120), rd16(sp, 122), rd16(sp, 124), rd16(sp, 126), rd16(sp, 128), rd16(sp, 130))
	fmt.Printf("   init_tech_level: [%4d, %4d, %4d, %4d, %4d, %4d]\n",
		rd16(sp, 132), rd16(sp, 134), rd16(sp, 136), rd16(sp, 138),
		rd16(sp, 140), rd16(sp, 142))
	fmt.Printf("    tech_knowledge: [%4d, %4d, %4d, %4d, %4d, %4d]\n",
		rd16(sp, 144), rd16(sp, 146), rd16(sp, 148), rd16(sp, 150), rd16(sp, 152),
		rd16(sp, 154))
	fmt.Printf("       num_namplas: %6d\n", rd32(sp, 156))
	fmt.Printf("         num_ships: %6d\n", rd32(sp, 160))
	fmt.Printf("          tech_eps: [%4d, %4d, %4d, %4d, %4d, %4d]\n",
		rd32(sp, 164), rd32(sp, 168), rd32(sp, 172), rd32(sp, 176), rd32(sp, 180), rd32(sp, 184))
	fmt.Printf("  hp_original_base: %12d\n", rd32(sp, 188))
	fmt.Printf("        econ_units: %12d\n", rd32(sp, 192))
	fmt.Printf("        fleet_cost: %12d\n", rd32(sp, 196))
	fmt.Printf("fleet_percent_cost: %4d\n", rd32(sp, 200))
	// NOTE: the C original prints contact[1] twice (and never contact[0]);
	// the bug is preserved here.
	fmt.Printf("           contact: [%8x %8x %8x %8x]\n", rdu32(sp, 208), rdu32(sp, 208), rdu32(sp, 212), rdu32(sp, 216))
	fmt.Printf("              ally: [%8x %8x %8x %8x]\n", rdu32(sp, 220), rdu32(sp, 224), rdu32(sp, 228), rdu32(sp, 232))
	fmt.Printf("             enemy: [%8x %8x %8x %8x]\n", rdu32(sp, 236), rdu32(sp, 240), rdu32(sp, 244), rdu32(sp, 248))
	fmt.Printf("           padding: [")
	for n := 0; n < 12; n++ {
		if n > 0 && (n%10 == 0) {
			fmt.Printf("\n                     ")
		}
		fmt.Printf(" %2x", sp[252+n])
	}
	fmt.Printf("]\n")
	fmt.Printf("------------------: -----------------------------------------------\n")

	numNamplas := rd32(sp, 156)
	numShips := rd32(sp, 160)

	fmt.Printf("---- named_planets: -----------------------------------------------\n")
	for i := 0; i < numNamplas; i++ {
		npd := make([]byte, binary_nampla_data_size)
		if _, err := io.ReadFull(fp, npd); err != nil {
			jsonPerror("named planet data:", err)
			os.Exit(2)
		} else {
			bytesRead += binary_nampla_data_size
		}
		fmt.Printf("          name: '%s'\n", nameString(npd[0:32]))
		fmt.Printf("   nampla_name: ")
		for n := 0; n < 32; n++ {
			fmt.Printf(" %2x", npd[n])
		}
		fmt.Printf("]\n")
		fmt.Printf("             x: %3d\n", npd[32])
		fmt.Printf("             y: %3d\n", npd[33])
		fmt.Printf("             z: %3d\n", npd[34])
		fmt.Printf("            pn: %3d\n", npd[35])
		fmt.Printf("        status: 0x%02x\n", npd[36])
		fmt.Printf("     reserved1: 0x%02x\n", npd[37])
		fmt.Printf("        hiding: 0x%02x\n", npd[38])
		fmt.Printf("        hidden: 0x%02x\n", npd[39])
		fmt.Printf("     reserved2: 0x%04x\n", hex16(npd, 40))
		fmt.Printf("  planet_index: %4d\n", rd16(npd, 42))
		fmt.Printf("     siege_eff: %4d\n", rd16(npd, 44))
		fmt.Printf("     shipyards: %4d\n", rd16(npd, 46))
		fmt.Printf("     reserved4: 0x%08x\n", hex32(npd, 48))
		fmt.Printf("    IUs_needed: %12d\n", rd32(npd, 52))
		fmt.Printf("    AUs_needed: %12d\n", rd32(npd, 56))
		fmt.Printf("      auto_IUs: %12d\n", rd32(npd, 60))
		fmt.Printf("      auto_AUs: %12d\n", rd32(npd, 64))
		fmt.Printf("     reserved5: 0x%08x\n", hex32(npd, 68))
		fmt.Printf("IUs_to_install: %12d\n", rd32(npd, 72))
		fmt.Printf("AUs_to_install: %12d\n", rd32(npd, 76))
		fmt.Printf("       mi_base: %12d\n", rd32(npd, 80))
		fmt.Printf("       ma_base: %12d\n", rd32(npd, 84))
		fmt.Printf("     pop_units: %12d\n", rd32(npd, 88))
		fmt.Printf(" item_quantity: [")
		for n := 0; n < MAX_ITEMS; n++ {
			if n > 0 && (n%10 == 0) {
				fmt.Printf("\n                 ")
			}
			fmt.Printf(" %4d", rd32(npd, 92+4*n))
		}
		fmt.Printf("]\n")
		fmt.Printf("     reserved6: 0x%08x\n", hex32(npd, 244))
		fmt.Printf(" use_on_ambush: %12d\n", rd32(npd, 248))
		fmt.Printf("       message: %12d\n", rd32(npd, 252))
		fmt.Printf("       special: 0x%08x\n", hex32(npd, 256))
		fmt.Printf("       padding:")
		for n := 0; n < 28; n++ {
			if n == 10 || n == 20 {
				fmt.Printf("\n               ")
			}
			fmt.Printf(" %2x", npd[260+n])
		}
		fmt.Printf("\n")
	}
	fmt.Printf("------------------: -----------------------------------------------\n")

	fmt.Printf("-------------- ships: ---------------------------------------------\n")
	for i := 0; i < numShips; i++ {
		sd := make([]byte, binary_ship_data_size)
		if _, err := io.ReadFull(fp, sd); err != nil {
			jsonPerror("ship data:", err)
			os.Exit(2)
		} else {
			bytesRead += binary_ship_data_size
		}
		fmt.Printf("                name: '%s'\n", nameString(sd[0:32]))
		fmt.Printf("           ship_name: [")
		for n := 0; n < 32; n++ {
			fmt.Printf(" %2x", sd[n])
		}
		fmt.Printf("]\n")
		fmt.Printf("                   x: %3d\n", sd[32])
		fmt.Printf("                   y: %3d\n", sd[33])
		fmt.Printf("                   z: %3d\n", sd[34])
		fmt.Printf("                  pn: %3d\n", sd[35])
		fmt.Printf("                type: %3d\n", sd[37])
		fmt.Printf("              dest_x: %3d\n", sd[38])
		fmt.Printf("              dest_y: %3d\n", sd[39])
		fmt.Printf("              dest_z: %3d\n", sd[40])
		fmt.Printf("         just_jumped: 0x%02x\n", sd[41])
		fmt.Printf("arrived_via_wormhole: 0x%02x\n", sd[42])
		fmt.Printf("           reserved1: 0x%02x\n", sd[43])
		fmt.Printf("           reserved2: 0x%04x\n", hex16(sd, 44))
		fmt.Printf("           reserved3: 0x%04x\n", hex16(sd, 46))
		fmt.Printf("               class: %8d\n", rd16(sd, 48))
		fmt.Printf("             tonnage: %8d\n", rd16(sd, 50))
		fmt.Printf("       item_quantity: [")
		for n := 0; n < MAX_ITEMS; n++ {
			if n > 0 && (n%10 == 0) {
				fmt.Printf("\n                       ")
			}
			fmt.Printf(" %4d", rd16(sd, 52+2*n))
		}
		fmt.Printf("]\n")
		fmt.Printf("                 age: %8d\n", rd16(sd, 128))
		fmt.Printf("      remaining_cost: %8d\n", rd16(sd, 130))
		fmt.Printf("           reserved4: 0x%08x\n", hex16(sd, 132))
		fmt.Printf("       loading_point: %8d\n", rd16(sd, 134))
		fmt.Printf("     unloading_point: %8d\n", rd16(sd, 136))
		fmt.Printf("             special: %8d\n", rd32(sd, 140))
		fmt.Printf("             padding:")
		for n := 0; n < 28; n++ {
			if n == 10 || n == 20 {
				fmt.Printf("\n                     ")
			}
			fmt.Printf(" %2x", sd[144+n])
		}
		fmt.Printf("\n")
	}
	fmt.Printf("--------------------: ---------------------------------------------\n")

	fmt.Printf("  bytes_read: %12d\n", bytesRead)
	bytesRead = 0
	buffer := make([]byte, 128)
	for {
		n, err := fp.Read(buffer)
		bytesRead += n
		if err != nil {
			break
		}
	}
	fmt.Printf("excess_bytes: %12d\n", bytesRead)

	fp.Close()

	return 0
}

func exportToJson() int {
	fmt.Printf(" info: loading binary data...\n")
	get_galaxy_data()
	get_star_data()
	get_planet_data()
	get_species_data()

	fmt.Printf(" info: exporting galaxy.json...\n")
	root := marshalGalaxyFile()
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: there was an error converting galaxy data to json\n")
		os.Exit(2)
	}
	jsonWriteFile(root, "galaxy", "galaxy.json")
	cJSON_Delete(root)

	fmt.Printf(" info: exporting systems.json...\n")
	root = marshalSystemsFile()
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: there was an error converting systems data to json\n")
		os.Exit(2)
	}
	jsonWriteFile(root, "systems", "systems.json")
	cJSON_Delete(root)

	for i := 0; i < MAX_SPECIES; i++ {
		if data_in_memory[i] != FALSE {
			root = marshalSpeciesFile(&spec_data[i], namp_data[i], ship_data[i])
			if root == nil {
				fmt.Fprintf(os.Stderr, "error: there was an error converting species data to json\n")
				os.Exit(2)
			}
			filename := fmt.Sprintf("species.%03d.json", i+1)
			fmt.Printf(" info: exporting %s...\n", filename)
			jsonWriteFile(root, "species", filename)
			cJSON_Delete(root)
		}
	}

	fmt.Printf(" info: export complete\n")

	return 0
}
