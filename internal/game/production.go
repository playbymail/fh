package game

// Port of production.c.

import (
	"fmt"
	"os"
	"strings"
)

func do_production_orders() {
	var i, command int

	truncate_name = TRUE /* For these commands, do not display age or landed/orbital status of ships. */

	if first_pass != FALSE {
		fmt.Printf("\nStart of production orders for species #%d, SP %s...\n", species_number, species.name)
	}

	doing_production = FALSE /* This will be set as soon as production actually starts. */
	for {
		command = get_command()

		if command == 0 {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! Unknown or missing command.\n")
			continue
		}

		if end_of_file != FALSE || command == END {
			/* Handle planets that were not given PRODUCTION orders. */
			for i = 0; i < species.num_namplas; i++ {
				next_nampla = nampla_base[i]

				if production_done[i] != FALSE {
					continue
				}

				production_done[i] = TRUE

				if next_nampla.status&DISBANDED_COLONY != 0 {
					continue
				}

				if next_nampla.mi_base+next_nampla.ma_base == 0 {
					continue
				}

				next_nampla_index = i

				do_PRODUCTION_command(TRUE)
			}

			transfer_balance() /* Terminate production for last planet for this species. */

			if first_pass != FALSE {
				gamemaster_abort_option()
				fmt.Printf("\nEnd of production orders for species #%d, SP %s.\n", species_number, species.name)
			}

			break /* END for this species. */
		}

		switch command {
		case ALLY:
			do_ALLY_command()
		case AMBUSH:
			do_AMBUSH_command()
		case BUILD:
			do_BUILD_command(FALSE, FALSE)
		case CONTINUE:
			do_BUILD_command(TRUE, FALSE)
		case DEVELOP:
			do_DEVELOP_command()
		case ENEMY:
			do_ENEMY_command()
		case ESTIMATE:
			do_ESTIMATE_command()
		case HIDE:
			do_HIDE_command()
		case IBUILD:
			do_BUILD_command(FALSE, TRUE)
		case ICONTINUE:
			do_BUILD_command(TRUE, TRUE)
		case INTERCEPT:
			do_INTERCEPT_command()
		case NEUTRAL:
			do_NEUTRAL_command()
		case PRODUCTION:
			do_PRODUCTION_command(FALSE)
		case RECYCLE:
			do_RECYCLE_command()
		case RESEARCH:
			do_RESEARCH_command()
		case SHIPYARD:
			do_SHIPYARD_command()
		case UPGRADE:
			do_UPGRADE_command()
		default:
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! Invalid production command.\n")
		}
	}
}

func productionCommand(args []string) int {
	do_all_species := TRUE
	dryRun := FALSE
	num_species := 0
	var sp_num [MAX_SPECIES]int

	first_pass = FALSE
	ignore_field_distorters = TRUE

	/* Get commonly used data. */
	get_galaxy_data()
	get_star_data()
	get_planet_data()
	get_transaction_data()

	/* Check arguments.
	 * If an argument is -p, then do two passes.
	 * In the first pass, display results and prompt the GM,
	 * allowing the GM to abort if necessary before saving results to disk.
	 * All other arguments must be species numbers.
	 * If no species numbers are specified, then do all species. */
	for i := 1; i < len(args); i++ {
		opt := args[i]
		val := "" // mirrors the C `char *val = NULL` ("" plays the role of NULL)
		if eq := strings.IndexByte(opt, '='); eq >= 0 {
			val = opt[eq+1:]
			opt = opt[:eq]
		}
		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr, "usage: production [--dry-run | --test]\n")
			return 2
		} else if opt == "-p" && val == "" {
			dryRun = TRUE
			first_pass = TRUE
		} else if opt == "-t" && val == "" {
			test_mode = TRUE
		} else if opt == "-v" && val == "" {
			verbose_mode = TRUE
		} else if opt == "--dry-run" && val == "" {
			dryRun = TRUE
			first_pass = TRUE
		} else if opt == "--test" && val == "" {
			test_mode = TRUE
		} else if val == "" && isdigit(at(opt, 0)) {
			n := cfgAtoi(opt)
			if n < 1 || n > galaxy.num_species {
				fmt.Fprintf(os.Stderr, "error: productionCommand: '%s' is not a valid species number!\n", opt)
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
	_ = dryRun // set but never read, as in the C original

	if num_species == 0 {
		do_all_species = TRUE
		num_species = galaxy.num_species
		for i := 0; i < num_species; i++ {
			sp_num[i] = i + 1
		}
	} else {
		do_all_species = FALSE
		// sort the species to get consistency on output
		for i := 0; i < num_species; i++ {
			for j := i + 1; j < num_species; j++ {
				if sp_num[j] < sp_num[i] {
					tmp := sp_num[i]
					sp_num[i] = sp_num[j]
					sp_num[j] = tmp
				}
			}
		}
	}

	/* Two passes through all orders will be done.
	 * The first pass will check for errors and abort if any are found.
	 * Results will be written to disk only on the second pass. */
	if first_pass != FALSE {
		fmt.Printf("\nStarting first pass...\n\n")
		productionPass(sp_num[:], num_species, do_all_species, TRUE)
	}
	productionPass(sp_num[:], num_species, do_all_species, FALSE)

	save_species_data()

	if planet_data_modified != FALSE {
		save_planet_data()
	}

	save_transaction_data()

	free_species_data()
	planet_base = nil

	return 0
}

// Note: the first_pass parameter shadows the global first_pass, exactly as
// the C function's parameter shadows the global within this function.
func productionPass(sp_num []int, num_species, do_all_species, first_pass int) int {
	if first_pass != FALSE {
		fmt.Printf("\nStarting first pass...\n\n")
	}

	get_species_data()

	/* Main loop. For each species, take appropriate action. */
	for sp_index := 0; sp_index < num_species; sp_index++ {
		species_number = sp_num[sp_index]
		species_index = species_number - 1

		found := data_in_memory[species_index]
		if found == FALSE {
			if do_all_species != FALSE {
				if first_pass != FALSE {
					fmt.Printf("\n    Skipping species #%d.\n", species_number)
				}
				continue
			} else {
				fmt.Fprintf(os.Stderr, "\n    Cannot get data for species #%d!\n", species_number)
				os.Exit(2)
			}
		}

		result := productionPassSpecies(species_number, do_all_species, first_pass)
		if result != 0 {
			// ?
		}
	}

	if first_pass != FALSE {
		fmt.Printf("\nFinal chance to abort safely!\n")

		gamemaster_abort_option()

		first_pass = FALSE

		free_species_data()

		star_base = nil   /* In case data was modified. */
		planet_base = nil /* In case data was modified. */

		fmt.Printf("\nStarting second pass...\n\n")
	}

	return 0
}

// Note: the first_pass parameter shadows the global first_pass, exactly as
// the C function's parameter shadows the global within this function.
func productionPassSpecies(spNo, do_all_species, first_pass int) int {
	_ = spNo // present in the C signature but unused there as well

	species = &spec_data[species_index]
	nampla_base = namp_data[species_index]
	ship_base = ship_data[species_index]

	home_planet = planet_base[nampla_base[0].planet_index]

	/* Open orders file for this species. */
	filename := fmt.Sprintf("sp%02d.ord", species_number)
	input_file = fopen_r(filename)
	if input_file == nil {
		if do_all_species != FALSE {
			if first_pass != FALSE {
				fmt.Printf("\n    No orders for species #%d.\n", species_number)
			}
			return 0
		} else {
			fmt.Fprintf(os.Stderr, "\n\tCannot open '%s' for reading!\n\n", filename)
			os.Exit(2)
		}
	}

	end_of_file = FALSE
	just_opened_file = TRUE /* Tell command parser to skip mail header, if any. */

	/* Search for START PRODUCTION order. */
	found := FALSE
	for found == FALSE {
		command := get_command()
		if command == MESSAGE {
			/* Skip MESSAGE text. It may contain a line that starts with "start". */
			for command > 0 && command != ZZZ {
				command = get_command()
			}
			if command < 0 {
				/* End of file. */
				fmt.Fprintf(os.Stderr, "WARNING: Unterminated MESSAGE command in file '%s'!\n", filename)
			}
		}

		if command == START {
			/* Get the first three letters of the keyword and convert to upper case. */
			skip_whitespace()
			keyword := ""
			for i := 0; i < 3 && at(input_line, input_line_pointer) != 0; i++ {
				keyword += string(toupper(at(input_line, input_line_pointer)))
				input_line_pointer++
			}
			if keyword == "PRO" {
				found = TRUE
			}
		}

		if command < 0 {
			/* End of file. */
			if first_pass != FALSE {
				fmt.Printf("\nNo production orders for species #%d, SP %s.\n", species_number, species.name)
			}
			input_file.fclose()
			input_file = nil
			return 0
		}
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
			fmt.Fprintf(os.Stderr, "\n\tCannot open '%s' for appending!\n\n", filename)
			os.Exit(2)
		}
		fmt.Fprintf(log_file, "\nProduction orders:\n")
		fmt.Fprintf(log_file, "\n  Number of economic units at start of production: %d\n\n", species.econ_units)
	}

	// initialize arrays
	for i := 0; i < species.num_namplas; i++ {
		// production_done prevents more than one set of orders per planet
		if i > 999 {
			fmt.Fprintf(os.Stderr, "\n\n\tInternal error. production_done array overflow!\n\n")
			os.Exit(2)
		}
		production_done[i] = FALSE

		// do other initializations
		nampla = nampla_base[i]
		nampla.auto_IUs = 0
		nampla.auto_AUs = 0
		nampla.IUs_needed = 0
		nampla.AUs_needed = 0
	}

	/* Handle production orders for this species. */
	num_intercepts = 0
	for i := 0; i < 6; i++ {
		sp_tech_level[i] = species.tech_level[i]
	}

	do_production_orders()

	for i := 0; i < 6; i++ {
		species.tech_level[i] = sp_tech_level[i]
	}

	for i := 0; i < num_intercepts; i++ {
		handle_intercept(i)
	}

	data_modified[species_index] = TRUE

	/* If this is the second pass, close the log file. */
	if first_pass == FALSE {
		log_file.Close()
	}

	input_file.fclose()
	input_file = nil

	return 0
}
