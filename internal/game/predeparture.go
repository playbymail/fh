package game

// Port of predeparture.c.

import (
	"fmt"
	"os"
	"strings"
)

func do_predeparture_orders() {
	var command, old_test_mode int

	if first_pass != FALSE {
		fmt.Printf("\nStart of pre-departure orders for species #%d, SP %s...\n", species_number, species.name)
	}

	truncate_name = TRUE /* For these commands, do not display age or landed/orbital status of ships. */

	for {
		command = get_command()

		if command == 0 {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! Unknown or missing command.\n")
			continue
		}

		if end_of_file != FALSE || command == END {
			if first_pass != FALSE {
				fmt.Printf("End of pre-departure orders for species #%d, SP %s.\n", species_number, species.name)
			}

			if first_pass != FALSE {
				gamemaster_abort_option()
			}

			break /* END for this species. */
		}

		switch command {
		case ALLY:
			do_ALLY_command()
		case BASE:
			do_BASE_command()
		case DEEP:
			do_DEEP_command()
		case DESTROY:
			do_DESTROY_command()
		case DISBAND:
			do_DISBAND_command()
		case ENEMY:
			do_ENEMY_command()
		case INSTALL:
			do_INSTALL_command()
		case LAND:
			do_LAND_command()
		case MESSAGE:
			do_MESSAGE_command()
		case NAME:
			do_NAME_command()
		case NEUTRAL:
			do_NEUTRAL_command()
		case ORBIT:
			do_ORBIT_command()
		case REPAIR:
			do_REPAIR_command()
		case SCAN:
			/* Scan is okay in test mode for pre-departure. */
			old_test_mode = test_mode
			test_mode = FALSE
			do_SCAN_command()
			test_mode = old_test_mode
		case SEND:
			do_SEND_command()
		case TRANSFER:
			do_TRANSFER_command()
		case UNLOAD:
			do_UNLOAD_command()
		default:
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! Invalid pre-departure command.\n")
		}
	}
}

func preDepartureCommand(args []string) int {
	do_all_species := TRUE
	dryRun := FALSE
	num_species := 0
	var sp_num [MAX_SPECIES]int

	/* Get commonly used data. */
	get_galaxy_data()
	get_star_data()
	get_planet_data()
	get_transaction_data()

	// set important globals
	ignore_field_distorters = TRUE

	/* Check arguments.
	 * If an argument is -p, then do two passes.
	 * In the first pass, display results and prompt the GM,
	 * allowing the GM to abort if necessary before saving results to disk.
	 * If an argument is -t, then set test mode.
	 * All other arguments must be species numbers.
	 * If no species numbers are specified, then do all species. */
	for i := 1; i < len(args); i++ {
		opt := args[i]
		// In C, the option is split on the first '=' and val is NULL when
		// there is no '=' or when nothing follows it.
		val := ""
		if idx := strings.IndexByte(opt, '='); idx >= 0 {
			val = opt[idx+1:]
			opt = opt[:idx]
		}
		haveVal := val != ""
		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr, "fh: usage: pre-departure [--dry-run | --test]\n")
			return 2
		} else if opt == "-p" && !haveVal {
			dryRun = TRUE
			first_pass = TRUE
		} else if opt == "-t" && !haveVal {
			test_mode = TRUE
		} else if opt == "-v" && !haveVal {
			verbose_mode = TRUE
		} else if opt == "--dry-run" && !haveVal {
			dryRun = TRUE
			first_pass = TRUE
		} else if opt == "--test" && !haveVal {
			test_mode = TRUE
		} else if !haveVal && isdigit(at(opt, 0)) {
			n := cfgAtoi(opt)
			if n < 1 || n > galaxy.num_species {
				fmt.Fprintf(os.Stderr, "error: preDepartureCommand: '%s' is not a valid species number!\n", opt)
				return 2
			}
			found := FALSE
			for j := 0; found == FALSE && j < num_species; j++ {
				if sp_num[j] == n {
					found = TRUE
				}
			}
			if found == FALSE {
				sp_num[num_species] = n
				num_species++
			}
			do_all_species = FALSE
		} else {
			fmt.Fprintf(os.Stderr, "error: unknown option '%s'\n", opt)
			return 2
		}
	}
	_ = dryRun

	if num_species == 0 || do_all_species != FALSE {
		num_species = galaxy.num_species
		for num_species = 0; num_species < galaxy.num_species; num_species++ {
			sp_num[num_species] = num_species + 1
		}
		do_all_species = TRUE
	}

	/* Two passes through all orders will be done.
	 * The first pass will check for errors and abort if any are found.
	 * Results will be written to disk only on the second pass. */
	if first_pass != FALSE {
		fmt.Printf("\nStarting first pass...\n\n")
		preDeparturePass(sp_num[:], do_all_species, TRUE)
	}
	preDeparturePass(sp_num[:], do_all_species, FALSE)

	// save any updates
	if star_data_modified != FALSE {
		save_star_data()
	}
	if planet_data_modified != FALSE {
		save_planet_data()
	}
	save_species_data()
	save_transaction_data()

	return 0
}

// Note: the first_pass parameter intentionally shadows the global, exactly
// as the C parameter shadows the enginevars global. do_predeparture_orders
// still reads the global.
func preDeparturePass(sp_num []int, do_all_species, first_pass int) int {
	get_star_data()
	get_planet_data()
	get_species_data()

	/* Main loop. For each species, take appropriate action. */
	// The C loop condition is `sp_index < sp_num[sp_index] != 0`, which
	// parses as `(sp_index < sp_num[sp_index]) != 0`; the zero entries at
	// the end of sp_num terminate the loop. The extra bounds check guards
	// the indexing that C leaves unchecked.
	for sp_index := 0; sp_index < len(sp_num) && sp_index < sp_num[sp_index]; sp_index++ {
		species_number = sp_num[sp_index]
		species_index = species_number - 1

		found := data_in_memory[species_index]
		if found == FALSE {
			if do_all_species == FALSE {
				fmt.Fprintf(os.Stderr, "\n    Cannot get data for species #%d!\n", species_number)
				return 2
			}
			if first_pass != FALSE {
				fmt.Printf("\n    Skipping species #%d.\n", species_number)
			}
			continue
		}

		result := preDepartureSpecies(species_number, do_all_species, first_pass)
		if result != 0 {
			fmt.Fprintf(os.Stderr, "error: unable to process pre-departure errors for species #%d\n", species_number)
			return result
		}
	}

	return 0
}

func preDepartureSpecies(spNo, do_all_species, first_pass int) int {
	species_number = spNo
	species_index = species_number - 1

	species = &spec_data[species_index]
	nampla_base = namp_data[species_index]
	ship_base = ship_data[species_index]

	/* Open orders file for this species. */
	filename := fmt.Sprintf("sp%02d.ord", species_number)
	input_file = fopen_r(filename)
	if input_file == nil {
		if do_all_species == FALSE {
			fmt.Fprintf(os.Stderr, "\n\tCannot open '%s' for reading!\n\n", filename)
			return 2
		}
		if first_pass != FALSE {
			fmt.Printf("\n    No orders for species #%d.\n", species_number)
		}
		return 0
	}

	end_of_file = FALSE
	just_opened_file = TRUE /* Tell command parser to skip mail header, if any. */

	/* Search for START PRE-DEPARTURE order. */
	foundStart := FALSE
	for found := FALSE; found == FALSE; {
		command := get_command()
		if command < 0 {
			/* End of file. */
			break
		} else if command == MESSAGE {
			/* Skip MESSAGE text. It may contain a line that starts with "start". */
			for command = get_command(); command != ZZZ; command = get_command() {
				if command < 0 {
					fmt.Fprintf(os.Stderr, "WARNING: Unterminated MESSAGE command in file '%s'!\n", filename)
					input_file.fclose()
					input_file = nil
					return 0
				}
			}
		} else if command == START {
			/* Get the first three letters of the keyword and convert to upper case. */
			skip_whitespace()

			var keyword [3]byte
			for i := 0; i < 3; i++ {
				keyword[i] = toupper(at(input_line, input_line_pointer))
				input_line_pointer++
			}

			if string(keyword[:]) == "PRE" {
				found = TRUE
				foundStart = TRUE
			}
		}
	}

	if foundStart == FALSE {
		if first_pass != FALSE {
			fmt.Printf("\nNo pre-departure orders for species #%d, SP %s.\n", species_number, species.name)
		}
		input_file.fclose()
		input_file = nil
		return 0
	}

	/* Open log file. Use stdout for first pass. */
	log_stdout = FALSE /* We will control value of log_file from here. */
	if first_pass != FALSE {
		log_file = os.Stdout
	} else {
		/* Open log file for appending. */
		filename = fmt.Sprintf("sp%02d.log", species_number)
		var err error
		log_file, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log_file = nil
			cfgPerror("preDepartureSpecies:", err)
			fmt.Fprintf(os.Stderr, "\n\tCannot open '%s' for appending!\n\n", filename)
			os.Exit(2)
		}
		log_string("\nPre-departure orders:\n")
	}

	/* Handle predeparture orders for this species. */
	do_predeparture_orders()

	data_modified[species_index] = TRUE

	/* If this is the second pass, close the log file. */
	if first_pass == FALSE {
		log_file.Close()
	}

	input_file.fclose()
	input_file = nil

	return 0
}
