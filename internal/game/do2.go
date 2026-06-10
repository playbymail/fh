package game

import (
	"fmt"
	"os"
)

// Port of do.c, part 2 of 3 (lines 2247-3907 of the C source): the JUMP,
// LAND, MESSAGE, MOVE, NAME, NEUTRAL, ORBIT, and PRODUCTION order
// handlers. Parts 1 and 3 are in do1.go and do3.go.

func do_JUMP_command(jumped_in_combat, using_jump_portal int) {
	var n, found, max_xyz, temp_x, temp_y, temp_z, difference int
	var status, mishap_gv int

	var mishap_chance, success_chance int

	var temp_string string
	var original_line_pointer int

	var mishap_age int

	/* Set default status at end of jump. */
	status = IN_DEEP_SPACE

	/* Check if this ship jumped in combat. */
	if jumped_in_combat != FALSE {
		x = ship.dest_x
		y = ship.dest_y
		z = ship.dest_z
		pn = 0
		using_jump_portal = FALSE
		nampla = nil
		goto do_jump
	}

	/* Get ship making the jump. */
	original_line_pointer = input_line_pointer
	found = get_ship()
	if found == FALSE {
		input_line_pointer = original_line_pointer
		fix_separator()    /* Check for missing comma or tab. */
		found = get_ship() /* Try again. */
		if found == FALSE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Invalid ship name in JUMP or PJUMP command.\n")
			return
		}
	}

	/* Make sure ship is not salvage of a disbanded colony. */
	if disbanded_ship(ship) != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! This ship is salvage of a disbanded colony!\n")
		return
	}

	/* Check if this ship withdrew or was was forced to jump from combat.
	   If so, ignore specified coordinates and use those provided by the
	   combat program. */
	if ship.status == FORCED_JUMP || ship.status == JUMPED_IN_COMBAT {
		x = ship.dest_x
		y = ship.dest_y
		z = ship.dest_z
		pn = 0
		jumped_in_combat = TRUE
		using_jump_portal = FALSE
		nampla = nil
		goto do_jump
	}

	/* Make sure ship can jump. */
	if ship.status == UNDER_CONSTRUCTION {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! %s is still under construction!\n",
			ship_name(ship))
		return
	}

	if ship.ship_type == STARBASE ||
		(using_jump_portal == FALSE && ship.ship_type == SUB_LIGHT) {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! %s cannot make an interstellar jump!\n",
			ship_name(ship))
		return
	}

	/* Check if JUMP, MOVE, or WORMHOLE was already done for this ship. */
	if ship.just_jumped != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! %s already jumped or moved this turn!\n",
			ship_name(ship))
		return
	}

	/* Get the destination. */
	original_line_pointer = input_line_pointer
	found = get_location()
	if found == FALSE {
		if using_jump_portal != FALSE {
			input_line_pointer = original_line_pointer
			fix_separator()        /* Check for missing comma or tab. */
			found = get_location() /* Try again. */
		}

		if found == FALSE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Invalid destination in JUMP or PJUMP command.\n")
			return
		}
	}

	/* Set status to IN_ORBIT if destination is a planet. */
	if pn > 0 {
		status = IN_ORBIT
	}

	/* Check if a jump portal is being used. */
	if using_jump_portal != FALSE {
		found = get_jump_portal()
		if found == FALSE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Invalid starbase name in PJUMP command.\n")
			return
		}
	}

	/* If using a jump portal, make sure that starbase has sufficient number
	   of jump portal units. */
	if using_jump_portal != FALSE {
		if jump_portal_units < ship.tonnage {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Starbase does not have enough Jump Portal Units!\n")
			return
		}
	}

do_jump:

	if x == ship.x && y == ship.y && z == ship.z {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! %s was already at specified x,y,z.\n",
			ship_name(ship))
		return
	}

	/* Set flags to show that ship jumped this turn. */
	ship.just_jumped = TRUE

	/* Calculate basic mishap probability. */
	if using_jump_portal != FALSE {
		mishap_age = jump_portal_age
		mishap_gv = jump_portal_gv
	} else {
		mishap_age = ship.age
		mishap_gv = species.tech_level[GV]
	}
	mishap_chance = (100 * (((x - ship.x) * (x - ship.x)) +
		((y - ship.y) * (y - ship.y)) +
		((z - ship.z) * (z - ship.z)))) /
		mishap_gv

	if mishap_chance > 10000 {
		mishap_chance = 10000
		goto start_jump
	}

	/* Add aging effect. */
	if mishap_age > 0 {
		success_chance = 10000 - mishap_chance
		success_chance -= (2 * mishap_age * success_chance) / 100
		if success_chance < 0 {
			success_chance = 0
		}
		mishap_chance = 10000 - success_chance
	}

start_jump:

	log_string("    ")
	log_string(ship_name(ship))
	log_string(" will try to jump to ")

	if nampla == nil {
		log_int(x)
		log_char(' ')
		log_int(y)
		log_char(' ')
		log_int(z)
	} else {
		log_string("PL ")
		log_string(nampla.name)
	}

	if using_jump_portal != FALSE {
		log_string(" via jump portal ")
		log_string(jump_portal_name)

		if using_alien_portal != FALSE && first_pass == FALSE {
			/* Define this transaction. */
			if num_transactions == MAX_TRANSACTIONS {
				fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS!\n\n")
				os.Exit(255)
			}

			n = num_transactions
			num_transactions++
			transaction[n].trans_type = ALIEN_JUMP_PORTAL_USAGE
			transaction[n].number1 = other_species_number
			transaction[n].name1 = species.name
			transaction[n].name2 = ship_name(ship)
			transaction[n].name3 = ship_name(alien_portal)
		}
	}

	temp_string = fmt.Sprintf(" (%d.%02d%%).\n", mishap_chance/100, mishap_chance%100)
	log_string(temp_string)

jump_again:

	if first_pass != FALSE || rnd(10000) > mishap_chance {
		ship.x = x
		ship.y = y
		ship.z = z
		ship.pn = pn
		ship.status = status

		if first_pass == FALSE {
			star_visited(x, y, z)
		}

		return
	}

	/* Ship had a mishap. Check if it has any fail-safe jump units. */
	if ship.item_quantity[FS] > 0 {
		if num_transactions == MAX_TRANSACTIONS {
			fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS in do_JUMP_command!\n\n")
			os.Exit(255)
		}

		n = num_transactions
		num_transactions++
		transaction[n].trans_type = SHIP_MISHAP
		transaction[n].value = 4 /* Use of one FS. */
		transaction[n].number1 = species_number
		transaction[n].name1 = ship_name(ship)

		ship.item_quantity[FS] -= 1
		goto jump_again
	}

	/* If ship was forced to jump, and it reached this point, then it
	   self-destructed. */
	if ship.status == FORCED_JUMP {
		goto self_destruct
	}

	/* Check if ship self-destructed or just mis-jumped. */
	if rnd(10000) > mishap_chance {
		/* Calculate mis-jump location. */
		max_xyz = 2*galaxy.radius - 1

	try_again:
		temp_x = -1
		if ship.x > x {
			difference = ship.x - x
		} else {
			difference = x - ship.x
		}
		difference = (2 * mishap_chance * difference) / 10000
		if difference < 3 {
			difference = 3
		}
		for temp_x < 0 || temp_x > max_xyz {
			temp_x = x - rnd(difference) + rnd(difference)
		}

		temp_y = -1
		if ship.y > y {
			difference = ship.y - y
		} else {
			difference = y - ship.y
		}
		difference = (2 * mishap_chance * difference) / 10000
		if difference < 3 {
			difference = 3
		}
		for temp_y < 0 || temp_y > max_xyz {
			temp_y = y - rnd(difference) + rnd(difference)
		}

		temp_z = -1
		if ship.z > z {
			difference = ship.z - z
		} else {
			difference = z - ship.z
		}
		difference = (2 * mishap_chance * difference) / 10000
		if difference < 3 {
			difference = 3
		}
		for temp_z < 0 || temp_z > max_xyz {
			temp_z = z - rnd(difference) + rnd(difference)
		}

		if x == temp_x && y == temp_y && z == temp_z {
			goto try_again
		}

		if num_transactions == MAX_TRANSACTIONS {
			fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS in do_JUMP_command!\n\n")
			os.Exit(255)
		}

		n = num_transactions
		num_transactions++
		transaction[n].trans_type = SHIP_MISHAP
		transaction[n].value = 3 /* Mis-jump. */
		transaction[n].number1 = species_number
		transaction[n].name1 = ship_name(ship)
		transaction[n].x = temp_x
		transaction[n].y = temp_y
		transaction[n].z = temp_z

		ship.x = temp_x
		ship.y = temp_y
		ship.z = temp_z
		ship.pn = 0

		ship.status = IN_DEEP_SPACE

		star_visited(temp_x, temp_y, temp_z)

		return
	}

self_destruct:

	/* Ship self-destructed. */
	if num_transactions == MAX_TRANSACTIONS {
		fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS in do_JUMP_command!\n\n")
		os.Exit(255)
	}

	n = num_transactions
	num_transactions++
	transaction[n].trans_type = SHIP_MISHAP
	transaction[n].value = 2 /* Self-destruction. */
	transaction[n].number1 = species_number
	transaction[n].name1 = ship_name(ship)

	delete_ship(ship)
}

func do_LAND_command() {
	var i, n, found, siege_effectiveness, landing_detected, landed int
	var alien_number, alien_index, alien_pn, array_index, bit_number int
	var requested_alien_landing, alien_here, already_logged int
	var bit_mask uint32
	var original_line_pointer int
	var alien *species_data_t
	var alien_nampla *nampla_data_t

	/* Get the ship. */
	original_line_pointer = input_line_pointer
	found = get_ship()
	if found == FALSE {
		/* Check for missing comma or tab after ship name. */
		input_line_pointer = original_line_pointer
		fix_separator()
		found = get_ship()
		if found == FALSE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Invalid ship name in LAND command.\n")
			return
		}
	}

	/* Make sure the ship is not a starbase. */
	if ship.ship_type == STARBASE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! A starbase cannot land on a planet!\n")
		return
	}

	if ship.status == UNDER_CONSTRUCTION {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Ship is still under construction.\n")
		return
	}

	if ship.status == FORCED_JUMP || ship.status == JUMPED_IN_COMBAT {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Ship jumped during combat and is still in transit.\n")
		return
	}

	/* Get the planet number, if specified. */
	found = get_value()

get_planet:

	alien_pn = 0
	alien_here = FALSE
	requested_alien_landing = FALSE
	landed = FALSE
	if found == FALSE {
		found = get_location()
		if found == FALSE || nampla == nil {
			found = FALSE
		}
	} else {
		/* Check if we or another species that has declared us ALLY has a colony on this planet. */
		found = FALSE
		alien_pn = value
		requested_alien_landing = TRUE
		array_index = (species_number - 1) / 32
		bit_number = (species_number - 1) % 32
		bit_mask = 1 << uint(bit_number)
		for alien_index = 0; alien_index < galaxy.num_species; alien_index++ {
			if data_in_memory[alien_index] == FALSE {
				continue
			}
			alien = &spec_data[alien_index]
			for i = 0; i < alien.num_namplas; i++ {
				alien_nampla = namp_data[alien_index][i]
				if ship.x != alien_nampla.x {
					continue
				}
				if ship.y != alien_nampla.y {
					continue
				}
				if ship.z != alien_nampla.z {
					continue
				}
				if alien_pn != alien_nampla.pn {
					continue
				}
				if (alien_nampla.status & POPULATED) == 0 {
					continue
				}
				if alien_index == species_number-1 {
					/* We have a colony here. No permission needed. */
					nampla = alien_nampla
					found = TRUE
					alien_here = FALSE
					requested_alien_landing = FALSE
					goto finish_up
				}
				alien_here = TRUE
				if (alien.ally[array_index] & bit_mask) == 0 {
					continue
				}
				found = TRUE
				break
			}
			if found != FALSE {
				break
			}
		}
	}

finish_up:

	already_logged = FALSE
	if requested_alien_landing != FALSE && alien_here != FALSE {
		/* Notify the other alien(s). */
		landed = found
		for alien_index = 0; alien_index < galaxy.num_species; alien_index++ {
			if data_in_memory[alien_index] == FALSE {
				continue
			}
			if alien_index == species_number-1 {
				continue
			}
			alien = &spec_data[alien_index]
			for i = 0; i < alien.num_namplas; i++ {
				alien_nampla = namp_data[alien_index][i]
				if ship.x != alien_nampla.x {
					continue
				}
				if ship.y != alien_nampla.y {
					continue
				}
				if ship.z != alien_nampla.z {
					continue
				}
				if alien_pn != alien_nampla.pn {
					continue
				}
				if (alien_nampla.status & POPULATED) == 0 {
					continue
				}
				if (alien.ally[array_index] & bit_mask) != 0 {
					found = TRUE
				} else {
					found = FALSE
				}
				if landed != FALSE && found == FALSE {
					continue
				}
				if landed != FALSE {
					log_string("    ")
				} else {
					log_string("!!! ")
				}
				log_string(ship_name(ship))
				if landed != FALSE {
					log_string(" was granted")
				} else {
					log_string(" was denied")
				}
				log_string(" permission to land on PL ")
				log_string(alien_nampla.name)
				log_string(" by SP ")
				log_string(alien.name)
				log_string(".\n")
				already_logged = TRUE
				nampla = alien_nampla
				if first_pass != FALSE {
					break
				}
				/* Define a 'landing request' transaction. */
				if num_transactions == MAX_TRANSACTIONS {
					fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS!\n\n")
					os.Exit(255)
				}
				n = num_transactions
				num_transactions++
				transaction[n].trans_type = LANDING_REQUEST
				transaction[n].value = landed
				transaction[n].number1 = alien_index + 1
				transaction[n].name1 = alien_nampla.name
				transaction[n].name2 = ship_name(ship)
				transaction[n].name3 = species.name
				break
			}
		}
		found = TRUE
	}
	if alien_here != FALSE && landed == FALSE {
		return
	}
	if found == FALSE {
		if (ship.status == IN_ORBIT || ship.status == ON_SURFACE) && requested_alien_landing == FALSE {
			/* Player forgot to specify planet. Use the one it's already at. */
			value = ship.pn
			found = TRUE
			goto get_planet
		}
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Invalid or missing planet in LAND command.\n")
		return
	}

	/* Make sure the ship and the planet are in the same star system. */
	if ship.x != nampla.x || ship.y != nampla.y || ship.z != nampla.z {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Ship and planet are not in the same sector.\n")
		return
	}

	/* Make sure planet is populated. */
	if (nampla.status & POPULATED) == 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Planet in LAND command is not populated.\n")
		return
	}

	/* Move the ship. */
	ship.pn = nampla.pn
	ship.status = ON_SURFACE

	if already_logged != FALSE {
		return
	}

	/* If the planet is under siege, the landing may be detected by the besiegers. */
	log_string("    ")
	log_string(ship_name(ship))

	if nampla.siege_eff != 0 {
		if first_pass != FALSE {
			log_string(" will attempt to land on PL ")
			log_string(nampla.name)
			log_string(" in spite of the siege")
		} else {
			if nampla.siege_eff < 0 {
				siege_effectiveness = -nampla.siege_eff
			} else {
				siege_effectiveness = nampla.siege_eff
			}
			landing_detected = FALSE
			if rnd(100) <= siege_effectiveness {
				landing_detected = TRUE
				for i = 0; i < num_transactions; i++ {
					/* Find out who is besieging this planet. */
					if transaction[i].trans_type != BESIEGE_PLANET {
						continue
					}
					if transaction[i].x != nampla.x {
						continue
					}
					if transaction[i].y != nampla.y {
						continue
					}
					if transaction[i].z != nampla.z {
						continue
					}
					if transaction[i].pn != nampla.pn {
						continue
					}
					if transaction[i].number2 != species_number {
						continue
					}
					alien_number = transaction[i].number1
					/* Define a 'detection' transaction. */
					if num_transactions == MAX_TRANSACTIONS {
						fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS!\n\n")
						os.Exit(255)
					}
					n = num_transactions
					num_transactions++
					transaction[n].trans_type = DETECTION_DURING_SIEGE
					transaction[n].value = 1 /* Landing. */
					transaction[n].name1 = nampla.name
					transaction[n].name2 = ship_name(ship)
					transaction[n].name3 = species.name
					transaction[n].number3 = alien_number
				}
			}
			if rnd(100) <= siege_effectiveness {
				/* Ship doesn't know if it was detected. */
				log_string(" may have been detected by the besiegers when it landed on PL ")
				log_string(nampla.name)
			} else {
				/* Ship knows whether or not it was detected. */
				if landing_detected != FALSE {
					log_string(" was detected by the besiegers when it landed on PL ")
					log_string(nampla.name)
				} else {
					log_string(" landed on PL ")
					log_string(nampla.name)
					log_string(" without being detected by the besiegers")
				}
			}
		}
	} else {
		if first_pass != FALSE {
			log_string(" will land on PL ")
		} else {
			log_string(" landed on PL ")
		}
		log_string(nampla.name)
	}
	log_string(".\n")
}

func do_MESSAGE_command() {
	var i, message_number, bad_species int
	var unterminated_message int
	var c1, c2, c3 byte
	var filename string
	var message_file *os.File

	/* Get destination of message. */
	if get_species_name() == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid species name in MESSAGE command.\n")
		bad_species = TRUE
	} else {
		bad_species = FALSE
	}

	/* Generate a random number, create a filename with it, and use it to store message. */
	if first_pass == FALSE && bad_species == FALSE {
		for {
			/* Generate a random filename. */
			message_number = 100000 + rnd(32000)
			filename = fmt.Sprintf("m%d.msg", message_number)
			/* Make sure that this filename is not already in use. */
			if _, err := os.Stat(filename); err == nil {
				/* File already exists. Try again. */
				continue
			}
			break
		}
		var err error
		message_file, err = os.Create(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "do_MESSAGE_command: %s\n", err)
			fmt.Fprintf(os.Stderr, "\n\n!!! Cannot open message file '%s' for writing !!!\n\n", filename)
			os.Exit(255)
		}
	}

	/* Copy message to file. */
	unterminated_message = FALSE
	for {
		/* Read next line. */
		var ok bool
		input_line, ok = readln(input_file, 256)
		if !ok {
			unterminated_message = TRUE
			end_of_file = TRUE
			break
		}
		input_line_pointer = 0
		skip_whitespace()
		c1 = at(input_line, input_line_pointer)
		input_line_pointer++
		c2 = at(input_line, input_line_pointer)
		input_line_pointer++
		c3 = at(input_line, input_line_pointer)
		c1 = toupper(c1)
		c2 = toupper(c2)
		c3 = toupper(c3)
		if c1 == 'Z' && c2 == 'Z' && c3 == 'Z' {
			break
		}
		if first_pass == FALSE && bad_species == FALSE {
			message_file.WriteString(input_line)
		}
	}

	if bad_species != FALSE {
		return
	}

	/* Log the result. */
	log_string("    A message was sent to SP ")
	log_string(g_spec_name)
	log_string(".\n")

	if unterminated_message != FALSE {
		log_string("  ! WARNING: Message was not properly terminated with ZZZ!")
		log_string(" Any orders that follow the message will be assumed")
		log_string(" to be part of the message and will be ignored!\n")
	}

	if first_pass != FALSE {
		return
	}

	message_file.Close()

	/* Define this message transaction and add to list of transactions. */
	if num_transactions == MAX_TRANSACTIONS {
		fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS!\n\n")
		os.Exit(255)
	}

	i = num_transactions
	num_transactions++
	transaction[i].trans_type = MESSAGE_TO_SPECIES
	transaction[i].value = message_number
	transaction[i].number1 = species_number
	transaction[i].name1 = species.name
	transaction[i].number2 = g_spec_number
	transaction[i].name2 = g_spec_name
}

func do_MOVE_command() {
	var i, n int
	original_line_pointer := input_line_pointer
	found := get_ship()
	if found == FALSE {
		/* Check for missing comma or tab after ship name. */
		input_line_pointer = original_line_pointer
		fix_separator()
		found = get_ship()
		if found == FALSE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Invalid ship name in MOVE command.\n")
			return
		}
	}

	if ship.status == UNDER_CONSTRUCTION {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Ship is still under construction.\n")
		return
	}

	if ship.status == FORCED_JUMP || ship.status == JUMPED_IN_COMBAT {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Ship jumped during combat and is still in transit.\n")
		return
	}

	/* Check if JUMP or MOVE was already done for this ship. */
	if ship.just_jumped != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! %s already jumped or moved this turn!\n",
			ship_name(ship))
		return
	}

	/* Make sure ship is not salvage of a disbanded colony. */
	if disbanded_ship(ship) != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! This ship is salvage of a disbanded colony!\n")
		return
	}

	/* Get the planet. */
	found = get_location()
	if found == FALSE || nampla != nil {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! You may not use a planet name in MOVE command.\n")
		return
	}

	/* Check if deltas are acceptable. */
	i = x - ship.x
	if i < 0 {
		n = -i
	} else {
		n = i
	}
	i = y - ship.y
	if i < 0 {
		n += -i
	} else {
		n += i
	}
	i = z - ship.z
	if i < 0 {
		n += -i
	} else {
		n += i
	}
	if n > 1 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Destination is too far in MOVE command.\n")
		return
	}

	/* Move the ship. */
	ship.x = x
	ship.y = y
	ship.z = z
	ship.pn = 0
	ship.status = IN_DEEP_SPACE
	ship.just_jumped = 50

	if first_pass == FALSE {
		star_visited(x, y, z)
	}

	/* Log result. */
	log_string("    ")
	log_string(ship_name(ship))
	if first_pass != FALSE {
		log_string(" will move to sector ")
	} else {
		log_string(" moved to sector ")
	}
	log_int(x)
	log_char(' ')
	log_int(y)
	log_char(' ')
	log_int(z)
	log_string(".\n")
}

func do_NAME_command() {
	var found, name_length, unused_nampla_available int
	var upper_nampla_name string
	var original_line_pointer int
	var planet *planet_data_t
	var unused_nampla *nampla_data_t

	/* Get x y z coordinates. */
	found = get_location()
	if found == FALSE || nampla != nil || pn == 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid coordinates in NAME command.\n")
		return
	}

	/* Get planet abbreviation. */
	skip_whitespace()
	original_line_pointer = input_line_pointer
	if get_class_abbr() != PLANET_ID {
		/* Check if PL was mispelled (i.e, "PT" or "PN"). Otherwise
		   assume that it was accidentally omitted. */
		/* tolower(c) != 'p' is equivalent to toupper(c) != 'P' for ASCII. */
		if toupper(at(input_line, original_line_pointer)) != 'P' ||
			isalnum(at(input_line, original_line_pointer+2)) {
			input_line_pointer = original_line_pointer
		}
	}

	/* Get planet name. */
	name_length = get_name()
	if name_length < 1 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid planet name in NAME command.\n")
		return
	}

	/* Search existing namplas for name and location. */
	found = FALSE
	unused_nampla_available = FALSE
	for nampla_index = 0; nampla_index < species.num_namplas; nampla_index++ {
		nampla = nampla_base[nampla_index]

		if nampla.pn == 99 {
			/* We can re-use this nampla rather than append a new one. */
			unused_nampla = nampla
			unused_nampla_available = TRUE
			continue
		}

		/* Check if a named planet already exists at this location. */
		if nampla.x == x && nampla.y == y && nampla.z == z && nampla.pn == pn {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! The planet at these coordinates already has a name.\n")
			return
		}

		/* Make upper case copy of nampla name. */
		upper_nampla_name = upcase(nampla.name)

		/* Compare names. */
		if upper_nampla_name == upper_name {
			found = TRUE
			break
		}
	}

	if found != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Planet in NAME command already exists.\n")
		return
	}

	/* Add new nampla to database for this species. */
	if unused_nampla_available != FALSE {
		nampla = unused_nampla
	} else {
		num_new_namplas[species_index]++
		if num_new_namplas[species_index] > NUM_EXTRA_NAMPLAS {
			fmt.Fprintf(os.Stderr, "\n\n\tInsufficient memory for new planet name:\n")
			fmt.Fprintf(os.Stderr, "\n\t%s\n", input_line)
			os.Exit(255)
		}
		/* nampla = nampla_base + species->num_namplas; -- the C code uses
		   pre-allocated headroom records; in Go grow the slice on demand. */
		if species.num_namplas < len(nampla_base) {
			nampla = nampla_base[species.num_namplas]
		} else {
			nampla = &nampla_data_t{}
			nampla_base = append(nampla_base, nampla)
			namp_data[species_index] = nampla_base
		}
		species.num_namplas += 1
		/* Set everything to zero. */
		delete_nampla(nampla)
	}

	/* Initialize new nampla. */
	nampla.name = original_name
	nampla.x = x
	nampla.y = y
	nampla.z = z
	nampla.pn = pn
	nampla.status = COLONY
	nampla.planet_index = star.planet_index + pn - 1
	planet = planet_base[nampla.planet_index]
	nampla.message = planet.message
	/* Everything else was set to zero in above call to 'delete_nampla'. */

	/* Mark sector as having been visited. */
	star_visited(x, y, z)

	/* Log result. */
	log_string("    Named PL ")
	log_string(nampla.name)
	log_string(" at ")
	log_int(nampla.x)
	log_char(' ')
	log_int(nampla.y)
	log_char(' ')
	log_int(nampla.z)
	log_string(", planet #")
	log_int(nampla.pn)
	log_string(".\n")
}

func do_NEUTRAL_command() {
	var i, array_index, bit_number int
	var bit_mask uint32

	/* See if declaration is for all species. */
	if get_value() != 0 {
		bit_mask = 0
		for i = 0; i < NUM_CONTACT_WORDS; i++ {
			species.enemy[i] = bit_mask /* Clear all enemy bits. */
			species.ally[i] = bit_mask  /* Clear all ally bits. */
		}
	} else {
		/* Get name of species. */
		if get_species_name() == FALSE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! Invalid or missing argument in NEUTRAL command.\n")
			return
		}

		/* Get array index and bit mask. */
		array_index = (g_spec_number - 1) / 32
		bit_number = (g_spec_number - 1) % 32
		bit_mask = 1 << uint(bit_number)

		/* Clear the appropriate bit. */
		species.enemy[array_index] &= ^bit_mask /* Clear enemy bit. */
		species.ally[array_index] &= ^bit_mask  /* Clear ally bit. */
	}

	/* Log the result. */
	log_string("    Neutrality was declared towards ")
	if bit_mask == 0 {
		log_string("ALL species")
	} else {
		log_string("SP ")
		log_string(g_spec_name)
	}
	log_string(".\n")
}

func do_ORBIT_command() {
	var i, found, specified_planet_number int
	var original_line_pointer int

	/* Get the ship. */
	original_line_pointer = input_line_pointer
	found = get_ship()
	if found == FALSE {
		/* Check for missing comma or tab after ship name. */
		input_line_pointer = original_line_pointer
		fix_separator()
		found = get_ship()
		if found == FALSE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Invalid ship name in ORBIT command.\n")
			return
		}
	}

	if ship.status == UNDER_CONSTRUCTION {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Ship is still under construction.\n")
		return
	}

	if ship.status == FORCED_JUMP || ship.status == JUMPED_IN_COMBAT {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Ship jumped during combat and is still in transit.\n")
		return
	}

	/* Make sure this ship didn't just arrive via a MOVE command. */
	if ship.just_jumped == 50 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! ORBIT not allowed immediately after a MOVE!\n")
		return
	}

	/* Make sure ship is not salvage of a disbanded colony. */
	if disbanded_ship(ship) != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! This ship is salvage of a disbanded colony!\n")
		return
	}

	/* Get the planet. */
	specified_planet_number = get_value()

get_planet:

	if specified_planet_number != 0 {
		found = FALSE
		specified_planet_number = value
		for i = 0; i < num_stars; i++ {
			star = star_base[i]
			if star.x != ship.x {
				continue
			}
			if star.y != ship.y {
				continue
			}
			if star.z != ship.z {
				continue
			}
			if specified_planet_number >= 1 && specified_planet_number <= star.num_planets {
				found = TRUE
			}
			break
		}

		if found == FALSE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Invalid planet in ORBIT command.\n")
			return
		}

		ship.pn = specified_planet_number

		goto finish_up
	}

	found = get_location()
	if found == FALSE || nampla == nil {
		if ship.status == IN_ORBIT || ship.status == ON_SURFACE {
			/* Player forgot to specify planet. Use the one it's already at. */
			specified_planet_number = ship.pn
			value = specified_planet_number
			goto get_planet
		}

		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Invalid or missing planet in ORBIT command.\n")
		return
	}

	/* Make sure the ship and the planet are in the same star system. */
	if ship.x != nampla.x || ship.y != nampla.y || ship.z != nampla.z {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Ship and planet are not in the same sector.\n")
		return
	}

	/* Move the ship. */
	ship.pn = nampla.pn

finish_up:

	ship.status = IN_ORBIT

	/* If a planet number is being used, see if it has a name.  If so, use the name. */
	if specified_planet_number != 0 {
		for i = 0; i < species.num_namplas; i++ {
			nampla = nampla_base[i]
			if nampla.x != ship.x {
				continue
			}
			if nampla.y != ship.y {
				continue
			}
			if nampla.z != ship.z {
				continue
			}
			if nampla.pn != specified_planet_number {
				continue
			}
			specified_planet_number = 0
			break
		}
	}

	/* Log result. */
	log_string("    ")
	log_string(ship_name(ship))
	if first_pass != FALSE {
		log_string(" will enter orbit around ")
	} else {
		log_string(" entered orbit around ")
	}
	if specified_planet_number != 0 {
		log_string("planet number ")
		log_int(specified_planet_number)
	} else {
		log_string("PL ")
		log_string(nampla.name)
	}
	log_string(".\n")
}

func do_PRODUCTION_command(missing_production_order int) {
	var i, j, abbr_type, found, alien_number, under_siege,
		siege_percent_effectiveness, new_alien, num_siege_ships,
		mining_colony, resort_colony, special_colony, ship_index,
		enemy_on_same_planet, trans_index, production_penalty,
		ls_needed, shipyards_for_this_species int

	var upper_nampla_name string

	var n, RMs_produced, total_siege_effectiveness,
		EUs_available_for_siege,
		EUs_for_distribution, EUs_for_this_species, total_EUs_stolen,
		special_production,
		alien_pop_units, total_alien_pop_here, total_besieged_pop,
		ib_for_this_species, ab_for_this_species, total_ib, total_ab,
		total_effective_tonnage int

	var siege_effectiveness, pop_units_here [MAX_SPECIES + 1]int

	var alien *species_data_t
	var alien_nampla_base []*nampla_data_t
	var alien_nampla *nampla_data_t
	var alien_ship_base []*ship_data_t
	var alien_ship *ship_data_t
	var ship *ship_data_t

	if doing_production != FALSE {
		/* Terminate production for previous planet. */
		if last_planet_produced != FALSE {
			transfer_balance()
			last_planet_produced = FALSE
		}

		/* Give gamemaster option to abort. */
		if first_pass != FALSE {
			gamemaster_abort_option()
		}
		log_char('\n')
	}

	doing_production = TRUE

	if missing_production_order != FALSE {
		nampla = next_nampla
		nampla_index = next_nampla_index

		goto got_nampla
	}

	/* Get PL abbreviation. */
	abbr_type = get_class_abbr()

	if abbr_type != PLANET_ID {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid planet name in PRODUCTION command.\n")
		return
	}

	/* Get planet name. */
	get_name()

	/* Search all namplas for name. */
	found = FALSE
	for nampla_index = 0; nampla_index < species.num_namplas; nampla_index++ {
		nampla = nampla_base[nampla_index]

		if nampla.pn == 99 {
			continue
		}

		/* Make upper case copy of nampla name. */
		upper_nampla_name = upcase(nampla.name)

		/* Compare names. */
		if upper_nampla_name == upper_name {
			found = TRUE
			break
		}
	}

	if found == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid planet name in PRODUCTION command.\n")
		return
	}

	/* Check if production was already done for this planet. */
	if production_done[nampla_index] != 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! More than one PRODUCTION command for planet.\n")
		return
	}
	production_done[nampla_index] = TRUE

	/* Check if this colony was disbanded. */
	if nampla.status&DISBANDED_COLONY != 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Production orders cannot be given for a disbanded colony!\n")
		return
	}

got_nampla:

	last_planet_produced = TRUE
	shipyard_built = FALSE
	shipyard_capacity = nampla.shipyards

	/* See if this is a mining or resort colony. */
	mining_colony = FALSE
	resort_colony = FALSE
	special_colony = FALSE
	if nampla.status&MINING_COLONY != 0 {
		mining_colony = TRUE
		special_colony = TRUE
	} else if nampla.status&RESORT_COLONY != 0 {
		resort_colony = TRUE
		special_colony = TRUE
	}

	/* Get planet data for this nampla. */
	planet = planet_base[nampla.planet_index]

	/* Check if fleet maintenance cost is so high that riots ensued. */
	i = 0
	j = (species.fleet_percent_cost - 10000) / 100
	if rnd(100) <= j {
		log_string("!!! WARNING! Riots on PL ")
		log_string(nampla.name)
		log_string(" due to excessive and unpopular military build-up reduced ")

		if mining_colony != FALSE || special_colony == FALSE {
			log_string("mining base by ")
			i = rnd(j)
			log_int(i)
			log_string(" percent ")
			nampla.mi_base -= (i * nampla.mi_base) / 100
		}

		if resort_colony != FALSE || special_colony == FALSE {
			if i != 0 {
				log_string("and ")
			}
			log_string("manufacturing base by ")
			i = rnd(j)
			log_int(i)
			log_string(" percent")
			nampla.ma_base -= (i * nampla.ma_base) / 100
		}
		log_string("!\n\n")
	}

	/* Calculate "balance" available for spending and create pseudo "checking account". */
	ls_needed = life_support_needed(species, home_planet, planet)

	if ls_needed == 0 {
		production_penalty = 0
	} else {
		production_penalty = (100 * ls_needed) / species.tech_level[LS]
	}

	RMs_produced = (10 * species.tech_level[MI] * nampla.mi_base) / planet.mining_difficulty
	RMs_produced -= (production_penalty * RMs_produced) / 100
	RMs_produced = ((planet.econ_efficiency * RMs_produced) + 50) / 100

	if special_colony != FALSE {
		/* RMs just 'sitting' on the planet cannot be converted to EUs on a mining colony, and cannot create a 'balance' on a resort colony. */
		raw_material_units = 0
	} else {
		raw_material_units = RMs_produced + nampla.item_quantity[RM]
	}

	production_capacity = (species.tech_level[MA] * nampla.ma_base) / 10
	production_capacity -= (production_penalty * production_capacity) / 100
	production_capacity = ((planet.econ_efficiency * production_capacity) + 50) / 100

	if raw_material_units > production_capacity {
		balance = production_capacity
	} else {
		balance = raw_material_units
	}

	if species.fleet_percent_cost > 10000 {
		n = 10000
	} else {
		n = species.fleet_percent_cost
	}

	if special_colony != FALSE {
		EU_spending_limit = 0
	} else {
		/* Only excess RMs may be recycled. */
		nampla.item_quantity[RM] = raw_material_units - balance

		balance -= ((n * balance) + 5000) / 10000
		raw_material_units = balance
		production_capacity = balance
		EUs_available_for_siege = balance
		if nampla.status&HOME_PLANET != 0 {
			if species.hp_original_base != 0 {
				/* HP was bombed. */
				EU_spending_limit = 4 * balance /* Factor = 4 + 1 = 5. */
			} else {
				EU_spending_limit = species.econ_units
			}
		} else {
			EU_spending_limit = balance
		}
	}

	/* Log what was done. Balances for mining and resort colonies will always
	   be zero and should not be printed. */
	log_string("  Start of production on PL ")
	log_string(nampla.name)
	log_char('.')
	if special_colony == FALSE {
		log_string(" (Initial balance is ")
		log_long(balance)
		log_string(".)")
	}
	log_char('\n')

	/* If this IS a mining or resort colony, convert RMs or production capacity to EUs. */
	if mining_colony != FALSE {
		special_production = (2 * RMs_produced) / 3
		special_production -= ((n * special_production) + 5000) / 10000
		log_string("    Mining colony ")
	} else if resort_colony != FALSE {
		special_production = (2 * production_capacity) / 3
		special_production -= ((n * special_production) + 5000) / 10000
		log_string("    Resort colony ")
	}

	if special_colony != FALSE {
		log_string(nampla.name)
		log_string(" generated ")
		log_long(special_production)
		log_string(" economic units.\n")

		EUs_available_for_siege = special_production
		species.econ_units += special_production

		if mining_colony != FALSE && first_pass == FALSE {
			planet.mining_difficulty += RMs_produced / 150
			planet_data_modified = TRUE
		}
	}

	/* Check if this planet is under siege. */
	nampla.siege_eff = 0
	under_siege = FALSE
	alien_number = 0
	num_siege_ships = 0
	total_siege_effectiveness = 0
	enemy_on_same_planet = FALSE
	total_alien_pop_here = 0
	for i = 1; i <= MAX_SPECIES; i++ {
		siege_effectiveness[i] = 0
		pop_units_here[i] = 0
	}

	for trans_index = 0; trans_index < num_transactions; trans_index++ {
		/* Check if this is a siege of this nampla. */
		if transaction[trans_index].trans_type != BESIEGE_PLANET {
			continue
		}
		if transaction[trans_index].x != nampla.x {
			continue
		}
		if transaction[trans_index].y != nampla.y {
			continue
		}
		if transaction[trans_index].z != nampla.z {
			continue
		}
		if transaction[trans_index].pn != nampla.pn {
			continue
		}
		if transaction[trans_index].number2 != species_number {
			continue
		}

		/* Check if alien ship is still in the same star system as the planet. */
		if alien_number != transaction[trans_index].number1 {
			/* First transaction for this alien. */
			alien_number = transaction[trans_index].number1
			if data_in_memory[alien_number-1] == FALSE {
				fmt.Fprintf(os.Stderr, "\n\tData for species #%d should be in memory but is not!\n\n", alien_number)
				os.Exit(255)
			}
			alien = &spec_data[alien_number-1]
			alien_nampla_base = namp_data[alien_number-1]
			alien_ship_base = ship_data[alien_number-1]

			new_alien = TRUE
		}

		/* Find the alien ship. */
		found = FALSE
		for i = 0; i < alien.num_ships; i++ {
			alien_ship = alien_ship_base[i]

			if alien_ship.pn == 99 {
				continue
			}

			if alien_ship.name == transaction[trans_index].name3 {
				found = TRUE
				break
			}
		}

		/* Check if alien ship is still at the siege location. */
		if found == FALSE {
			/* It must have jumped away and self-destructed, or was recycled. */
			continue
		}
		if alien_ship.x != nampla.x {
			continue
		}
		if alien_ship.y != nampla.y {
			continue
		}
		if alien_ship.z != nampla.z {
			continue
		}
		if alien_ship.class == TR {
			continue
		}

		/* This nampla is under siege. */
		if under_siege == FALSE {
			log_string("\n    WARNING! PL ")
			log_string(nampla.name)
			log_string(" is under siege by the following:\n      ")
			under_siege = TRUE
		}

		if num_siege_ships > 0 {
			log_string(", ")
		}
		num_siege_ships++
		if new_alien != FALSE {
			log_string(alien.name)
			log_char(' ')
			new_alien = FALSE

			/* Check if this alien has a colony on the same planet. */
			for i = 0; i < alien.num_namplas; i++ {
				alien_nampla = alien_nampla_base[i]

				if alien_nampla.x != nampla.x {
					continue
				}
				if alien_nampla.y != nampla.y {
					continue
				}
				if alien_nampla.z != nampla.z {
					continue
				}
				if alien_nampla.pn != nampla.pn {
					continue
				}

				/* Enemy population that will count for both detection AND assimilation. */
				alien_pop_units = alien_nampla.mi_base + alien_nampla.ma_base + alien_nampla.IUs_to_install +
					alien_nampla.AUs_to_install

				/* Any base over 200.0 has only 5% effectiveness. */
				if alien_pop_units > 2000 {
					alien_pop_units = (alien_pop_units-2000)/20 + 2000
				}

				/* Enemy population that counts ONLY for detection. */
				n = alien_nampla.pop_units + alien_nampla.item_quantity[CU] + alien_nampla.item_quantity[PD]

				if alien_pop_units > 0 {
					enemy_on_same_planet = TRUE
					pop_units_here[alien_number] = alien_pop_units
					total_alien_pop_here += alien_pop_units
				} else if n > 0 {
					enemy_on_same_planet = TRUE
				}

				if alien_nampla.item_quantity[PD] == 0 {
					continue
				}

				log_string("planetary defenses of PL ")
				log_string(alien_nampla.name)
				log_string(", ")

				n = (4 * alien_nampla.item_quantity[PD]) / 5
				n = (n * alien.tech_level[ML]) / (species.tech_level[ML] + 1)
				total_siege_effectiveness += n
				siege_effectiveness[alien_number] += n
			}
		}
		log_string(ship_name(alien_ship))

		/* Determine the number of planets that this ship is besieging. */
		n = 0
		for j = 0; j < num_transactions; j++ {
			if transaction[j].trans_type != BESIEGE_PLANET {
				continue
			}
			if transaction[j].number1 != alien_number {
				continue
			}
			if transaction[j].name3 != alien_ship.name {
				continue
			}

			n++
		}

		/* Determine the effectiveness of this ship on the siege. */
		if alien_ship.ship_type == STARBASE {
			i = alien_ship.tonnage /* One quarter of normal ships. */
		} else {
			i = 4 * alien_ship.tonnage
		}

		i = (i * alien.tech_level[ML]) / (species.tech_level[ML] + 1)

		i /= n

		total_siege_effectiveness += i
		siege_effectiveness[alien_number] += i
	}

	if under_siege != FALSE {
		log_string(".\n")
	} else {
		return
	}

	/* Determine percent effectiveness of the siege. */
	total_effective_tonnage = 2500 * total_siege_effectiveness

	if nampla.mi_base+nampla.ma_base == 0 {
		siege_percent_effectiveness = -9999 /* New colony with nothing installed yet. */
	} else {
		siege_percent_effectiveness = total_effective_tonnage /
			(((species.tech_level[MI] * nampla.mi_base) +
				(species.tech_level[MA] * nampla.ma_base)) / 10)
	}

	if siege_percent_effectiveness > 95 {
		siege_percent_effectiveness = 95
	} else if siege_percent_effectiveness == -9999 {
		log_string("      However, although planet is populated, it has no economic base.\n\n")
		return
	} else if siege_percent_effectiveness < 1 {
		log_string("      However, because of the weakness of the siege, it was completely ineffective!\n\n")
		return
	}

	if enemy_on_same_planet != FALSE {
		nampla.siege_eff = -siege_percent_effectiveness
	} else {
		nampla.siege_eff = siege_percent_effectiveness
	}

	log_string("      The siege is approximately ")
	log_int(siege_percent_effectiveness)
	log_string("% effective.\n")

	/* Add siege EU transfer(s). */
	EUs_for_distribution = (siege_percent_effectiveness * EUs_available_for_siege) / 100

	total_EUs_stolen = 0

	for alien_number = 1; alien_number <= MAX_SPECIES; alien_number++ {
		n = siege_effectiveness[alien_number]
		if n < 1 {
			continue
		}
		alien = &spec_data[alien_number-1]
		EUs_for_this_species = (n * EUs_for_distribution) / total_siege_effectiveness
		if EUs_for_this_species < 1 {
			continue
		}
		total_EUs_stolen += EUs_for_this_species
		log_string("      ")
		log_long(EUs_for_this_species)
		log_string(" economic unit")
		if EUs_for_this_species > 1 {
			log_string("s were")
		} else {
			log_string(" was")
		}
		log_string(" lost and 25% of the amount was transferred to SP ")
		log_string(alien.name)
		log_string(".\n")

		if first_pass != FALSE {
			continue
		}

		/* Define this transaction and add to list of transactions. */
		if num_transactions == MAX_TRANSACTIONS {
			fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS!\n\n")
			os.Exit(255)
		}

		trans_index = num_transactions
		num_transactions++
		transaction[trans_index].trans_type = SIEGE_EU_TRANSFER
		transaction[trans_index].donor = species_number
		transaction[trans_index].recipient = alien_number
		transaction[trans_index].value = EUs_for_this_species / 4
		transaction[trans_index].x = nampla.x
		transaction[trans_index].y = nampla.y
		transaction[trans_index].z = nampla.z
		transaction[trans_index].number1 = siege_percent_effectiveness
		transaction[trans_index].name1 = species.name
		transaction[trans_index].name2 = alien.name
		transaction[trans_index].name3 = nampla.name
	}
	log_char('\n')

	/* Correct balances. */
	if special_colony != FALSE {
		species.econ_units -= total_EUs_stolen
	} else {
		if check_bounced(total_EUs_stolen) != FALSE {
			fmt.Fprintf(os.Stderr, "\nWARNING! Internal error! Should never reach this point!\n\n")
			os.Exit(255)
		}
	}

	if enemy_on_same_planet == FALSE {
		return
	}

	/* All ships currently under construction may be detected by the besiegers and destroyed. */
	for ship_index = 0; ship_index < species.num_ships; ship_index++ {
		ship = ship_base[ship_index]

		if ship.status == UNDER_CONSTRUCTION && ship.x == nampla.x && ship.y == nampla.y &&
			ship.z == nampla.z && ship.pn == nampla.pn {
			if rnd(100) > siege_percent_effectiveness {
				continue
			}

			log_string("      ")
			log_string(ship_name(ship))
			log_string(", under construction when the siege began, was detected by the besiegers and destroyed!\n")
			if first_pass == FALSE {
				delete_ship(ship)
			}
		}
	}

	/* Check for assimilation. */
	if nampla.status&HOME_PLANET != 0 {
		return
	}
	if total_alien_pop_here < 1 {
		return
	}

	total_besieged_pop = nampla.mi_base + nampla.ma_base + nampla.IUs_to_install + nampla.AUs_to_install

	/* Any base over 200.0 has only 5% effectiveness. */
	if total_besieged_pop > 2000 {
		total_besieged_pop = (total_besieged_pop-2000)/20 + 2000
	}

	if total_besieged_pop/total_alien_pop_here >= 5 {
		return
	}
	if siege_percent_effectiveness < 95 {
		return
	}

	log_string("      PL ")
	log_string(nampla.name)
	log_string(" has become assimilated by the besieging species")
	log_string(" and is no longer under your control.\n\n")

	total_ib = nampla.mi_base /* My stupid compiler can't add an int and an unsigned short. */
	total_ib += nampla.IUs_to_install
	total_ab = nampla.ma_base
	total_ab += nampla.AUs_to_install

	for alien_number = 1; alien_number <= MAX_SPECIES; alien_number++ {
		n = pop_units_here[alien_number]
		if n < 1 {
			continue
		}

		shipyards_for_this_species = (n * nampla.shipyards) / total_alien_pop_here

		ib_for_this_species = (n * total_ib) / total_alien_pop_here
		total_ib -= ib_for_this_species

		ab_for_this_species = (n * total_ab) / total_alien_pop_here
		total_ab -= ab_for_this_species

		if ib_for_this_species == 0 && ab_for_this_species == 0 {
			continue
		}

		if first_pass != FALSE {
			continue
		}

		/* Define this transaction and add to list of transactions. */
		if num_transactions == MAX_TRANSACTIONS {
			fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS!\n\n")
			os.Exit(255)
		}

		trans_index = num_transactions
		num_transactions++
		transaction[trans_index].trans_type = ASSIMILATION
		transaction[trans_index].value = alien_number
		transaction[trans_index].x = nampla.x
		transaction[trans_index].y = nampla.y
		transaction[trans_index].z = nampla.z
		transaction[trans_index].pn = nampla.pn
		transaction[trans_index].number1 = ib_for_this_species / 2
		transaction[trans_index].number2 = ab_for_this_species / 2
		transaction[trans_index].number3 = shipyards_for_this_species
		transaction[trans_index].name1 = species.name
		transaction[trans_index].name2 = nampla.name
	}

	/* Erase the original colony. */
	balance = 0
	EU_spending_limit = 0
	raw_material_units = 0
	production_capacity = 0
	nampla.mi_base = 0
	nampla.ma_base = 0
	nampla.IUs_to_install = 0
	nampla.AUs_to_install = 0
	nampla.pop_units = 0
	nampla.siege_eff = 0
	nampla.status = COLONY
	nampla.shipyards = 0
	nampla.hiding = 0
	nampla.hidden = 0
	nampla.use_on_ambush = 0

	for i = 0; i < MAX_ITEMS; i++ {
		nampla.item_quantity[i] = 0
	}
}
