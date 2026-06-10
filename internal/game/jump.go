package game

// Port of jump.c.

import (
	"fmt"
	"os"
	"strings"
)

func do_jump_orders() {
	var command int

	if first_pass != FALSE {
		fmt.Printf("\nStart of jump orders for species #%d, SP %s...\n", species_number, species.name)
	}

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
				fmt.Printf("End of jump orders for species #%d, SP %s.\n", species_number, species.name)
			}
			if first_pass != FALSE {
				gamemaster_abort_option()
			}
			break /* END for this species. */
		}

		switch command {
		case JUMP:
			do_JUMP_command(FALSE, FALSE)
		case MOVE:
			do_MOVE_command()
		case PJUMP:
			do_JUMP_command(FALSE, TRUE)
		case VISITED:
			do_VISITED_command()
		case WORMHOLE:
			do_WORMHOLE_command()
		default:
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! Invalid jump command.\n")
		}
	}
}

func jumpCommand(args []string) int {
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
		// In C, the option is split on the first '=' and val is NULL when
		// there is no '=' or when nothing follows it.
		val := ""
		if idx := strings.IndexByte(opt, '='); idx >= 0 {
			val = opt[idx+1:]
			opt = opt[:idx]
		}
		haveVal := val != ""
		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr, "usage: jump [--dry-run | --test]\n")
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
				fmt.Fprintf(os.Stderr, "error: jumpCommand: '%s' is not a valid species number!\n", opt)
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

	/* For these commands, do not display age or landed/orbital status of ships. */
	truncate_name = TRUE
	log_stdout = FALSE /* We will control value of log_file from here. */

	/* Initialize array that will indicate which species provided jump orders.
	 * If ships of a species withdrew or were forced from combat and there were
	 * no jump orders for that species, then combat jumps will not take place.
	 * This array will allow us to handle them separately. */
	var species_jumped [MAX_SPECIES]int
	for i := 0; i < galaxy.num_species; i++ {
		species_jumped[i] = FALSE
	}

	/* Two passes through all orders will be done.
	 * The first pass will  check for errors and abort if any are found.
	 * Results will be written to disk only on the second pass. */

start_pass:

	if first_pass != FALSE {
		fmt.Printf("\nStarting first pass...\n\n")
	}

	get_star_data()
	get_planet_data()
	get_species_data()

	/* Main loop. For each species, take appropriate action. */
	for sp_index := 0; sp_index < num_species; sp_index++ {
		species_number = sp_num[sp_index]

		found := data_in_memory[species_number-1]
		if found == FALSE {
			if do_all_species != FALSE {
				if first_pass != FALSE {
					fmt.Printf("\n    Skipping species #%d.\n", species_number)
				}
				continue
			} else {
				fmt.Fprintf(os.Stderr, "\n    Cannot get data for species #%d!\n", species_number)
				os.Exit(255)
			}
		}

		species = &spec_data[species_number-1]
		nampla_base = namp_data[species_number-1]
		ship_base = ship_data[species_number-1]

		/* Open orders file for this species. */
		filename := fmt.Sprintf("sp%02d.ord", species_number)
		input_file = fopen_r(filename)
		if input_file == nil {
			if do_all_species != FALSE {
				if first_pass != FALSE {
					fmt.Printf("\n    No orders for species #%d.\n", species_number)
				}
				continue
			} else {
				fmt.Fprintf(os.Stderr, "\n\tCannot open '%s' for reading!\n\n", filename)
				os.Exit(255)
			}
		}

		/* Open log file. Use stdout for first pass. */
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
				os.Exit(255)
			}
		}

		end_of_file = FALSE

		just_opened_file = TRUE /* Tell command parser to skip mail header, if any. */
	find_start:

		/* Search for START JUMPS order. */
		found = FALSE
		for found == FALSE {
			command := get_command()
			if command == MESSAGE {
				/* Skip MESSAGE text.
				 * It may contain a line that starts with "start". */
				for {
					command = get_command()
					if command < 0 {
						fmt.Fprintf(os.Stderr, "WARNING: Unterminated MESSAGE command in file %s!\n", filename)
						break
					}
					if command == ZZZ {
						goto find_start
					}
				}
			}

			if command < 0 {
				/* End of file. */
				break
			}
			if command != START {
				continue
			}

			/* Get the first three letters of the keyword and convert to upper case. */
			skip_whitespace()
			var keyword [3]byte
			for i := 0; i < 3 && at(input_line, input_line_pointer) != 0; i++ {
				keyword[i] = toupper(at(input_line, input_line_pointer))
				input_line_pointer++
			}

			if string(keyword[:]) == "JUM" {
				found = TRUE
			}
		}

		if found == FALSE {
			if first_pass != FALSE {
				fmt.Printf("\nNo jump orders for species #%d, SP %s.\n", species_number, species.name)
			}
			goto done_orders
		}

		/* Handle jump orders for this species. */
		log_string("\nJump orders:\n")
		do_jump_orders()
		species_jumped[species_number-1] = TRUE
		data_modified[species_number-1] = TRUE

	done_orders:

		input_file.fclose()

		/* Take care of any ships that withdrew or were forced to jump during combat. */
		for ship_index = 0; ship_index < species.num_ships; ship_index++ {
			ship = ship_base[ship_index]
			if ship.status == FORCED_JUMP || ship.status == JUMPED_IN_COMBAT {
				do_JUMP_command(TRUE, FALSE)
			}
		}

		/* If this is the second pass, close the log file. */
		if first_pass == FALSE {
			log_file.Close()
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

		goto start_pass
	}

	/* no_jump_orders: */

	/* Take care of any ships that withdrew from combat but were not
	 * handled above because no jump orders were received for species. */
	log_stdout = TRUE
	{
		log_file_open := FALSE
		for species_number = 1; species_number <= galaxy.num_species; species_number++ {
			if species_jumped[species_number-1] != FALSE {
				continue
			}
			if data_in_memory[species_number-1] == FALSE {
				continue
			}

			species = &spec_data[species_number-1]
			nampla_base = namp_data[species_number-1]
			ship_base = ship_data[species_number-1]

			for ship_index = 0; ship_index < species.num_ships; ship_index++ {
				ship = ship_base[ship_index]
				if ship.status == FORCED_JUMP || ship.status == JUMPED_IN_COMBAT {
					if log_file_open == FALSE {
						filename := fmt.Sprintf("sp%02d.log", species_number)
						var err error
						log_file, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
						if err != nil {
							log_file = nil
							fmt.Fprintf(os.Stderr, "\n\tCannot open '%s' for appending!\n\n", filename)
							os.Exit(2)
						}
						log_file_open = TRUE
						log_string("\nWithdrawals and forced jumps during combat:\n")
					}

					do_JUMP_command(TRUE, FALSE)
				}
			}

			data_modified[species_number-1] = log_file_open

			if log_file_open != FALSE {
				log_file.Close()
				log_file_open = FALSE
			}
		}
	}

	save_species_data()
	save_transaction_data()
	if star_data_modified != FALSE {
		save_star_data()
	}
	if planet_data_modified != FALSE {
		save_planet_data()
	}
	free_species_data()
	star_base = nil
	planet_base = nil

	return 0
}
