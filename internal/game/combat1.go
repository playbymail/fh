package game

import (
	"fmt"
	"os"
)

// Port of combat.c, part 1 of 2 (lines 1-2028): the combat driver.
// Shared combat types, constants, and file-scope globals live in
// combat_types.go. Part 2 (combat2.go) ports do_bombardment,
// do_germ_warfare, do_round, do_siege, fighting_params,
// forced_jump_units_used, regenerate_shields, and withdrawal_check.

// fgets_stdin mimics C fgets(buf, n, stdin) using the shared stdin
// reader: it reads at most n-1 bytes, stopping after a newline.
func fgets_stdin(n int) string {
	var buf []byte
	for len(buf) < n-1 {
		c, err := stdin_reader.ReadByte()
		if err != nil {
			break
		}
		buf = append(buf, c)
		if c == '\n' {
			break
		}
	}
	return string(buf)
}

/* This routine will find all species that have declared alliance with both a traitor and betrayed species.
 * It will then set a flag to indicate that their allegiance should be changed from ALLY to ENEMY. */
func auto_enemy(traitor_species_number, betrayed_species_number int) {
	traitor_array_index := (traitor_species_number - 1) / 32
	traitor_bit_mask := uint32(1) << uint((traitor_species_number-1)%32)

	betrayed_array_index := (betrayed_species_number - 1) / 32
	betrayed_bit_mask := uint32(1) << uint((betrayed_species_number-1)%32)

	for species_index := 0; species_index < galaxy.num_species; species_index++ {
		if (spec_data[species_index].ally[traitor_array_index] & traitor_bit_mask) == 0 {
			continue
		}
		if (spec_data[species_index].ally[betrayed_array_index] & betrayed_bit_mask) == 0 {
			continue
		}
		if (spec_data[species_index].contact[traitor_array_index] & traitor_bit_mask) == 0 {
			continue
		}
		if (spec_data[species_index].contact[betrayed_array_index] & betrayed_bit_mask) == 0 {
			continue
		}
		make_enemy[species_index][traitor_species_number-1] = betrayed_species_number
	}
}

func bad_argument() {
	fmt.Fprintf(log_file, "!!! Order ignored:\n")
	fmt.Fprintf(log_file, "!!! %s", input_line)
	fmt.Fprintf(log_file, "!!! Invalid argument in command.\n")
}

func bad_coordinates() {
	fmt.Fprintf(log_file, "!!! Order ignored:\n")
	fmt.Fprintf(log_file, "!!! %s", input_line)
	fmt.Fprintf(log_file, "!!! Invalid coordinates in command.\n")
}

func bad_species() {
	fmt.Fprintf(log_file, "!!! Order ignored:\n")
	fmt.Fprintf(log_file, "!!! %s", input_line)
	fmt.Fprintf(log_file, "!!! Invalid species name!\n")
}

func battle_error(species_number int) {
	fmt.Fprintf(log_file, "!!! Order ignored:\n")
	fmt.Fprintf(log_file, "!!! %s", input_line)
	fmt.Fprintf(log_file, "!!! Missing BATTLE command!\n")
}

// combat returns TRUE if planet, species, and transaction data should be saved
func combat(default_summary, do_all_species, num_species int, sp_num []int, sp_name []string, locations_base []sp_loc_data_t) int {
	var save int = TRUE
	var i int
	var j int
	var k int
	var found int
	var command int
	var species_number int
	var sp_index int
	var num_battles int
	var location_index int
	var num_enemies int
	var battle_index int
	var option_index int
	var arg_index int
	var at_index int
	var really_hidden int
	var num_pls int
	var pl_num [9]int
	var enemy_word_number int
	var enemy_bit_number int
	var log_open int
	var distorted_name int
	var best_score int
	var next_best_score int
	var best_species_index int
	var betrayed_species_number int
	var name_length int
	var minimum_score int
	var n int
	var enemy_mask uint32
	var x int
	var y int
	var z int
	var option int
	var filename string
	var kw [3]byte
	var keyword string
	var answer string
	var log_line string
	var ok bool
	var err error
	var temp_ptr int
	var temp_species_log *cfile
	var species_log *os.File
	var sp *species_data_t
	var namp *nampla_data_t
	var sh *ship_data_t
	var bat *battle_data_t
	var location *sp_loc_data_t

	/* Main loop. For each species, take appropriate action. */
	num_battles = 0
	for arg_index = 0; arg_index < num_species; arg_index++ {
		species_number = sp_num[arg_index]
		if data_in_memory[species_number-1] == FALSE {
			continue
		}

		sp = &spec_data[species_number-1]

		/* The following two items are needed by get_ship(). */
		species = sp
		ship_base = ship_data[species_number-1]

		/* Open orders file for this species. */
		filename = fmt.Sprintf("sp%02d.ord", species_number)
		input_file = fopen_r(filename)
		if input_file == nil {
			if do_all_species != FALSE {
				if prompt_gm != FALSE {
					fmt.Printf("\nNo orders for species #%d, SP %s.\n", species_number, sp.name)
				}
				continue
			} else {
				fmt.Fprintf(os.Stderr, "\n\tCannot open '%s' for reading!\n\n", filename)
				os.Exit(255)
			}
		}

		end_of_file = FALSE

		just_opened_file = TRUE /* Tell command parser to skip mail header, if any. */
	find_start:

		/* Search for START COMBAT order. */
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

			for i = 0; i < 3; i++ {
				kw[i] = toupper(at(input_line, input_line_pointer))
				input_line_pointer++
			}
			keyword = string(kw[:])

			if strike_phase != FALSE {
				if keyword == "STR" {
					found = TRUE
				}
			} else {
				if keyword == "COM" {
					found = TRUE
				}
			}
		}

		if found != FALSE {
			if prompt_gm != FALSE {
				if strike_phase != FALSE {
					fmt.Printf("\nStrike orders for species #%d, SP %s...\n", species_number, sp.name)
				} else {
					fmt.Printf("\nCombat orders for species #%d, SP %s...\n", species_number, sp.name)
				}
			}
		} else {
			if prompt_gm != FALSE {
				if strike_phase != FALSE {
					fmt.Printf("\nNo strike orders for species #%d, SP %s...\n", species_number, sp.name)
				} else {
					fmt.Printf("\nNo combat orders for species #%d, SP %s...\n", species_number, sp.name)
				}
			}
			goto done_orders
		}

		/* Open temporary log file for appending. */
		filename = fmt.Sprintf("sp%02d.temp.log", species_number)
		log_file, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n\tCannot open '%s' for appending!\n\n", filename)
			os.Exit(255)
		}

		append_log[species_number-1] = TRUE

		log_stdout = FALSE
		if strike_phase != FALSE {
			log_string("\nStrike orders:\n")
		} else {
			log_string("\nCombat orders:\n")
		}
		log_stdout = prompt_gm

		/* Parse all combat commands for this species and save results for later use. */
		battle_index = -1
		for {
			command = get_command()
			if end_of_file != FALSE {
				break
			}

			if command == END {
				break
			}

			if command == BATTLE {
				num_enemies = 0 /* No enemies specified yet. */

				if get_value() == 0 {
					bad_coordinates()
					continue
				}
				x = value

				if get_value() == 0 {
					bad_coordinates()
					continue
				}
				y = value

				if get_value() == 0 {
					bad_coordinates()
					continue
				}
				z = value

				/* Make sure that species is present at battle location. */
				found = FALSE
				for i = 0; i < num_locs; i++ {
					location = &locations_base[i]
					if location.s != species_number {
						continue
					}
					if location.x != x {
						continue
					}
					if location.y != y {
						continue
					}
					if location.z != z {
						continue
					}

					found = TRUE
					break
				}
				if found == FALSE {
					fmt.Fprintf(log_file, "!!! Order ignored:\n")
					fmt.Fprintf(log_file, "!!! %s", input_line)
					fmt.Fprintf(log_file, "!!! Your species is not at this location!\n")
					continue
				}

				log_string("  A battle order was issued for sector ")
				log_int(x)
				log_char(' ')
				log_int(y)
				log_char(' ')
				log_int(z)
				log_string(".\n")

				/* Add coordinates to list if not already there. */
				found = FALSE
				for i = 0; i < num_battles; i++ {
					bat = battle_base[i]
					if x == bat.x && y == bat.y && z == bat.z {
						found = TRUE
						battle_index = i
						break
					}
				}

				if found == FALSE {
					/* This is a new battle location. */
					if num_battles == MAX_BATTLES {
						fmt.Fprintf(os.Stderr, "\n\n\tMAX_BATTLES exceeded! Edit file 'combat.h' and re-compile!\n\n")
						os.Exit(255)
					}
					bat = battle_base[num_battles]
					battle_index = num_battles
					sp_index = 0
					bat.x = x
					bat.y = y
					bat.z = z
					bat.spec_num[0] = species_number
					bat.special_target[0] = 0              /* Default. */
					bat.transport_withdraw_age[0] = 0      /* Default. */
					bat.warship_withdraw_age[0] = 100      /* Default. */
					bat.fleet_withdraw_percentage[0] = 100 /* Default. */
					bat.haven_x[0] = 127
					/* 127 means not yet specified. */
					bat.engage_option[sp_index][0] = DEFENSE_IN_PLACE
					bat.num_engage_options[0] = 1
					bat.can_be_surprised[0] = FALSE
					bat.hijacker[0] = FALSE
					bat.summary_only[0] = default_summary
					bat.num_species_here = 1
					for i = 0; i < MAX_SPECIES; i++ {
						bat.enemy_mine[0][i] = 0
					}
					num_battles++
				} else {
					/* Add another species to existing battle location. */
					sp_index = bat.num_species_here
					bat.spec_num[sp_index] = species_number
					bat.special_target[sp_index] = 0              /* Default. */
					bat.transport_withdraw_age[sp_index] = 0      /* Default. */
					bat.warship_withdraw_age[sp_index] = 100      /* Default. */
					bat.fleet_withdraw_percentage[sp_index] = 100 /* Default. */
					bat.haven_x[sp_index] = 127
					/* 127 means not yet specified. */
					bat.engage_option[sp_index][0] = DEFENSE_IN_PLACE
					bat.num_engage_options[sp_index] = 1
					bat.can_be_surprised[sp_index] = FALSE
					bat.hijacker[sp_index] = FALSE
					bat.summary_only[sp_index] = default_summary
					bat.num_species_here++
					for i = 0; i < MAX_SPECIES; i++ {
						bat.enemy_mine[sp_index][i] = 0
					}
				}
				continue
			}

			if command == SUMMARY {
				if battle_index < 0 {
					battle_error(species_number)
					continue
				}

				bat.summary_only[sp_index] = TRUE

				log_string("    Summary mode was specified.\n")

				continue
			}

			if command == WITHDRAW {
				if battle_index < 0 {
					battle_error(species_number)
					continue
				}

				if get_value() == 0 || value < 0 || value > 100 {
					bad_argument()
					continue
				}
				i = value
				bat.transport_withdraw_age[sp_index] = i

				if get_value() == 0 || value < 0 || value > 100 {
					bad_argument()
					continue
				}
				j = value
				bat.warship_withdraw_age[sp_index] = j

				if get_value() == 0 || value < 0 || value > 100 {
					bad_argument()
					continue
				}
				k = value
				bat.fleet_withdraw_percentage[sp_index] = k

				log_string("    Withdrawal conditions were set to ")
				log_int(i)
				log_char(' ')
				log_int(j)
				log_char(' ')
				log_int(k)
				log_string(".\n")

				continue
			}

			if command == HAVEN {
				if battle_index < 0 {
					battle_error(species_number)
					continue
				}

				if get_value() == 0 {
					bad_coordinates()
					continue
				}
				i = value
				bat.haven_x[sp_index] = value

				if get_value() == 0 {
					bad_coordinates()
					continue
				}
				j = value
				bat.haven_y[sp_index] = value

				if get_value() == 0 {
					bad_coordinates()
					continue
				}
				k = value
				bat.haven_z[sp_index] = value

				log_string("    Haven location set to sector ")
				log_int(i)
				log_char(' ')
				log_int(j)
				log_char(' ')
				log_int(k)
				log_string(".\n")

				continue
			}

			if command == ENGAGE {
				if battle_index < 0 {
					battle_error(species_number)
					continue
				}

				option_index = bat.num_engage_options[sp_index]
				if option_index >= MAX_ENGAGE_OPTIONS {
					fmt.Fprintf(log_file, "!!! Order ignored:\n")
					fmt.Fprintf(log_file, "!!! %s", input_line)
					fmt.Fprintf(log_file, "!!! Too many ENGAGE orders!\n")
					continue
				}

				if get_value() == 0 || value < 0 || value > 7 {
					fmt.Fprintf(log_file, "!!! Order ignored:\n")
					fmt.Fprintf(log_file, "!!! %s", input_line)
					fmt.Fprintf(log_file, "!!! Invalid ENGAGE option!\n")
					continue
				}
				option = value

				if strike_phase != FALSE && (option > 4) {
					fmt.Fprintf(log_file, "!!! Order ignored:\n")
					fmt.Fprintf(log_file, "!!! %s", input_line)
					fmt.Fprintf(log_file, "!!! Invalid ENGAGE option for strike phase!\n")
					continue
				}

				bat.engage_option[sp_index][option_index] = option

				/* Get planet to attack/defend, if any. */
				if option == PLANET_DEFENSE || (option >= PLANET_ATTACK && option <= SIEGE) {
					if get_value() == 0 {
						fmt.Fprintf(log_file, "!!! Order ignored:\n")
						fmt.Fprintf(log_file, "!!! %s", input_line)
						fmt.Fprintf(log_file, "!!! Missing planet argument in ENGAGE order!\n")
						continue
					}

					if value < 1 || value > 9 {
						fmt.Fprintf(log_file, "!!! Order ignored:\n")
						fmt.Fprintf(log_file, "!!! %s", input_line)
						fmt.Fprintf(log_file, "!!! Invalid planet argument in ENGAGE order!\n")
						continue
					}

					bat.engage_planet[sp_index][option_index] = value
				} else {
					value = 0
					bat.engage_planet[sp_index][option_index] = 0
				}

				bat.num_engage_options[sp_index]++

				log_string("    Engagement order ")
				log_int(option)
				if value != 0 {
					log_char(' ')
					log_long(value)
				}
				log_string(" was specified.\n")

				continue
			}

			if command == HIDE {
				if battle_index < 0 {
					battle_error(species_number)
					continue
				}
				if get_ship() == FALSE {
					fmt.Fprintf(log_file, "!!! Order ignored:\n")
					fmt.Fprintf(log_file, "!!! %s", input_line)
					fmt.Fprintf(log_file, "!!! Invalid or missing ship name!\n")
					continue
				}
				if ship.status != ON_SURFACE {
					fmt.Fprintf(log_file, "!!! Order ignored:\n")
					fmt.Fprintf(log_file, "!!! %s", input_line)
					fmt.Fprintf(log_file, "!!! Ship must be landed to HIDE!\n")
					continue
				}
				ship.special = NON_COMBATANT
				log_string("    ")
				log_string(ship_name(ship))
				log_string(" will attempt to stay out of the battle.\n")
				continue
			}

			if command == TARGET {
				if battle_index < 0 {
					battle_error(species_number)
					continue
				}

				if get_value() == 0 || value < 1 || value > 4 {
					fmt.Fprintf(log_file, "!!! Order ignored:\n")
					fmt.Fprintf(log_file, "!!! %s", input_line)
					fmt.Fprintf(log_file, "!!! Invalid TARGET option!\n")
					continue
				}
				bat.special_target[sp_index] = value

				log_string("    Strategic target ")
				log_long(value)
				log_string(" was specified.\n")

				continue
			}

			if command == ATTACK || command == HIJACK {
				if battle_index < 0 {
					battle_error(species_number)
					continue
				}

				if command == HIJACK {
					bat.hijacker[sp_index] = TRUE
				}

				/* Check if this is an order to attack all declared enemies. */
				if get_value() != 0 && value == 0 {
					for i = 0; i < galaxy.num_species; i++ {
						if species_number == i+1 {
							continue
						}
						if data_in_memory[i] == FALSE {
							continue
						}

						enemy_word_number = i / 32
						enemy_bit_number = i % 32
						enemy_mask = uint32(1) << uint(enemy_bit_number)

						if sp.enemy[enemy_word_number]&enemy_mask != 0 {
							if num_enemies == MAX_SPECIES {
								fmt.Fprintf(os.Stderr, "\n\n\tToo many enemies to ATTACK or HIJACK!\n\n")
								os.Exit(255)
							}
							if command == HIJACK {
								bat.enemy_mine[sp_index][num_enemies] = -(i + 1)
							} else {
								bat.enemy_mine[sp_index][num_enemies] = i + 1
							}
							num_enemies++
						}
					}
					if command == HIJACK {
						log_string("    An order was given to hijack all declared enemies.\n")
					} else {
						log_string("    An order was given to attack all declared enemies.\n")
					}
					continue
				}

				if num_enemies == MAX_SPECIES {
					fmt.Fprintf(os.Stderr, "\n\n\tToo many enemies to ATTACK or HIJACK!\n\n")
					os.Exit(255)
				}

				/* Set 'n' to the species number of the named enemy. */
				temp_ptr = input_line_pointer
				if get_class_abbr() != SPECIES_ID {
					/* Check if SP abbreviation was accidentally omitted. */
					if isdigit(at(input_line, temp_ptr)) {
						input_line_pointer = temp_ptr
					} else if at(input_line, input_line_pointer) != ' ' && at(input_line, input_line_pointer) != '\t' {
						input_line_pointer = temp_ptr
					}
				}

				distorted_name = FALSE
				if get_value() != 0 && !isalpha(at(input_line, input_line_pointer)) {
					n = undistorted(value)
					if n != 0 {
						distorted_name = TRUE
						goto att1
					}
				}
				if get_name() < 5 {
					bad_species()
					continue
				}

				/* Check for spelling error. */
				best_score = -9999
				next_best_score = -9999
				for i = 0; i < galaxy.num_species; i++ {
					if len(sp_name[i]) == 0 {
						continue
					}
					n = agrep_score(sp_name[i], upper_name)
					if n > best_score {
						best_score = n
						best_species_index = i
					} else if n > next_best_score {
						next_best_score = n
					}
				}

				name_length = len(sp_name[best_species_index])
				minimum_score = name_length - ((name_length / 7) + 1)

				if best_score < minimum_score || best_score == next_best_score {
					/* Score too low or another name with equal score. */
					bad_species()
					continue
				}

				n = best_species_index + 1

			att1:

				/* Make sure the named species is at the battle location. */
				found = FALSE
				for i = 0; i < num_locs; i++ {
					location = &locations_base[i]
					if location.s != n {
						continue
					}
					if location.x != bat.x {
						continue
					}
					if location.y != bat.y {
						continue
					}
					if location.z != bat.z {
						continue
					}

					found = TRUE
					break
				}

				/* Save species number temporarily in enemy_mine array. */
				if found != FALSE {
					if command == HIJACK {
						bat.enemy_mine[sp_index][num_enemies] = -n
					} else {
						bat.enemy_mine[sp_index][num_enemies] = n
					}
					num_enemies++
				}

				if command == HIJACK {
					log_string("    An order was given to hijack SP ")
				} else {
					log_string("    An order was given to attack SP ")
				}

				if distorted_name != FALSE {
					log_int(distorted(n))
				} else {
					log_string(spec_data[n-1].name)
				}
				log_string(".\n")

				continue
			}

			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! Invalid combat command.\n")
		}

		log_file.Close()

	done_orders:

		input_file.fclose()
	}

	/* Check each battle.  If a species specified a BATTLE command but did not specify any engage options, then add a DEFENSE_IN_PLACE option. */
	for battle_index = 0; battle_index < num_battles; battle_index++ {
		bat = battle_base[battle_index]
		for i = 0; i < bat.num_species_here; i++ {
			if bat.num_engage_options[i] == 0 {
				bat.num_engage_options[i] = 1
				bat.engage_option[i][0] = DEFENSE_IN_PLACE
			}
		}
	}

	/* Initialize make_enemy array. */
	for i = 0; i < galaxy.num_species; i++ {
		for j = 0; j < galaxy.num_species; j++ {
			make_enemy[i][j] = 0
		}
	}

	/* Check each battle location. If a species is at the location
	   but has no combat orders, add it to the list of species at that
	   battle, and apply defaults. After all species are accounted for
	   at the current battle location, do battle. */
	for battle_index = 0; battle_index < num_battles; battle_index++ {
		bat = battle_base[battle_index]

		x = bat.x
		y = bat.y
		z = bat.z

		/* Check file 'locations.dat' for other species at this location. */
		for location_index = 0; location_index < num_locs; location_index++ {
			location = &locations_base[location_index]
			if location.x != x {
				continue
			}
			if location.y != y {
				continue
			}
			if location.z != z {
				continue
			}

			/* Check if species is already accounted for. */
			found = FALSE
			species_number = location.s
			for sp_index = 0; sp_index < bat.num_species_here; sp_index++ {
				if bat.spec_num[sp_index] == species_number {
					found = TRUE
					break
				}
			}

			if found != FALSE {
				continue
			}

			/* Species is present but did not give any combat orders.
			   This species will be included in the battle ONLY if it has
			   ships in deep space or in orbit or if it has an unhidden,
			   populated planet in this sector or if it has a hidden
			   planet that is being explicitly attacked. */
			found = FALSE

			sp = &spec_data[species_number-1]

			num_pls = 0

			for i = 0; i < sp.num_namplas; i++ {
				namp = namp_data[species_number-1][i]

				if namp.pn == 99 {
					continue
				}
				if namp.x != x {
					continue
				}
				if namp.y != y {
					continue
				}
				if namp.z != z {
					continue
				}
				if (namp.status & POPULATED) == 0 {
					continue
				}

				really_hidden = FALSE
				if namp.hidden != FALSE {
					/* If this species and planet is explicitly mentioned in ATTACK/ENGAGE orders, then the planet cannot hide during the battle. */
					really_hidden = TRUE

					for at_index = 0; at_index < bat.num_species_here; at_index++ {
						for j = 0; j < MAX_SPECIES; j++ {
							k = bat.enemy_mine[at_index][j]
							if k < 0 {
								k = -k
							}
							if k == species_number {
								for k = 0; k < bat.num_engage_options[at_index]; k++ {
									if bat.engage_option[at_index][k] >= PLANET_ATTACK &&
										bat.engage_option[at_index][k] <= SIEGE &&
										bat.engage_planet[at_index][k] == namp.pn {
										really_hidden = FALSE
										break
									}
								}
								if really_hidden == FALSE {
									break
								}
							}
						}
						if really_hidden == FALSE {
							break
						}
					}
				}

				if really_hidden != FALSE {
					continue
				}

				found = TRUE
				pl_num[num_pls] = namp.pn
				num_pls++
			}

			for i = 0; i < sp.num_ships; i++ {
				sh = ship_data[species_number-1][i]

				if sh.pn == 99 {
					continue
				}
				if sh.x != x {
					continue
				}
				if sh.y != y {
					continue
				}
				if sh.z != z {
					continue
				}
				if sh.status == UNDER_CONSTRUCTION {
					continue
				}
				if sh.status == ON_SURFACE {
					continue
				}
				if sh.status == JUMPED_IN_COMBAT {
					continue
				}
				if sh.status == FORCED_JUMP {
					continue
				}
				found = TRUE

				break
			}

			if found == FALSE {
				continue
			}

			sp_index = bat.num_species_here
			bat.spec_num[sp_index] = location.s
			bat.special_target[sp_index] = 0
			bat.transport_withdraw_age[sp_index] = 0
			bat.warship_withdraw_age[sp_index] = 100
			bat.fleet_withdraw_percentage[sp_index] = 100
			bat.haven_x[sp_index] = 127
			bat.engage_option[sp_index][0] = DEFENSE_IN_PLACE
			bat.num_engage_options[sp_index] = 1
			if num_pls > 0 {
				/* Provide default Engage 2 options. */
				for i = 0; i < num_pls; i++ {
					bat.engage_option[sp_index][i+1] = PLANET_DEFENSE
					bat.engage_planet[sp_index][i+1] = pl_num[i]
				}
				bat.num_engage_options[sp_index] = num_pls + 1
			}
			bat.can_be_surprised[sp_index] = TRUE
			bat.hijacker[sp_index] = FALSE
			bat.summary_only[sp_index] = default_summary
			for i = 0; i < MAX_SPECIES; i++ {
				bat.enemy_mine[sp_index][i] = 0
			}
			bat.num_species_here++
		}

		/* If haven locations have not been specified, provide random locations nearby. */
		for sp_index = 0; sp_index < bat.num_species_here; sp_index++ {
			if bat.haven_x[sp_index] != 127 {
				continue
			}

			for {
				i = x + 2 - rnd(3)
				j = y + 2 - rnd(3)
				k = z + 2 - rnd(3)

				if i != x || j != y || k != z {
					break
				}
			}

			bat.haven_x[sp_index] = i
			bat.haven_y[sp_index] = j
			bat.haven_z[sp_index] = k
		}

		/* Do battle at this battle location. */
		do_battle(bat)

		if prompt_gm != FALSE {
			fmt.Printf("Hit RETURN to continue...")

			os.Stdout.Sync()
			fgets_stdin(16)
		}
	}

	/* Declare new enmities. */
	for i = 0; i < galaxy.num_species; i++ {
		log_open = FALSE

		for j = 0; j < galaxy.num_species; j++ {
			if i == j {
				continue
			}

			betrayed_species_number = make_enemy[i][j]
			if betrayed_species_number == 0 {
				continue
			}

			enemy_word_number = j / 32
			enemy_bit_number = j % 32
			enemy_mask = uint32(1) << uint(enemy_bit_number)

			/* Clear ally bit. */
			spec_data[i].ally[enemy_word_number] &^= enemy_mask

			/* Set enemy and contact bits (in case this is first encounter). */
			spec_data[i].enemy[enemy_word_number] |= enemy_mask
			spec_data[i].contact[enemy_word_number] |= enemy_mask

			data_modified[i] = TRUE

			if log_open == FALSE {
				/* Open temporary species log file for appending. */
				filename = fmt.Sprintf("sp%02d.temp.log", i+1)
				log_file, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
				if err != nil {
					fmt.Fprintf(os.Stderr, "\n\tCannot open '%s' for appending!\n\n", filename)
					os.Exit(255)
				}

				append_log[i] = TRUE
				log_open = TRUE
			}

			log_string("\n!!! WARNING: Enmity has been automatically declared towards SP ")
			log_string(spec_data[j].name)
			log_string(" because they surprise-attacked SP ")
			log_string(spec_data[betrayed_species_number-1].name)
			log_string("!\n")
		}

		if log_open != FALSE {
			log_file.Close()
		}
	}

	if prompt_gm != FALSE {
		fmt.Printf("\n*** Gamemaster safe-abort option ... type q or Q to quit: ")

		os.Stdout.Sync()
		answer = fgets_stdin(16)
		if at(answer, 0) == 'q' || at(answer, 0) == 'Q' {
			save = FALSE
		}
	}

	/* If results are to be saved, append temporary logs to actual species logs. In either case, delete temporary logs. */
	for i = 0; i < galaxy.num_species; i++ {
		if append_log[i] == FALSE {
			continue
		}

		if save != FALSE {
			filename = fmt.Sprintf("sp%02d.log", i+1)
			species_log, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\n\tCannot open '%s' for appending!\n\n", filename)
				os.Exit(255)
			}
		}

		filename = fmt.Sprintf("sp%02d.temp.log", i+1)

		if save != FALSE {
			temp_species_log = fopen_r(filename)
			if temp_species_log == nil {
				fmt.Fprintf(os.Stderr, "\n\tCannot open '%s' for reading!\n\n", filename)
				os.Exit(255)
			}

			/* Copy temporary log to permanent species log. */
			for {
				log_line, ok = readln(temp_species_log, 256)
				if !ok {
					break
				}
				species_log.WriteString(log_line)
			}

			temp_species_log.fclose()
			species_log.Close()
		}

		/* Delete temporary log file. */
		os.Remove(filename)
	}

	return save
}

func combatCommand(args []string) int {
	default_summary := FALSE
	do_all_species := TRUE
	num_species := 0
	var sp *species_data_t
	sp_name := make([]string, MAX_SPECIES)

	var sp_num [MAX_SPECIES]int

	prompt_gm = FALSE
	strike_phase = FALSE // assume combat mode
	if args[0] == "strike" {
		strike_phase = TRUE
	}

	/* Get commonly used data. */
	get_galaxy_data()
	get_planet_data()
	get_transaction_data()
	get_location_data()

	/* Allocate memory for battle data. */
	battle_base = make([]*battle_data_t, MAX_BATTLES)
	for i := 0; i < MAX_BATTLES; i++ {
		battle_base[i] = &battle_data_t{}
	}

	/* Check arguments.
	 * If an argument is -s, then set SUMMARY mode for everyone.
	 * The default is for players to receive a detailed report of the battles.
	 * If an argument is -p, then prompt the GM before saving results;
	 * otherwise, operate quietly; i.e, do not prompt GM before saving results
	 * and do not display anything except errors.
	 * Any additional arguments must be species numbers.
	 * If no species numbers are specified, then do all species. */
	for i := 1; i < len(args); i++ {
		if args[i] == "-s" {
			default_summary = TRUE
		} else if args[i] == "-p" {
			prompt_gm = TRUE
		} else if args[i] == "-t" {
			test_mode = TRUE
		} else if args[i] == "-v" {
			verbose_mode = TRUE
			fmt.Printf(" info: combat: last_random is %12d\n", prngGetSeed())
		} else if args[i] == "--combat" {
			strike_phase = FALSE
		} else if args[i] == "--strike" {
			strike_phase = TRUE
		} else {
			n := cfgAtoi(args[i])
			if 0 < n && n <= galaxy.num_species {
				sp_num[num_species] = n
				num_species++
			}
		}
	}

	if strike_phase != FALSE {
		fmt.Printf(" info: combat: running %s mode\n", "strike")
	} else {
		fmt.Printf(" info: combat: running %s mode\n", "combat")
	}

	log_stdout = prompt_gm

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

	if default_summary != FALSE && prompt_gm != FALSE {
		fmt.Printf("\nSUMMARY mode is in effect for all species.\n\n")
	}

	/* Read in species data and make an uppercase copy of each name for comparison purposes later. Also do some initializations. */
	get_species_data()

	for sp_index := 0; sp_index < galaxy.num_species; sp_index++ {
		sp_name[sp_index] = ""
		if data_in_memory[sp_index] == FALSE {
			/* No longer in game. */
			continue
		}
		sp = &spec_data[sp_index]
		ship_base = ship_data[sp_index]
		/* Convert name to upper case. */
		sp_name[sp_index] = upcase(sp.name)
		for i := 0; i < sp.num_ships; i++ {
			ship = ship_base[i]
			ship.special = 0
		}
	}

	save := combat(default_summary, do_all_species, num_species, sp_num[:], sp_name, loc[:])
	if save != FALSE {
		save_planet_data()
		save_species_data()
		save_transaction_data()
	}

	free_species_data()

	return 0
}

func consolidate_option(option, location int) {
	/* Only attack options go in list. */
	if option < DEEP_SPACE_FIGHT {
		return
	}
	/* Make sure pre-requisites are already in the list. Bombardment, and germ warfare must follow a successful planet attack. */
	if option > PLANET_ATTACK {
		consolidate_option(PLANET_ATTACK, location)
	}
	/* Check if option and location are already in list. */
	for i := 0; i < num_combat_options; i++ {
		if option == combat_option[i] && location == combat_location[i] {
			return
		}
	}
	/* Add new option to list. */
	combat_option[num_combat_options] = option
	combat_location[num_combat_options] = location
	num_combat_options++
}

func disbanded_species_ship(species_index int, sh *ship_data_t) int {
	for nampla_index := 0; nampla_index < c_species[species_index].num_namplas; nampla_index++ {
		nam := c_nampla[species_index][nampla_index]
		if nam.x != sh.x {
			continue
		}
		if nam.y != sh.y {
			continue
		}
		if nam.z != sh.z {
			continue
		}
		if nam.pn != sh.pn {
			continue
		}
		if (nam.status & DISBANDED_COLONY) == 0 {
			continue
		}
		if sh.ship_type != STARBASE && sh.status == IN_ORBIT {
			continue
		}
		/* This ship is either on the surface of a disbanded colony or is a starbase orbiting a disbanded colony. */
		return TRUE
	}
	return FALSE
}

func do_ambush(ambushing_species_index int, bat *battle_data_t) {
	var i, j, n, num_sp, ambushed_species_index, num_ships, age_increment int
	var species_number int
	var old_truncate_name int
	var friendly_tonnage, enemy_tonnage int
	var sh *ship_data_t

	/* Get total ambushing tonnage. */
	friendly_tonnage = 0
	num_ships = c_species[ambushing_species_index].num_ships
	for i = 0; i < num_ships; i++ {
		sh = c_ship[ambushing_species_index][i]
		if sh.pn == 99 {
			continue
		}
		if sh.x != bat.x {
			continue
		}
		if sh.y != bat.y {
			continue
		}
		if sh.z != bat.z {
			continue
		}
		if sh.class != TR && sh.class != BA {
			friendly_tonnage += sh.tonnage
		}
	}

	/* Determine which species are being ambushed and get total enemy tonnage. */
	num_sp = bat.num_species_here
	enemy_tonnage = 0
	for ambushed_species_index = 0; ambushed_species_index < num_sp; ambushed_species_index++ {
		if bat.enemy_mine[ambushing_species_index][ambushed_species_index] == FALSE {
			continue
		}

		/* This species is being ambushed.  Get total effective tonnage. */
		num_ships = c_species[ambushed_species_index].num_ships
		for i = 0; i < num_ships; i++ {
			sh = c_ship[ambushed_species_index][i]

			if sh.pn == 99 {
				continue
			}
			if sh.x != bat.x {
				continue
			}
			if sh.y != bat.y {
				continue
			}
			if sh.z != bat.z {
				continue
			}
			if sh.class == TR {
				enemy_tonnage += sh.tonnage
			} else {
				enemy_tonnage += 10 * sh.tonnage
			}
		}
	}

	/* Determine the amount of aging that will be added to each ambushed ship. */
	if enemy_tonnage == 0 {
		return
	}
	age_increment = (10 * bat.ambush_amount[ambushing_species_index]) / enemy_tonnage
	age_increment = (friendly_tonnage * age_increment) / enemy_tonnage
	ambush_took_place = TRUE

	if age_increment < 1 {
		log_string("\n    SP ")
		log_string(c_species[ambushing_species_index].name)
		log_string(" attempted an ambush, but the ambush was completely ineffective!\n")
		return
	}

	/* Age each ambushed ship. */
	for ambushed_species_index = 0; ambushed_species_index < num_sp; ambushed_species_index++ {
		if bat.enemy_mine[ambushing_species_index][ambushed_species_index] == FALSE {
			continue
		}
		log_string("\n    SP ")
		species_number = bat.spec_num[ambushed_species_index]
		if field_distorted[ambushed_species_index] != FALSE {
			log_int(distorted(species_number))
		} else {
			log_string(c_species[ambushed_species_index].name)
		}
		log_string(" was ambushed by SP ")
		log_string(c_species[ambushing_species_index].name)
		log_string("!\n")
		num_ships = c_species[ambushed_species_index].num_ships
		for i = 0; i < num_ships; i++ {
			sh = c_ship[ambushed_species_index][i]
			if sh.pn == 99 {
				continue
			}
			if sh.x != bat.x {
				continue
			}
			if sh.y != bat.y {
				continue
			}
			if sh.z != bat.z {
				continue
			}
			sh.age += age_increment
			if sh.arrived_via_wormhole != FALSE {
				sh.age += age_increment
			}

			if sh.age > 49 {
				old_truncate_name = truncate_name
				truncate_name = TRUE
				log_string("      ")
				log_string(ship_name(sh))
				if field_distorted[ambushed_species_index] != FALSE {
					log_string(" = ")
					log_string(c_species[ambushed_species_index].name)
					log_char(' ')
					n = sh.item_quantity[FD]
					sh.item_quantity[FD] = 0
					log_string(ship_name(sh))
					sh.item_quantity[FD] = n
				}
				n = 0
				for j = 0; j < MAX_ITEMS; j++ {
					if sh.item_quantity[j] > 0 {
						if n == 0 {
							log_string(" (cargo: ")
						} else {
							log_char(',')
						}
						n++
						log_int(sh.item_quantity[j])
						log_char(' ')
						log_string(item_abbr[j])
					}
				}
				if n > 0 {
					log_char(')')
				}
				log_string(" was destroyed in the ambush!\n")
				truncate_name = old_truncate_name
			}
		}
	}
}

func do_battle(bat *battle_data_t) {
	var i, j, k int
	var species_index int
	var species_number int
	var num_sp,
		max_rounds, round_number, battle_here, fight_here,
		unit_index, option_index, current_species, temp_status,
		temp_pn, num_namplas, array_index, bit_number, first_action,
		traitor_number, betrayed_number, betrayal, need_comma,
		TRUE_value, do_withdraw_check_first int
	var identifiable_units, unidentifiable_units [MAX_SPECIES]int
	var bit_mask uint32
	var where, option int
	var filename string
	var enemy int
	var enemy_num [MAX_SPECIES]int
	var log_line string
	var ok bool
	var err error
	var combat_log *cfile
	var species_log *os.File
	var act action_data_t
	var namp, attacked_nampla *nampla_data_t
	var sh *ship_data_t

	ambush_took_place = FALSE

	/* Open log file for writing. */
	log_file, err = os.Create("combat.log")
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n\tCannot open 'combat.log' for writing!\n\n")
		os.Exit(255)
	}

	/* Open summary file for writing. */
	summary_file, err = os.Create("summary.log")
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n\tCannot open 'summary.log' for writing!\n\n")
		os.Exit(255)
	}
	log_summary = TRUE

	/* Get data for all species present at this battle. */
	num_sp = bat.num_species_here
	for species_index = 0; species_index < num_sp; species_index++ {
		species_number = bat.spec_num[species_index]
		c_species[species_index] = &spec_data[species_number-1]
		c_nampla[species_index] = namp_data[species_number-1]
		c_ship[species_index] = ship_data[species_number-1]
		if data_in_memory[species_number-1] != FALSE {
			data_modified[species_number-1] = TRUE
		} else {
			fmt.Fprintf(os.Stderr, "\n\tData for species #%d is needed but is not available!\n\n",
				species_number)
			os.Exit(255)
		}

		/* Determine number of identifiable and unidentifiable units present. */
		identifiable_units[species_index] = 0
		unidentifiable_units[species_index] = 0

		for i = 0; i < c_species[species_index].num_namplas; i++ {
			namp = c_nampla[species_index][i]

			if namp.x != bat.x {
				continue
			}
			if namp.y != bat.y {
				continue
			}
			if namp.z != bat.z {
				continue
			}

			if namp.status&POPULATED != 0 {
				identifiable_units[species_index]++
			}
		}

		for i = 0; i < c_species[species_index].num_ships; i++ {
			sh = c_ship[species_index][i]

			if sh.x != bat.x {
				continue
			}
			if sh.y != bat.y {
				continue
			}
			if sh.z != bat.z {
				continue
			}
			if sh.status == UNDER_CONSTRUCTION {
				continue
			}
			if sh.status == JUMPED_IN_COMBAT {
				continue
			}
			if sh.status == FORCED_JUMP {
				continue
			}

			sh.dest_x = 0   /* Not yet exposed. */
			sh.dest_y = 100 /* Shields at 100%. */

			if sh.item_quantity[FD] == sh.tonnage {
				unidentifiable_units[species_index]++
			} else {
				identifiable_units[species_index]++
			}
		}

		if identifiable_units[species_index] > 0 || unidentifiable_units[species_index] == 0 {
			field_distorted[species_index] = FALSE
		} else {
			field_distorted[species_index] = TRUE
		}
	}

	/* Start log of what's happening. */
	if strike_phase != FALSE {
		log_string("\nStrike log:\n")
	} else {
		log_string("\nCombat log:\n")
	}
	first_battle = FALSE

	log_string("\n  Battle orders were received for sector ")
	log_int(bat.x)
	log_string(", ")
	log_int(bat.y)
	log_string(", ")
	log_int(bat.z)
	log_string(". The following species are present:\n\n")

	/* Convert enemy_mine array from a list of species numbers to an array
	   of TRUE/FALSE values whose indices are:

	        [species_index1][species_index2]

	   such that the value will be TRUE if #1 mentioned #2 in an ATTACK
	   or HIJACK command.  The actual TRUE value will be 1 for ATTACK or
	   2 for HIJACK. */

	for species_index = 0; species_index < num_sp; species_index++ {
		/* Make copy of list of enemies. */
		for i = 0; i < MAX_SPECIES; i++ {
			enemy_num[i] = bat.enemy_mine[species_index][i]
			bat.enemy_mine[species_index][i] = FALSE
		}

		for i = 0; i < MAX_SPECIES; i++ {
			enemy = enemy_num[i]
			if enemy == 0 {
				break
			} /* No more enemies in list. */

			if enemy < 0 {
				enemy = -enemy
				TRUE_value = 2 /* This is a hijacking. */
			} else {
				TRUE_value = 1
			} /* This is a normal attack. */

			/* Convert absolute species numbers to species indices that
			   have been assigned in the current battle. */
			for j = 0; j < num_sp; j++ {
				if enemy == bat.spec_num[j] {
					bat.enemy_mine[species_index][j] = TRUE_value
				}
			}
		}
	}

	/* For each species that has been mentioned in an attack order, check
	   if it can be surprised. A species can only be surprised if it has
	   not given a BATTLE order and if it is being attacked ONLY by one
	   or more ALLIES. */
	for species_index = 0; species_index < num_sp; species_index++ {
		j = bat.spec_num[species_index] - 1
		array_index = j / 32
		bit_number = j % 32
		bit_mask = uint32(1) << uint(bit_number)

		for i = 0; i < num_sp; i++ {
			if i == species_index {
				continue
			}

			if bat.enemy_mine[species_index][i] == FALSE {
				continue
			}

			if field_distorted[species_index] != FALSE {
				/* Attacker is field-distorted. Surprise not possible. */
				bat.can_be_surprised[i] = FALSE
				continue
			}

			if c_species[i].ally[array_index]&bit_mask != 0 {
				betrayal = TRUE
			} else {
				betrayal = FALSE
			}

			if betrayal != FALSE {
				/* Someone is being attacked by an ALLY. */
				traitor_number = bat.spec_num[species_index]
				betrayed_number = bat.spec_num[i]
				make_enemy[betrayed_number-1][traitor_number-1] = betrayed_number
				auto_enemy(traitor_number, betrayed_number)
			}

			if bat.can_be_surprised[i] == FALSE {
				continue
			}

			if betrayal == FALSE { /* At least one attacker is not an ally. */
				bat.can_be_surprised[i] = FALSE
			}
		}
	}

	/* For each species that has been mentioned in an attack order, see if
	   there are other species present that have declared it as an ALLY.
	   If so, have the attacker attack the other species and vice-versa. */
	for species_index = 0; species_index < num_sp; species_index++ {
		for i = 0; i < num_sp; i++ {
			if i == species_index {
				continue
			}

			if bat.enemy_mine[species_index][i] == FALSE {
				continue
			}

			j = bat.spec_num[i] - 1
			array_index = j / 32
			bit_number = j % 32
			bit_mask = uint32(1) << uint(bit_number)

			for k = 0; k < num_sp; k++ {
				if k == species_index {
					continue
				}
				if k == i {
					continue
				}

				if c_species[k].ally[array_index]&bit_mask != 0 {
					/* Make sure it's not already set (it may already be set
					   for HIJACK and we don't want to accidentally change
					   it to ATTACK). */
					if bat.enemy_mine[species_index][k] == FALSE {
						bat.enemy_mine[species_index][k] = TRUE
					}
					if bat.enemy_mine[k][species_index] == FALSE {
						bat.enemy_mine[k][species_index] = TRUE
					}
				}
			}
		}
	}

	/* If a species did not give a battle order and is not the target of an
	   attack, set can_be_surprised flag to a special value. */
	for species_index = 0; species_index < num_sp; species_index++ {
		if bat.can_be_surprised[species_index] == FALSE {
			continue
		}

		bat.can_be_surprised[species_index] = 55

		for i = 0; i < num_sp; i++ {
			if i == species_index {
				continue
			}

			if bat.enemy_mine[i][species_index] == FALSE {
				continue
			}

			bat.can_be_surprised[species_index] = TRUE

			break
		}
	}

	/* List combatants. */
	for species_index = 0; species_index < num_sp; species_index++ {
		species_number = bat.spec_num[species_index]

		log_string("    SP ")
		if field_distorted[species_index] != FALSE {
			log_int(distorted(species_number))
		} else {
			log_string(c_species[species_index].name)
		}
		if bat.can_be_surprised[species_index] != FALSE {
			log_string(" does not appear to be ready for combat.\n")
		} else {
			log_string(" is mobilized and ready for combat.\n")
		}
	}

	/* Check if a declared enemy is being ambushed. */
	for i = 0; i < num_sp; i++ {
		num_namplas = c_species[i].num_namplas
		bat.ambush_amount[i] = 0
		for j = 0; j < num_namplas; j++ {
			namp = c_nampla[i][j]

			if namp.x != bat.x {
				continue
			}
			if namp.y != bat.y {
				continue
			}
			if namp.z != bat.z {
				continue
			}

			bat.ambush_amount[i] += namp.use_on_ambush
		}

		if bat.ambush_amount[i] == 0 {
			continue
		}

		for j = 0; j < num_sp; j++ {
			if bat.enemy_mine[i][j] != FALSE {
				do_ambush(i, bat)
			}
		}
	}

	/* For all species that specified enemies, make the feeling mutual. */
	for i = 0; i < num_sp; i++ {
		for j = 0; j < num_sp; j++ {
			if bat.enemy_mine[i][j] != FALSE {
				/* Make sure it's not already set (it may already be set for
				   HIJACK and we don't want to accidentally change it to
				   ATTACK). */
				if bat.enemy_mine[j][i] == FALSE {
					bat.enemy_mine[j][i] = TRUE
				}
			}
		}
	}

	/* Create a sequential list of combat options. First check if a
	   deep space defense has been ordered. If so, then make sure that
	   first option is DEEP_SPACE_FIGHT. */
	num_combat_options = 0
	for species_index = 0; species_index < num_sp; species_index++ {
		for i = 0; i < bat.num_engage_options[species_index]; i++ {
			option = bat.engage_option[species_index][i]
			if option == DEEP_SPACE_DEFENSE {
				consolidate_option(DEEP_SPACE_FIGHT, 0)
				goto consolidate
			}
		}
	}

consolidate:
	for species_index = 0; species_index < num_sp; species_index++ {
		for i = 0; i < bat.num_engage_options[species_index]; i++ {
			option = bat.engage_option[species_index][i]
			where = bat.engage_planet[species_index][i]
			consolidate_option(option, where)
		}
	}

	/* If ships are given unconditional withdraw orders, they will always have
	   time to escape if fighting occurs first in a different part of the
	   sector. The flag "do_withdraw_check_first" will be set only after the
	   first round of combat. */
	do_withdraw_check_first = FALSE

	/* Handle each combat option. */
	battle_here = FALSE
	first_action = TRUE
	for option_index = 0; option_index < num_combat_options; option_index++ {
		option = combat_option[option_index]
		where = combat_location[option_index]

		/* Fill action arrays with data about ships taking part in current action. */
		fight_here = fighting_params(option, where, bat, &act)
		/* Check if a fight will take place here. */
		if fight_here == FALSE {
			continue
		}
		/* See if anyone is taken by surprise. */
		if battle_here == FALSE {
			/* Combat is just starting. */
			for species_index = 0; species_index < num_sp; species_index++ {
				species_number = bat.spec_num[species_index]
				if bat.can_be_surprised[species_index] == 55 {
					continue
				}
				if bat.can_be_surprised[species_index] != FALSE {
					log_string("\n    SP ")
					if field_distorted[species_index] != FALSE {
						log_int(distorted(species_number))
					} else {
						log_string(c_species[species_index].name)
					}
					log_string(" is taken by surprise!\n")
				}
			}
		}

		battle_here = TRUE

		/* Clear out can_be_surprised array. */
		for i = 0; i < MAX_SPECIES; i++ {
			bat.can_be_surprised[i] = FALSE
		}

		/* Determine maximum number of rounds. */
		max_rounds = 10000 /* Something ridiculously large. */
		if option == DEEP_SPACE_FIGHT && attacking_ML > 0 && defending_ML > 0 && deep_space_defense != FALSE {
			/* This is the initial deep space fight and the defender wants the
			   fight to remain in deep space for as long as possible. */
			if defending_ML > attacking_ML {
				max_rounds = defending_ML - attacking_ML
			} else {
				max_rounds = 1
			}
		} else if option == PLANET_BOMBARDMENT {
			/* To determine the effectiveness of the bombardment, we will
			   simulate ten rounds of combat and add up the damage. */
			max_rounds = 10
		} else if option == GERM_WARFARE || option == SIEGE {
			/* We just need to see who is attacking whom and get the number
			   of germ warfare bombs being used. */
			max_rounds = 1
		}

		/* Log start of action. */
		if where == 0 {
			log_string("\n    The battle begins in deep space, outside the range of planetary defenses...\n")
		} else if option == PLANET_ATTACK {
			log_string("\n    The battle ")
			if first_action != FALSE {
				log_string("begins")
			} else {
				log_string("moves")
			}
			log_string(" within range of planet #")
			log_int(where)
			log_string("...\n")
		} else if option == PLANET_BOMBARDMENT {
			log_string("\n    Bombardment of planet #")
			log_int(where)
			log_string(" begins...\n")
		} else if option == GERM_WARFARE {
			log_string("\n    Germ warfare commences against planet #")
			log_int(where)
			log_string("...\n")
		} else if option == SIEGE {
			log_string("\n    Siege of planet #")
			log_int(where)
			log_string(" is now in effect...\n\n")
			goto do_combat
		}

		/* List combatants. */
		truncate_name = FALSE
		log_string("\n      Units present:")
		current_species = -1
		for unit_index = 0; unit_index < act.num_units_fighting; unit_index++ {
			if act.fighting_species_index[unit_index] != current_species {
				/* Display species name. */
				i = act.fighting_species_index[unit_index]
				log_string("\n        SP ")
				species_number = bat.spec_num[i]
				if field_distorted[i] != FALSE {
					log_int(distorted(species_number))
				} else {
					log_string(c_species[i].name)
				}
				log_string(": ")
				current_species = i
				need_comma = FALSE
			}

			if act.unit_type[unit_index] == SHIP {
				sh = act.fighting_unit_ship[unit_index]
				temp_status = sh.status
				temp_pn = sh.pn
				if option == DEEP_SPACE_FIGHT {
					sh.status = IN_DEEP_SPACE
					sh.pn = 0
				} else {
					sh.status = IN_ORBIT
					sh.pn = where
				}
				if field_distorted[current_species] == FALSE {
					ignore_field_distorters = TRUE
				} else {
					ignore_field_distorters = FALSE
				}
				if sh.special != NON_COMBATANT {
					if need_comma != FALSE {
						log_string(", ")
					}
					log_string(ship_name(sh))
					need_comma = TRUE
				}
				ignore_field_distorters = FALSE
				sh.status = temp_status
				sh.pn = temp_pn
			} else {
				namp = act.fighting_unit_nampla[unit_index]
				if need_comma != FALSE {
					log_string(", ")
				}
				log_string("PL ")
				log_string(namp.name)
				need_comma = TRUE
			}
		}
		log_string("\n\n")

	do_combat:

		/* Long names are not necessary for the rest of the action. */
		truncate_name = TRUE

		/* Do combat rounds. Stop if maximum count is reached, or if combat does not occur when do_round() is called. */

		round_number = 1

		log_summary = FALSE /* do_round() and the routines that it calls will set this for important stuff. */

		if option == PLANET_BOMBARDMENT || option == GERM_WARFARE || option == SIEGE {
			logging_disabled = TRUE
		} /* Disable logging during simulation. */

		for round_number <= max_rounds {
			if do_withdraw_check_first != FALSE {
				withdrawal_check(bat, &act)
			}

			if do_round(option, round_number, bat, &act) == FALSE {
				break
			}

			if do_withdraw_check_first == FALSE {
				withdrawal_check(bat, &act)
			}

			do_withdraw_check_first = TRUE

			regenerate_shields(&act)

			round_number++
		}

		log_summary = TRUE
		logging_disabled = FALSE

		if round_number == 1 {
			log_string("      ...But it seems that the attackers had nothing to attack!\n")
			continue
		}

		if option == PLANET_BOMBARDMENT || option == GERM_WARFARE {
			for unit_index = 0; unit_index < act.num_units_fighting; unit_index++ {
				if act.unit_type[unit_index] == GENOCIDE_NAMPLA {
					attacked_nampla = act.fighting_unit_nampla[unit_index]
					j = act.fighting_species_index[unit_index]
					for i = 0; i < num_sp; i++ {
						if x_attacked_y[i][j] != FALSE {
							species_number = bat.spec_num[i]
							log_string("      SP ")
							if field_distorted[i] != FALSE {
								log_int(distorted(species_number))
							} else {
								log_string(c_species[i].name)
							}
							log_string(" bombards SP ")
							log_string(c_species[j].name)
							log_string(" on PL ")
							log_string(attacked_nampla.name)
							log_string(".\n")

							if option == GERM_WARFARE {
								do_germ_warfare(i, j, unit_index, bat, &act)
							}
						}
					}

					/* Determine results of bombardment. */
					if option == PLANET_BOMBARDMENT {
						do_bombardment(unit_index, &act)
					}
				}
			}
		} else if option == SIEGE {
			do_siege(bat, &act)
		}
		truncate_name = FALSE
		first_action = FALSE
	}

	if battle_here == FALSE {
		if bat.num_species_here == 1 {
			log_string("    But there was no one to fight with!\n")
		} else if ambush_took_place == FALSE {
			log_string("    But no one was willing to throw the first punch!\n")
		}
	}

	/* Close combat log and append it to the log files of all species involved in this battle. */
	if prompt_gm != FALSE {
		fmt.Printf("\n  End of battle in sector %d, %d, %d.\n", bat.x, bat.y, bat.z)
	}
	fmt.Fprintf(log_file, "\n  End of battle in sector %d, %d, %d.\n", bat.x, bat.y, bat.z)
	fmt.Fprintf(summary_file, "\n  End of battle in sector %d, %d, %d.\n", bat.x, bat.y, bat.z)
	log_file.Close()
	summary_file.Close()

	for species_index = 0; species_index < num_sp; species_index++ {
		species_number = bat.spec_num[species_index]
		/* Open combat log file for reading. */
		if bat.summary_only[species_index] != FALSE {
			combat_log = fopen_r("summary.log")
		} else {
			combat_log = fopen_r("combat.log")
		}

		if combat_log == nil {
			fmt.Fprintf(os.Stderr, "\n\tCannot open combat log for reading!\n\n")
			os.Exit(255)
		}

		/* Open a temporary species log file for appending. */
		filename = fmt.Sprintf("sp%02d.temp.log", species_number)
		species_log, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n\tCannot open '%s' for appending!\n\n", filename)
			os.Exit(255)
		}

		/* Copy combat log to temporary species log. */
		for {
			log_line, ok = readln(combat_log, 256)
			if !ok {
				break
			}
			species_log.WriteString(log_line)
		}

		species_log.Close()
		combat_log.fclose()

		append_log[species_number-1] = TRUE

		/* Get rid of ships that were destroyed. */
		if data_modified[species_number-1] == FALSE {
			continue
		}
		for i = 0; i < c_species[species_index].num_ships; i++ {
			sh = c_ship[species_index][i]

			if sh.age < 50 {
				continue
			}
			if sh.pn == 99 {
				continue
			}
			if sh.x != bat.x {
				continue
			}
			if sh.y != bat.y {
				continue
			}
			if sh.z != bat.z {
				continue
			}
			if sh.status == UNDER_CONSTRUCTION {
				continue
			}

			delete_ship(sh)
		}
	}
}
