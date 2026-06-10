package game

// Port of import.c — reads the JSON files produced by the export
// command back into engine state and saves the binary .dat files.

import (
	"fmt"
	"os"
	"strings"
)

// importCommand mirrors the C entry point; args[0] is the command name.
func importCommand(args []string) int {
	doImportJson := FALSE
	doTest := FALSE

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
			fmt.Fprintf(os.Stderr, "usage: import json\n")
			return 2
		} else if opt == "-t" && !hasVal {
			doTest = TRUE
		} else if opt == "--test" && !hasVal {
			doTest = TRUE
		} else if opt == "json" && !hasVal {
			doImportJson = TRUE
		} else {
			if hasVal {
				fmt.Fprintf(os.Stderr, "import: unknown option '%s=%s'\n", opt, val)
			} else {
				// the C passes a NULL char* to %s here; the C library
				// renders that as "(null)"
				fmt.Fprintf(os.Stderr, "import: unknown option '%s(null)'\n", opt)
			}
			return 2
		}
	}

	if doImportJson != FALSE {
		return importFromJson(doTest)
	}

	return 0
}

func importFromJson(doTest int) int {
	fmt.Printf(" info: loading binary data...\n")
	get_galaxy_data()
	get_star_data()
	get_planet_data()
	get_species_data()

	fmt.Printf(" info: importing galaxy.json...\n")
	root := jsonParseFile("galaxy.json")
	unmarshalGalaxyFile(root, &galaxy)
	cJSON_Delete(root)

	fmt.Printf(" info: importing systems.json...\n")
	root = jsonParseFile("systems.json")
	unmarshalSystemsFile(root, star_base, planet_base)
	cJSON_Delete(root)

	for i := 0; i < MAX_SPECIES; i++ {
		if data_in_memory[i] != FALSE {
			filename := fmt.Sprintf("species.%03d.json", i+1)
			if _, err := os.Stat(filename); err != nil {
				// assume that file is missing
				fmt.Printf(" warn: missing species file '%s'\n", filename)
				continue
			}
			fmt.Printf(" info: importing %s...\n", filename)
			root = jsonParseFile(filename)
			namp_data[i], ship_data[i] = unmarshalSpeciesFile(root, &spec_data[i], namp_data[i], ship_data[i])
			data_modified[i] = TRUE
		}
	}

	if doTest != FALSE {
		fmt.Printf(" test: changes not saved\n")
		return 0
	}

	fmt.Printf(" info: saving binary data...\n")
	save_galaxy_data()
	save_star_data()
	save_planet_data()
	save_species_data()
	fmt.Printf(" info: import and save complete\n")

	return 0
}
