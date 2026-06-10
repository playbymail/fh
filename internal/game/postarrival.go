package game

// Port of postarrival.c.

import (
	"fmt"
	"os"
)

func do_postarrival_orders() {
	var command int

	if first_pass != FALSE {
		fmt.Printf("\nStart of post-arrival orders for species #%d, SP %s...\n", species_number, species.name)
	}

	/* For these commands, do not display age or landed/orbital status of ships. */
	truncate_name = TRUE

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
				fmt.Printf("End of post-arrival orders for species #%d, SP %s.\n", species_number, species.name)
			}
			if first_pass != FALSE {
				gamemaster_abort_option()
			}
			break /* END for this species. */
		}

		switch command {
		case ALLY:
			do_ALLY_command()
		case AUTO:
			species.auto_orders = TRUE
			log_string("    An AUTO order was executed.\n")
		case DEEP:
			do_DEEP_command()
		case DESTROY:
			do_DESTROY_command()
		case ENEMY:
			do_ENEMY_command()
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
			do_SCAN_command()
		case SEND:
			do_SEND_command()
		case TEACH:
			do_TEACH_command()
		case TECH:
			// do_TECH_command();
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! Invalid post-arrival command.\n")
		case TRANSFER:
			do_TRANSFER_command()
		case TELESCOPE:
			do_TELESCOPE_command()
		case TERRAFORM:
			do_TERRAFORM_command()
		default:
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! Invalid post-arrival command.\n")
		}
	}
}

func postArrivalCommand(args []string) int {
	var num_species, sp_index, command, do_all_species int
	var sp_num [MAX_SPECIES]int

	/* Get commonly used data. */
	get_galaxy_data()
	get_star_data()
	get_planet_data()
	get_transaction_data()

	ignore_field_distorters = TRUE

	/* Check arguments.
	 * If an argument is -p, then do two passes.
	 * In the first pass, display results and prompt the GM, allowing him
	 * to abort if necessary before saving results to disk.
	 * All other arguments must be species numbers.
	 * If no species numbers are specified, then do all species. */
	num_species = 0
	first_pass = FALSE
	test_mode = FALSE
	verbose_mode = FALSE
	for i := 1; i < len(args); i++ {
		if args[i] == "-p" {
			first_pass = TRUE
		} else if args[i] == "-t" {
			test_mode = TRUE
		} else if args[i] == "-v" {
			verbose_mode = TRUE
		} else {
			n := cfgAtoi(args[i])
			if n < 1 || n > galaxy.num_species {
				fmt.Fprintf(os.Stderr, "\n    '%s' is not a valid argument!\n", args[i])
				os.Exit(2)
			}
			sp_num[num_species] = n
			num_species++
		}
	}

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

start_pass:

	if first_pass != FALSE {
		fmt.Printf("\nStarting first pass...\n\n")
	}

	get_star_data()
	get_planet_data()
	get_species_data()

	/* Main loop. For each species, take appropriate action. */
	for sp_index = 0; sp_index < num_species; sp_index++ {
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
				fmt.Fprintf(os.Stderr, "\n    Cannot get data for species #%d!\n",
					species_number)
				os.Exit(255)
			}
		}

		species = &spec_data[species_index]
		nampla_base = namp_data[species_index]
		ship_base = ship_data[species_index]

		/* Do some initializations. */
		species.auto_orders = FALSE

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

		end_of_file = FALSE

		/* Tell command parser to skip mail header, if any. */
		just_opened_file = TRUE

	find_start:

		/* Search for START POST-ARRIVAL order. */
		found = FALSE
		for found == FALSE {
			command = get_command()
			if command == MESSAGE {
				/* Skip MESSAGE text. It may contain a line that starts with "start". */
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

			if string(keyword[:]) == "POS" {
				found = TRUE
			}
		}

		if found == FALSE {
			if first_pass != FALSE {
				fmt.Printf("\nNo post-arrival orders for species #%d, SP %s.\n", species_number, species.name)
			}
			goto done_orders
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
			log_string("\nPost-arrival orders:\n")
		}

		/* For each ship, set dest_z to zero.
		 * If a starbase is used as a gravitic telescope, it will be set to non-zero.
		 * This will prevent more than one TELESCOPE order per turn per starbase. */
		for i := 0; i < species.num_ships; i++ {
			ship = ship_base[i]
			ship.dest_z = 0
		}

		/* Handle post-arrival orders for this species. */
		do_postarrival_orders()

		data_modified[species_index] = TRUE

		/* If this is the second pass, close the log file. */
		if first_pass == FALSE {
			log_file.Close()
		}

	done_orders:

		input_file.fclose()
	}

	if first_pass != FALSE {
		fmt.Printf("\nFinal chance to abort safely!\n")
		gamemaster_abort_option()
		first_pass = FALSE
		free_species_data()
		/* In case data was modified. */
		if planet_base != nil {
			planet_base = nil
		}
		if star_base != nil {
			star_base = nil
		}

		fmt.Printf("\nStarting second pass...\n\n")

		goto start_pass
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
	planet_base = nil
	star_base = nil

	return 0
}
