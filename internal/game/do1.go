package game

// Port of do.c, part 1 of 3 (lines 1-2246 of the C source):
// the AMBUSH, ALLY, BASE, BUILD, DEEP, DESTROY, DEVELOP, DISBAND,
// ENEMY, ESTIMATE, HIDE, INSTALL and INTERCEPT order handlers.
// The JUMP..PRODUCTION handlers live in do2.go and RECYCLE..WORMHOLE
// in do3.go.

import (
	"fmt"
	"os"
)

func do_AMBUSH_command() {
	var status int
	var cost int

	/* Check if this order was preceded by a PRODUCTION order. */
	if doing_production == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Missing PRODUCTION order!\n")
		return
	}

	/* Get amount to spend. */
	status = get_value()
	if status == 0 || value < 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid or missing amount.\n")
		return
	}
	if value == 0 {
		value = balance
	}
	if value == 0 {
		return
	}
	cost = value

	/* Check if planet is under siege. */
	if nampla.siege_eff != 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Besieged planet cannot ambush!\n")
		return
	}

	/* Check if sufficient funds are available. */
	if check_bounced(cost) != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Insufficient funds to execute order.\n")
		return
	}

	/* Increment amount spent on ambush. */
	nampla.use_on_ambush += cost

	/* Log transaction. */
	log_string("    Spent ")
	log_long(cost)
	log_string(" in preparation for an ambush.\n")
}

func do_ALLY_command() {
	var array_index, bit_number int
	var bit_mask uint32

	/* Get name of species that is being declared an ally. */
	if get_species_name() == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid or missing argument in ALLY command.\n")
		return
	}

	/* Get array index and bit mask. */
	array_index = (g_spec_number - 1) / 32
	bit_number = (g_spec_number - 1) % 32
	bit_mask = uint32(1) << uint(bit_number)

	/* Check if we've met this species and make sure it is not an enemy. */
	if (species.contact[array_index] & bit_mask) == 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! You can't declare alliance with a species you haven't met.\n")
		return
	}

	/* Set/clear the appropriate bit. */
	species.ally[array_index] |= bit_mask   /* Set ally bit. */
	species.enemy[array_index] &^= bit_mask /* Clear enemy bit. */

	/* Log the result. */
	log_string("    Alliance was declared with ")
	if bit_mask == 0 {
		log_string("ALL species")
	} else {
		log_string("SP ")
		log_string(g_spec_name)
	}
	log_string(".\n")
}

func do_BASE_command() {
	var i, found, su_count, original_count int
	var unused_ship_available, new_tonnage, max_tonnage, new_starbase int
	var source_is_a_planet int
	var x, y, z, pn int
	var upper_ship_name string
	var original_line_pointer int
	var source_nampla *nampla_data_t
	var source_ship, starbase, unused_ship *ship_data_t

	/* Get number of starbase units to use. */
	i = get_value()
	if i == 0 {
		value = 0
	} else {
		/* Make sure value is meaningful. */
		if value < 0 {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Invalid SU count in BASE command.\n")
			return
		}
	}
	su_count = value
	original_count = su_count

	/* Get source of starbase units. */
	original_line_pointer = input_line_pointer
	if get_transfer_point() == FALSE {
		input_line_pointer = original_line_pointer
		fix_separator() /* Check for missing comma or tab. */
		if get_transfer_point() == FALSE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Invalid source location in BASE command.\n")
			return
		}
	}

	/* Make sure everything makes sense. */
	if abbr_type == SHIP_CLASS {
		source_is_a_planet = FALSE
		source_ship = ship

		if source_ship.status == UNDER_CONSTRUCTION {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! %s is still under construction!\n",
				ship_name(source_ship))
			return
		}

		if source_ship.status == FORCED_JUMP ||
			source_ship.status == JUMPED_IN_COMBAT {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Ship jumped during combat and is still in transit.\n")
			return
		}

		if su_count == 0 {
			su_count = source_ship.item_quantity[SU]
		}
		if su_count == 0 {
			return
		}
		if source_ship.item_quantity[SU] < su_count {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! %s does not enough starbase units!\n",
				ship_name(source_ship))
			return
		}

		x = source_ship.x
		y = source_ship.y
		z = source_ship.z
		pn = source_ship.pn
	} else /* Source is a planet. */ {
		source_is_a_planet = TRUE
		source_nampla = nampla

		if su_count == 0 {
			su_count = source_nampla.item_quantity[SU]
		}
		if su_count == 0 {
			return
		}
		if source_nampla.item_quantity[SU] < su_count {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! PL %s does not have enough starbase units!\n",
				source_nampla.name)
			return
		}

		x = source_nampla.x
		y = source_nampla.y
		z = source_nampla.z
		pn = source_nampla.pn
	}

	/* Get starbase name. */
	if get_class_abbr() != SHIP_CLASS || abbr_index != BA {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Invalid starbase name.\n")
		return
	}
	get_name()

	/* Search all ships for name. */
	found = FALSE
	unused_ship_available = FALSE
	for ship_index = 0; ship_index < species.num_ships; ship_index++ {
		ship = ship_base[ship_index]

		if ship.pn == 99 {
			unused_ship_available = TRUE
			unused_ship = ship
			continue
		}

		/* Make upper case copy of ship name. */
		upper_ship_name = upcase(ship.name)

		/* Compare names. */
		if upper_ship_name == upper_name {
			found = TRUE
			break
		}
	}

	if found != FALSE {
		if ship.ship_type != STARBASE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Ship name already in use.\n")
			return
		}

		if ship.x != x || ship.y != y || ship.z != z {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Starbase units and starbase are not at same X Y Z.\n")
			return
		}
		starbase = ship
		new_starbase = FALSE
	} else {
		if unused_ship_available != FALSE {
			starbase = unused_ship
		} else {
			/* Make sure we have enough memory for new starbase. */
			if num_new_ships[species_index] == NUM_EXTRA_SHIPS {
				fmt.Fprintf(os.Stderr, "\n\n\tInsufficient memory for new starbase!\n\n")
				os.Exit(255)
			}
			num_new_ships[species_index]++
			/* C: starbase = ship_base + species->num_ships (headroom
			   allocated by ncalloc); grow the slice instead. */
			for len(ship_data[species_index]) <= species.num_ships {
				ship_data[species_index] = append(ship_data[species_index], &ship_data_t{})
			}
			ship_base = ship_data[species_index]
			starbase = ship_base[species.num_ships]
			species.num_ships++
			delete_ship(starbase) /* Initialize everything to zero. */
		}

		/* Initialize non-zero data for new ship. */
		starbase.name = original_name
		starbase.x = x
		starbase.y = y
		starbase.z = z
		starbase.pn = pn
		if pn == 0 {
			starbase.status = IN_DEEP_SPACE
		} else {
			starbase.status = IN_ORBIT
		}
		starbase.ship_type = STARBASE
		starbase.class = BA
		starbase.tonnage = 0
		starbase.age = -1
		starbase.remaining_cost = 0

		/* Everything else was set to zero in above call to 'delete_ship'. */

		new_starbase = TRUE
	}

	/* Make sure that starbase is not being built in the deep space section
	   of a star system .*/
	if starbase.pn == 0 {
		for i = 0; i < num_stars; i++ {
			star = star_base[i]

			if star.x != x {
				continue
			}
			if star.y != y {
				continue
			}
			if star.z != z {
				continue
			}

			if star.num_planets < 1 {
				break
			}

			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Starbase cannot be built in deep space if there are planets available!\n")
			if new_starbase != FALSE {
				delete_ship(starbase)
			}
			return
		}
	}

	/* Make sure species can build a starbase of this size. */
	max_tonnage = species.tech_level[MA] / 2
	new_tonnage = starbase.tonnage + su_count
	if new_tonnage > max_tonnage && original_count == 0 {
		su_count = max_tonnage - starbase.tonnage
		if su_count < 1 {
			if new_starbase != FALSE {
				delete_ship(starbase)
			}
			return
		}
		new_tonnage = starbase.tonnage + su_count
	}

	if new_tonnage > max_tonnage {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Maximum allowable tonnage exceeded.\n")
		if new_starbase != FALSE {
			delete_ship(starbase)
		}
		return
	}

	/* Finish up and log results. */
	log_string("    ")
	if starbase.tonnage == 0 {
		log_string(ship_name(starbase))
		log_string(" was constructed.\n")
	} else {
		/* Weighted average. */
		starbase.age = ((starbase.age * starbase.tonnage) - su_count) / new_tonnage
		log_string("Size of ")
		log_string(ship_name(starbase))
		log_string(" was increased to ")
		log_string(commas(10000 * new_tonnage))
		log_string(" tons.\n")
	}

	starbase.tonnage = new_tonnage

	if source_is_a_planet != FALSE {
		source_nampla.item_quantity[SU] -= su_count
	} else {
		source_ship.item_quantity[SU] -= su_count
	}
}

func do_BUILD_command(continuing_construction, interspecies_construction int) {
	var i, n, class, critical_tech, found, name_length int
	var siege_effectiveness, cost_given, new_ship, max_tonnage int
	var tonnage_increase, alien_number, cargo_on_board int
	var unused_nampla_available, unused_ship_available, capacity int
	var pop_check_needed, contact_word_number, contact_bit_number int
	var already_notified [MAX_SPECIES]int
	var upper_ship_name string
	var original_line_pointer int
	var cost, cost_argument, unit_cost, num_items, pop_reduction int
	var premium, total_cost, original_num_items int
	var contact_mask uint32
	var max_funds_available int
	var recipient_species *species_data_t
	var recipient_nampla, unused_nampla, destination_nampla, temp_nampla *nampla_data_t
	var recipient_ship, unused_ship *ship_data_t

	/* Check if this order was preceded by a PRODUCTION order. */
	if doing_production == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Missing PRODUCTION order!\n")
		return
	}

	/* Get ready if planet is under siege. */
	if nampla.siege_eff < 0 {
		siege_effectiveness = -nampla.siege_eff
	} else {
		siege_effectiveness = nampla.siege_eff
	}

	/* Get species name and make appropriate tests if this is an interspecies construction order. */
	if interspecies_construction != FALSE {
		original_line_pointer = input_line_pointer
		if get_species_name() == FALSE {
			/* Check for missing comma or tab after species name. */
			input_line_pointer = original_line_pointer
			fix_separator()
			if get_species_name() == FALSE {
				fmt.Fprintf(log_file, "!!! Order ignored:\n")
				fmt.Fprintf(log_file, "!!! %s", original_line)
				fmt.Fprintf(log_file, "!!! Invalid species name.\n")
				return
			}
		}
		recipient_species = &spec_data[g_spec_number-1]

		if species.tech_level[MA] < 25 {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! MA tech level must be at least 25 to do interspecies construction.\n")
			return
		}

		/* Check if we've met this species and make sure it is not an enemy. */
		contact_word_number = (g_spec_number - 1) / 32
		contact_bit_number = (g_spec_number - 1) % 32
		contact_mask = uint32(1) << uint(contact_bit_number)
		if (species.contact[contact_word_number] & contact_mask) == 0 {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! You can't do interspecies construction for a species you haven't met.\n")
			return
		}
		if species.enemy[contact_word_number]&contact_mask != 0 {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! You can't do interspecies construction for an ENEMY.\n")
			return
		}
	}

	/* Get number of items to build. */
	i = get_value()

	if i == 0 {
		goto build_ship
	} /* Not an item. */
	num_items = value
	original_num_items = value

	/* Get class of item. */
	class = get_class_abbr()

	if class != ITEM_CLASS || abbr_index == RM {
		/* Players sometimes accidentally use "MI" for "IU" or "MA" for "AU". */
		if class == TECH_ID && abbr_index == MI {
			abbr_index = IU
		} else if class == TECH_ID && abbr_index == MA {
			abbr_index = AU
		} else {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Invalid item class.\n")
			return
		}
	}
	class = abbr_index

	if interspecies_construction != FALSE {
		if class == PD || class == CU {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! You cannot build CUs or PDs for another species.\n")
			return
		}
	}

	/* Make sure species knows how to build this item. */
	critical_tech = item_critical_tech[class]
	if species.tech_level[critical_tech] < item_tech_requirment[class] {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Insufficient tech level to build item.\n")
		return
	}

	/* Get cost of item. */
	if class == TP {
		/* Terraforming plant. */
		unit_cost = item_cost[class] / species.tech_level[critical_tech]
	} else {
		unit_cost = item_cost[class]
	}

	if num_items == 0 {
		num_items = balance / unit_cost
	}
	if num_items == 0 {
		return
	}

	/* Make sure item count is meaningful. */
	if num_items < 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Meaningless item count.\n")
		return
	}

	/* Make sure there is enough available population. */
	pop_reduction = 0
	if class == CU || class == PD {
		if nampla.pop_units < num_items {
			if original_num_items == 0 {
				num_items = nampla.pop_units
				if num_items == 0 {
					return
				}
			} else {
				if nampla.pop_units > 0 {
					fmt.Fprintf(log_file, "! WARNING: %s", original_line)
					fmt.Fprintf(log_file, "! Insufficient available population units. Substituting %d for %d.\n",
						nampla.pop_units, num_items)
					num_items = nampla.pop_units
				} else {
					fmt.Fprintf(log_file, "!!! Order ignored:\n")
					fmt.Fprintf(log_file, "!!! %s", original_line)
					fmt.Fprintf(log_file, "!!! Insufficient available population units.\n")
					return
				}
			}
		}
		pop_reduction = num_items
	}

	/* Calculate total cost and see if planet has enough money. */
do_cost:
	cost = num_items * unit_cost
	if interspecies_construction != FALSE {
		premium = (cost + 9) / 10
	} else {
		premium = 0
	}

	cost += premium

	if check_bounced(cost) != FALSE {
		if interspecies_construction != FALSE && original_num_items == 0 {
			num_items--
			if num_items < 1 {
				return
			}
			goto do_cost
		}

		max_funds_available = species.econ_units
		if max_funds_available > EU_spending_limit {
			max_funds_available = EU_spending_limit
		}
		max_funds_available += balance

		num_items = max_funds_available / unit_cost
		if interspecies_construction != FALSE {
			num_items -= (num_items + 9) / 10
		}

		if num_items > 0 {
			fmt.Fprintf(log_file, "! WARNING: %s", original_line)
			fmt.Fprintf(log_file, "! Insufficient funds. Substituting %d for %d.\n",
				num_items, original_num_items)
			goto do_cost
		} else {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Insufficient funds to execute order.\n")
			return
		}
	}

	/* Update planet inventory. */
	nampla.item_quantity[class] += num_items
	nampla.pop_units -= pop_reduction

	/* Log what was produced. */
	log_string("    ")
	log_long(num_items)
	log_char(' ')
	log_string(item_name[class])

	if num_items > 1 {
		log_string("s were")
	} else {
		log_string(" was")
	}

	if first_pass != FALSE && class == PD && siege_effectiveness > 0 {
		log_string(" scheduled for production despite the siege.\n")
		return
	} else {
		log_string(" produced")
		if interspecies_construction != FALSE {
			log_string(" for SP ")
			log_string(recipient_species.name)
		}
	}

	if unit_cost != 1 || premium != 0 {
		log_string(" at a cost of ")
		log_long(cost)
	}

	/* Check if planet is under siege and if production of planetary defenses was detected. */
	if class == PD && rnd(100) <= siege_effectiveness {
		log_string(". However, they were detected and destroyed by the besiegers!!!\n")
		nampla.item_quantity[PD] = 0

		/* Make sure we don't notify the same species more than once. */
		for i = 0; i < MAX_SPECIES; i++ {
			already_notified[i] = FALSE
		}

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

			if already_notified[alien_number-1] != FALSE {
				continue
			}

			/* Define a 'detection' transaction. */
			if num_transactions == MAX_TRANSACTIONS {
				fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS!\n\n")
				os.Exit(255)
			}

			n = num_transactions
			num_transactions++
			transaction[n].trans_type = DETECTION_DURING_SIEGE
			transaction[n].value = 3 /* Construction of PDs. */
			transaction[n].name1 = nampla.name
			transaction[n].name3 = species.name
			transaction[n].number3 = alien_number

			already_notified[alien_number-1] = TRUE
		}
		return
	}

	if interspecies_construction == FALSE {
		/* Get destination of transfer, if any. */
		pop_check_needed = FALSE
		temp_nampla = nampla
		found = get_transfer_point()
		destination_nampla = nampla
		nampla = temp_nampla
		if found == FALSE {
			goto done_transfer
		}

		if abbr_type == SHIP_CLASS /* Destination is 'ship'. */ {
			if ship.x != nampla.x || ship.y != nampla.y || ship.z != nampla.z ||
				ship.status == UNDER_CONSTRUCTION {
				goto done_transfer
			}

			if ship.class == TR {
				capacity = (10 + (ship.tonnage / 2)) * ship.tonnage
			} else if ship.class == BA {
				capacity = 10 * ship.tonnage
			} else {
				capacity = ship.tonnage
			}

			for i = 0; i < MAX_ITEMS; i++ {
				capacity -= ship.item_quantity[i] * item_carry_capacity[i]
			}

			n = num_items
			if num_items*item_carry_capacity[class] > capacity {
				num_items = capacity / item_carry_capacity[class]
			}

			ship.item_quantity[class] += num_items
			nampla.item_quantity[class] -= num_items
			log_string(" and ")
			if n > num_items {
				log_long(num_items)
				log_string(" of them ")
			}
			if num_items == 1 {
				log_string("was")
			} else {
				log_string("were")
			}
			log_string(" transferred to ")
			log_string(ship_name(ship))

			if class == CU && num_items > 0 {
				if nampla == nampla_base[0] {
					ship.loading_point = 9999 /* Home planet. */
				} else {
					/* C: ship->loading_point = (nampla - nampla_base). */
					for i = 0; i < species.num_namplas; i++ {
						if nampla_base[i] == nampla {
							ship.loading_point = i
							break
						}
					}
				}
			}
		} else {
			/* Destination is 'destination_nampla'. */
			if destination_nampla.x != nampla.x || destination_nampla.y != nampla.y ||
				destination_nampla.z != nampla.z {
				goto done_transfer
			}

			if nampla.siege_eff != 0 {
				goto done_transfer
			}
			if destination_nampla.siege_eff != 0 {
				goto done_transfer
			}

			destination_nampla.item_quantity[class] += num_items
			nampla.item_quantity[class] -= num_items
			log_string(" and transferred to PL ")
			log_string(destination_nampla.name)
			pop_check_needed = TRUE
		}

	done_transfer:

		log_string(".\n")

		if pop_check_needed != FALSE {
			check_population(destination_nampla)
		}

		return
	}

	log_string(".\n")

	/* Check if recipient species has a nampla at this location. */
	found = FALSE
	unused_nampla_available = FALSE
	for i = 0; i < recipient_species.num_namplas; i++ {
		recipient_nampla = namp_data[g_spec_number-1][i]

		if recipient_nampla.pn == 99 {
			unused_nampla = recipient_nampla
			unused_nampla_available = TRUE
		}

		if recipient_nampla.x != nampla.x {
			continue
		}
		if recipient_nampla.y != nampla.y {
			continue
		}
		if recipient_nampla.z != nampla.z {
			continue
		}
		if recipient_nampla.pn != nampla.pn {
			continue
		}

		found = TRUE
		break
	}

	if found == FALSE {
		/* Add new nampla to database for the recipient species. */
		if unused_nampla_available != FALSE {
			recipient_nampla = unused_nampla
		} else {
			num_new_namplas[species_index]++
			if num_new_namplas[species_index] > NUM_EXTRA_NAMPLAS {
				fmt.Fprintf(os.Stderr, "\n\n\tInsufficient memory for new planet name in do_BUILD_command!\n")
				os.Exit(255)
			}
			/* C: recipient_nampla = namp_data[g_spec_number-1] +
			   recipient_species->num_namplas (headroom); grow the slice. */
			for len(namp_data[g_spec_number-1]) <= recipient_species.num_namplas {
				namp_data[g_spec_number-1] = append(namp_data[g_spec_number-1], &nampla_data_t{})
			}
			recipient_nampla = namp_data[g_spec_number-1][recipient_species.num_namplas]
			recipient_species.num_namplas += 1
			delete_nampla(recipient_nampla) /* Set everything to zero. */
		}

		/* Initialize new nampla. */
		recipient_nampla.name = nampla.name
		recipient_nampla.x = nampla.x
		recipient_nampla.y = nampla.y
		recipient_nampla.z = nampla.z
		recipient_nampla.pn = nampla.pn
		recipient_nampla.planet_index = nampla.planet_index
		recipient_nampla.status = COLONY
	}

	/* Transfer the goods. */
	nampla.item_quantity[class] -= num_items
	recipient_nampla.item_quantity[class] += num_items
	data_modified[g_spec_number-1] = TRUE

	if first_pass != FALSE {
		return
	}

	/* Define transaction so that recipient will be notified. */
	if num_transactions == MAX_TRANSACTIONS {
		fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS!\n\n")
		os.Exit(255)
	}

	n = num_transactions
	num_transactions++
	transaction[n].trans_type = INTERSPECIES_CONSTRUCTION
	transaction[n].donor = species_number
	transaction[n].recipient = g_spec_number
	transaction[n].value = 1 /* Items, not ships. */
	transaction[n].number1 = num_items
	transaction[n].number2 = class
	transaction[n].number3 = cost
	transaction[n].name1 = species.name
	transaction[n].name2 = recipient_nampla.name

	return

build_ship:

	original_line_pointer = input_line_pointer
	if continuing_construction != FALSE {
		found = get_ship()
		if found == FALSE {
			/* Check for missing comma or tab after ship name. */
			input_line_pointer = original_line_pointer
			fix_separator()
			found = get_ship()
		}

		if found != FALSE {
			goto check_ship
		}
		input_line_pointer = original_line_pointer
	}

	class = get_class_abbr()

	if class != SHIP_CLASS || tonnage < 1 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Invalid ship class.\n")
		return
	}
	class = abbr_index

	/* Get ship name. */
	name_length = get_name()
	if name_length < 1 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Invalid ship name.\n")
		return
	}

	/* Search all ships for name. */
	found = FALSE
	unused_ship_available = FALSE
	for ship_index = 0; ship_index < species.num_ships; ship_index++ {
		ship = ship_base[ship_index]

		if ship.pn == 99 {
			unused_ship_available = TRUE
			unused_ship = ship
			continue
		}

		/* Make upper case copy of ship name. */
		upper_ship_name = upcase(ship.name)

		/* Compare names. */
		if upper_ship_name == upper_name {
			found = TRUE
			break
		}
	}

check_ship:

	if found != FALSE {
		/* Check if BUILD was accidentally used instead of CONTINUE. */
		if (ship.status == UNDER_CONSTRUCTION || ship.ship_type == STARBASE) && ship.x == nampla.x &&
			ship.y == nampla.y && ship.z == nampla.z && ship.pn == nampla.pn {
			continuing_construction = TRUE
		}

		if (ship.status != UNDER_CONSTRUCTION && ship.ship_type != STARBASE) || (continuing_construction == FALSE) {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Ship name already in use.\n")
			return
		}

		new_ship = FALSE
	} else {
		/* If CONTINUE command was used, the player probably mis-spelled the name. */
		if continuing_construction != FALSE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Invalid ship name.\n")
			return
		}

		if unused_ship_available != FALSE {
			ship = unused_ship
		} else {
			/* Make sure we have enough memory for new ship. */
			if num_new_ships[species_index] >= NUM_EXTRA_SHIPS {
				if num_new_ships[species_index] == 9999 {
					return
				}

				fmt.Fprintf(log_file, "!!! Order ignored:\n")
				fmt.Fprintf(log_file, "!!! %s", original_line)
				fmt.Fprintf(log_file, "!!! You cannot build more than %d ships per turn!\n", NUM_EXTRA_SHIPS)
				num_new_ships[species_index] = 9999
				return
			}
			new_ship = TRUE
			/* C: ship = ship_base + species->num_ships (headroom
			   allocated by ncalloc); grow the slice instead. */
			for len(ship_data[species_index]) <= species.num_ships {
				ship_data[species_index] = append(ship_data[species_index], &ship_data_t{})
			}
			ship_base = ship_data[species_index]
			ship = ship_base[species.num_ships]
			/* Initialize everything to zero. */
			delete_ship(ship)
		}

		/* Initialize non-zero data for new ship. */
		ship.name = original_name
		ship.x = nampla.x
		ship.y = nampla.y
		ship.z = nampla.z
		ship.pn = nampla.pn
		ship.status = UNDER_CONSTRUCTION
		if class == BA {
			ship.ship_type = STARBASE
			ship.status = IN_ORBIT
		} else if sub_light != FALSE {
			ship.ship_type = SUB_LIGHT
		} else {
			ship.ship_type = FTL
		}
		ship.class = class
		ship.age = -1
		if ship.ship_type != STARBASE {
			ship.tonnage = tonnage
		}
		ship.remaining_cost = ship_cost[class]
		if ship.class == TR {
			ship.remaining_cost = ship_cost[TR] * tonnage
		}
		if ship.ship_type == SUB_LIGHT {
			ship.remaining_cost = (3 * ship.remaining_cost) / 4
		}
		ship.just_jumped = FALSE

		/* Everything else was set to zero in above call to 'delete_ship'. */
	}

	/* Check if amount to spend was specified. */
	cost_given = get_value()
	cost = value
	cost_argument = value

	if cost_given != FALSE {
		if interspecies_construction != FALSE && (ship.ship_type != STARBASE) {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Amount to spend may not be specified.\n")
			return
		}

		if cost == 0 {
			cost = balance
			if ship.ship_type == STARBASE {
				if cost%ship_cost[BA] != 0 {
					cost = ship_cost[BA] * (cost / ship_cost[BA])
				}
			}
			if cost < 1 {
				if new_ship != FALSE {
					delete_ship(ship)
				}
				return
			}
		}

		if cost < 1 {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Amount specified is meaningless.\n")
			if new_ship != FALSE {
				delete_ship(ship)
			}
			return
		}

		if ship.ship_type == STARBASE {
			if cost%ship_cost[BA] != 0 {
				fmt.Fprintf(log_file, "!!! Order ignored:\n")
				fmt.Fprintf(log_file, "!!! %s", original_line)
				fmt.Fprintf(log_file, "!!! Amount spent on starbase must be multiple of %d.\n", ship_cost[BA])
				if new_ship != FALSE {
					delete_ship(ship)
				}
				return
			}
		}
	} else {
		if ship.ship_type == STARBASE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Amount to spend MUST be specified for starbase.\n")
			if new_ship != FALSE {
				delete_ship(ship)
			}
			return
		}

		cost = ship.remaining_cost
	}

	/* Make sure species can build a ship of this size. */
	max_tonnage = species.tech_level[MA] / 2
	if ship.ship_type == STARBASE {
		tonnage_increase = cost / ship_cost[BA]
		tonnage = ship.tonnage + tonnage_increase
		if tonnage > max_tonnage && cost_argument == 0 {
			tonnage_increase = max_tonnage - ship.tonnage
			if tonnage_increase < 1 {
				return
			}
			tonnage = ship.tonnage + tonnage_increase
			cost = tonnage_increase * ship_cost[BA]
		}
	}

	if tonnage > max_tonnage {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Maximum allowable tonnage exceeded.\n")
		if new_ship != FALSE {
			delete_ship(ship)
		}
		return
	}

	/* Make sure species has gravitics technology if this is an FTL ship. */
	if ship.ship_type == FTL && species.tech_level[GV] < 1 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Gravitics tech needed to build FTL ship!\n")
		if new_ship != FALSE {
			delete_ship(ship)
		}
		return
	}

	/* Make sure amount specified is not an overpayment. */
	if ship.ship_type != STARBASE && cost > ship.remaining_cost {
		cost = ship.remaining_cost
	}

	/* Make sure planet has sufficient shipyards. */
	if shipyard_capacity < 1 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Shipyard capacity exceeded!\n")
		if new_ship != FALSE {
			delete_ship(ship)
		}
		return
	}

	/* Make sure there is enough money to pay for it. */
	premium = 0
	if interspecies_construction != FALSE {
		if ship.class == TR || ship.ship_type == STARBASE {
			total_cost = ship_cost[ship.class] * tonnage
		} else {
			total_cost = ship_cost[ship.class]
		}

		if ship.ship_type == SUB_LIGHT {
			total_cost = (3 * total_cost) / 4
		}

		premium = total_cost / 10
		if total_cost%10 != 0 {
			premium++
		}
	}

	if check_bounced(cost+premium) != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Insufficient funds to execute order.\n")
		if new_ship != FALSE {
			delete_ship(ship)
		}
		return
	}

	shipyard_capacity--

	/* Test if this is a starbase and if planet is under siege. */
	if ship.ship_type == STARBASE && siege_effectiveness > 0 {
		log_string("    Your attempt to build ")
		log_string(ship_name(ship))
		log_string(" was detected by the besiegers and the starbase was destroyed!!!\n")

		/* Make sure we don't notify the same species more than once. */
		for i = 0; i < MAX_SPECIES; i++ {
			already_notified[i] = FALSE
		}

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

			if already_notified[alien_number-1] != FALSE {
				continue
			}

			/* Define a 'detection' transaction. */
			if num_transactions == MAX_TRANSACTIONS {
				fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS!\n\n")
				os.Exit(255)
			}

			n = num_transactions
			num_transactions++
			transaction[n].trans_type = DETECTION_DURING_SIEGE
			transaction[n].value = 2 /* Construction of ship/starbase. */
			transaction[n].name1 = nampla.name
			transaction[n].name2 = ship_name(ship)
			transaction[n].name3 = species.name
			transaction[n].number3 = alien_number

			already_notified[alien_number-1] = TRUE
		}

		delete_ship(ship)

		return
	}

	/* Finish up and log results. */
	log_string("    ")
	if ship.ship_type == STARBASE {
		if ship.tonnage == 0 {
			log_string(ship_name(ship))
			log_string(" was constructed")
		} else {
			/* Weighted average. */
			ship.age = ((ship.age * ship.tonnage) - tonnage_increase) / tonnage
			log_string("Size of ")
			log_string(ship_name(ship))
			log_string(" was increased to ")
			log_string(commas(10000 * tonnage))
			log_string(" tons")
		}

		ship.tonnage = tonnage
	} else {
		ship.remaining_cost -= cost
		if ship.remaining_cost == 0 {
			ship.status = ON_SURFACE /* Construction is complete. */
			if continuing_construction != FALSE {
				if first_pass != FALSE && siege_effectiveness > 0 {
					log_string("An attempt will be made to finish construction on ")
				} else {
					log_string("Construction finished on ")
				}
				log_string(ship_name(ship))
				if first_pass != FALSE && siege_effectiveness > 0 {
					log_string(" despite the siege")
				}
			} else {
				if first_pass != FALSE && siege_effectiveness > 0 {
					log_string("An attempt will be made to construct ")
				}
				log_string(ship_name(ship))
				if first_pass != FALSE && siege_effectiveness > 0 {
					log_string(" despite the siege")
				} else {
					log_string(" was constructed")
				}
			}
		} else {
			if continuing_construction != FALSE {
				if first_pass != FALSE && siege_effectiveness > 0 {
					log_string("An attempt will be made to continue construction on ")
				} else {
					log_string("Construction continued on ")
				}
				log_string(ship_name(ship))
				if first_pass != FALSE && siege_effectiveness > 0 {
					log_string(" despite the siege")
				}
			} else {
				if first_pass != FALSE && siege_effectiveness > 0 {
					log_string("An attempt will be made to start construction on ")
				} else {
					log_string("Construction started on ")
				}
				log_string(ship_name(ship))
				if first_pass != FALSE && siege_effectiveness > 0 {
					log_string(" despite the siege")
				}
			}
		}
	}
	log_string(" at a cost of ")
	log_long(cost + premium)

	if interspecies_construction != FALSE {
		log_string(" for SP ")
		log_string(recipient_species.name)
	}

	log_char('.')

	if new_ship != FALSE && (unused_ship_available == FALSE) {
		num_new_ships[species_index]++
		species.num_ships++
	}

	/* Check if planet is under siege and if construction was detected. */
	if first_pass == FALSE && rnd(100) <= siege_effectiveness {
		log_string(" However, the work was detected by the besiegers and the ship was destroyed!!!")

		/* Make sure we don't notify the same species more than once. */
		for i = 0; i < MAX_SPECIES; i++ {
			already_notified[i] = FALSE
		}

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

			if already_notified[alien_number-1] != FALSE {
				continue
			}

			/* Define a 'detection' transaction. */
			if num_transactions == MAX_TRANSACTIONS {
				fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS!\n\n")
				os.Exit(255)
			}

			n = num_transactions
			num_transactions++
			transaction[n].trans_type = DETECTION_DURING_SIEGE
			transaction[n].value = 2 /* Construction of ship/starbase. */
			transaction[n].name1 = nampla.name
			transaction[n].name2 = ship_name(ship)
			transaction[n].name3 = species.name
			transaction[n].number3 = alien_number

			already_notified[alien_number-1] = TRUE
		}

		/* Remove ship from inventory. */
		delete_ship(ship)
	}

	log_char('\n')

	if interspecies_construction == FALSE {
		return
	}

	/* Transfer any cargo on the ship to the planet. */
	cargo_on_board = FALSE
	for i = 0; i < MAX_ITEMS; i++ {
		if ship.item_quantity[i] > 0 {
			nampla.item_quantity[i] += ship.item_quantity[i]
			ship.item_quantity[i] = 0
			cargo_on_board = TRUE
		}
	}
	if cargo_on_board != FALSE {
		log_string("      Forgotten cargo on the ship was first transferred to the planet.\n")
	}

	/* Transfer the ship to the recipient species. */
	unused_ship_available = FALSE
	for i = 0; i < recipient_species.num_ships; i++ {
		recipient_ship = ship_data[g_spec_number-1][i]
		if recipient_ship.pn == 99 {
			unused_ship_available = TRUE
			break
		}
	}

	if unused_ship_available == FALSE {
		/* Make sure we have enough memory for new ship. */
		if num_new_ships[g_spec_number-1] == NUM_EXTRA_SHIPS {
			fmt.Fprintf(os.Stderr, "\n\n\tInsufficient memory for new recipient ship!\n\n")
			os.Exit(255)
		}
		/* C: recipient_ship = ship_data[g_spec_number-1] +
		   recipient_species->num_ships (headroom); grow the slice. */
		for len(ship_data[g_spec_number-1]) <= recipient_species.num_ships {
			ship_data[g_spec_number-1] = append(ship_data[g_spec_number-1], &ship_data_t{})
		}
		recipient_ship = ship_data[g_spec_number-1][recipient_species.num_ships]
		recipient_species.num_ships++
		num_new_ships[g_spec_number-1]++
	}

	/* Copy donor ship to recipient ship. */
	*recipient_ship = *ship

	recipient_ship.status = IN_ORBIT

	data_modified[g_spec_number-1] = TRUE

	/* Delete donor ship. */
	delete_ship(ship)

	if first_pass != FALSE {
		return
	}

	/* Define transaction so that recipient will be notified. */
	if num_transactions == MAX_TRANSACTIONS {
		fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS!\n\n")
		os.Exit(255)
	}

	n = num_transactions
	num_transactions++
	transaction[n].trans_type = INTERSPECIES_CONSTRUCTION
	transaction[n].donor = species_number
	transaction[n].recipient = g_spec_number
	transaction[n].value = 2 /* Ship, not items. */
	transaction[n].number3 = total_cost + premium
	transaction[n].name1 = species.name
	transaction[n].name2 = ship_name(recipient_ship)
}

func do_DEEP_command() {
	/* Get the ship. */
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
			fmt.Fprintf(log_file, "!!! Invalid ship name in ORBIT command.\n")
			return
		}
	}
	if ship.ship_type == STARBASE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! DEEP order may not be given for a starbase.\n")
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
	/* Make sure ship is not salvage of a disbanded colony. */
	if disbanded_ship(ship) != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! This ship is salvage of a disbanded colony!\n")
		return
	}

	/* Move the ship. */
	ship.pn = 0
	ship.status = IN_DEEP_SPACE

	/* Log result. */
	log_string("    ")
	log_string(ship_name(ship))
	log_string(" moved into deep space.\n")
}

func do_DESTROY_command() {
	var found int
	/* Get the ship. */
	correct_spelling_required = TRUE
	found = get_ship()
	if found == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid ship or starbase name in DESTROY command.\n")
		return
	}
	/* Log result. */
	log_string("    ")
	log_string(ship_name(ship))
	if first_pass != FALSE {
		log_string(" will be destroyed.\n")
		return
	}
	log_string(" was destroyed.\n")
	delete_ship(ship)
}

func do_DEVELOP_command() {
	var i, num_CUs, num_AUs, num_IUs, more_args, load_transport int
	var capacity, resort_colony, mining_colony, production_penalty int
	var CUs_only int
	var c byte
	var original_line_pointer, tp int
	var n, ni, na, amount_to_spend, max_funds_available int
	var ls_needed, raw_material_units, production_capacity int
	var colony_production, ib, ab, md, denom, reb, specified_max int
	var colony_planet, home_planet *planet_data_t
	var temp_nampla, colony_nampla *nampla_data_t

	/* Check if this order was preceded by a PRODUCTION order. */
	if doing_production == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Missing PRODUCTION order!\n")
		return
	}

	/* Get default spending limit. */
	max_funds_available = species.econ_units
	if max_funds_available > EU_spending_limit {
		max_funds_available = EU_spending_limit
	}
	max_funds_available += balance

	/* Get specified spending limit, if any. */
	specified_max = -1
	if get_value() != 0 {
		if value == 0 {
			max_funds_available = balance
		} else if value > 0 {
			specified_max = value
			if value <= max_funds_available {
				max_funds_available = value
			} else {
				fmt.Fprintf(log_file, "! WARNING: %s", input_line)
				fmt.Fprintf(log_file, "! Insufficient funds. Substituting %d for %d.\n", max_funds_available, value)
				if max_funds_available == 0 {
					return
				}
			}
		} else {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! Invalid spending limit.\n")
			return
		}
	}

	/* See if there are any more arguments. */
	tp = input_line_pointer
	more_args = FALSE
	for {
		c = at(input_line, tp)
		tp++
		if c == 0 {
			break
		}
		if c == ';' || c == '\n' {
			break
		}
		if c == ' ' || c == '\t' {
			continue
		}
		more_args = TRUE
		break
	}

	if more_args == FALSE {
		/* Make sure planet is not a healthy home planet. */
		if nampla.status&HOME_PLANET != 0 {
			reb = species.hp_original_base - (nampla.mi_base + nampla.ma_base)
			if reb > 0 {
				/* Home planet is recovering from bombing. */
				if reb < max_funds_available {
					max_funds_available = reb
				}
			} else {
				fmt.Fprintf(log_file, "!!! Order ignored:\n")
				fmt.Fprintf(log_file, "!!! %s", input_line)
				fmt.Fprintf(log_file, "!!! You can only DEVELOP a home planet if it is recovering from bombing.\n")
				return
			}
		}

		/* No arguments. Order is for this planet. */
		num_CUs = nampla.pop_units
		if 2*num_CUs > max_funds_available {
			num_CUs = max_funds_available / 2
		}
		if num_CUs <= 0 {
			return
		}

		colony_planet = planet_base[nampla.planet_index]
		ib = nampla.mi_base + nampla.IUs_to_install
		ab = nampla.ma_base + nampla.AUs_to_install
		md = colony_planet.mining_difficulty

		denom = 100 + md
		num_AUs =
			(100*(num_CUs+ib) - (md * ab) + denom/2) / denom
		num_IUs = num_CUs - num_AUs

		if num_IUs < 0 {
			num_AUs = num_CUs
			num_IUs = 0
		}
		if num_AUs < 0 {
			num_IUs = num_CUs
			num_AUs = 0
		}

		amount_to_spend = num_CUs + num_AUs + num_IUs

		if check_bounced(amount_to_spend) != FALSE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Internal error. Please notify GM!\n")
			return
		}

		nampla.pop_units -= num_CUs
		nampla.item_quantity[CU] += num_CUs
		nampla.item_quantity[IU] += num_IUs
		nampla.item_quantity[AU] += num_AUs

		nampla.auto_IUs += num_IUs
		nampla.auto_AUs += num_AUs

		start_dev_log(num_CUs, num_IUs, num_AUs)
		log_string(".\n")

		check_population(nampla)

		return
	}

	/* Get the planet to be developed. */
	temp_nampla = nampla
	original_line_pointer = input_line_pointer
	i = get_location()
	if i == FALSE || nampla == nil {
		/* Check for missing comma or tab after source name. */
		input_line_pointer = original_line_pointer
		fix_separator()
		i = get_location()
	}
	colony_nampla = nampla
	nampla = temp_nampla
	if i == FALSE || colony_nampla == nil {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid planet name in DEVELOP command.\n")
		return
	}

	/* Make sure planet is not a healthy home planet. */
	if colony_nampla.status&HOME_PLANET != 0 {
		reb = species.hp_original_base - (colony_nampla.mi_base + colony_nampla.ma_base)
		if reb > 0 {
			/* Home planet is recovering from bombing. */
			if reb < max_funds_available {
				max_funds_available = reb
			}
		} else {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! You can only DEVELOP a home planet if it is recovering from bombing.\n")
			return
		}
	}

	/* Determine if its a mining or resort colony, and if it can afford to
	   build its own IUs and AUs. Note that we cannot use nampla->status
	   because it is not correctly set until the Finish program is run. */

	home_planet = planet_base[nampla_base[0].planet_index]
	colony_planet = planet_base[colony_nampla.planet_index]
	ls_needed = life_support_needed(species, home_planet, colony_planet)

	ni = colony_nampla.mi_base + colony_nampla.IUs_to_install
	na = colony_nampla.ma_base + colony_nampla.AUs_to_install

	if ni > 0 && na == 0 {
		colony_production = 0
		mining_colony = TRUE
		resort_colony = FALSE
	} else if na > 0 && ni == 0 && ls_needed <= 6 && colony_planet.gravity <= home_planet.gravity {
		colony_production = 0
		resort_colony = TRUE
		mining_colony = FALSE
	} else {
		mining_colony = FALSE
		resort_colony = FALSE

		raw_material_units = (10 * species.tech_level[MI] * ni) / colony_planet.mining_difficulty
		production_capacity = (species.tech_level[MA] * na) / 10

		if ls_needed == 0 {
			production_penalty = 0
		} else {
			production_penalty = (100 * ls_needed) / species.tech_level[LS]
		}

		raw_material_units -= (production_penalty * raw_material_units) / 100
		production_capacity -= (production_penalty * production_capacity) / 100

		if production_capacity > raw_material_units {
			colony_production = raw_material_units
		} else {
			colony_production = production_capacity
		}

		colony_production -= colony_nampla.IUs_needed + colony_nampla.AUs_needed
		/* In case there is more than one DEVELOP order for this colony. */
	}

	/* See if there are more arguments. */
	tp = input_line_pointer
	more_args = FALSE
	for {
		c = at(input_line, tp)
		tp++
		if c == 0 {
			break
		}
		if c == ';' || c == '\n' {
			break
		}
		if c == ' ' || c == '\t' {
			continue
		}
		more_args = TRUE
		break
	}

	if more_args != FALSE {
		load_transport = TRUE

		/* Get the ship to receive the cargo. */
		if get_ship() == FALSE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Ship to be loaded does not exist!\n")
			return
		}

		if ship.class == TR {
			capacity = (10 + (ship.tonnage / 2)) * ship.tonnage
		} else if ship.class == BA {
			capacity = 10 * ship.tonnage
		} else {
			capacity = ship.tonnage
		}

		for i = 0; i < MAX_ITEMS; i++ {
			capacity -= ship.item_quantity[i] * item_carry_capacity[i]
		}

		if capacity <= 0 {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! %s was already full and could take no more cargo!\n", ship_name(ship))
			return
		}

		if capacity > max_funds_available {
			capacity = max_funds_available
			if max_funds_available != specified_max {
				fmt.Fprintf(log_file, "! WARNING: %s", input_line)
				fmt.Fprintf(log_file, "! Insufficient funds to completely fill %s!\n", ship_name(ship))
				fmt.Fprintf(log_file, "! Will use all remaining funds (= %d).\n", capacity)
			}
		}
	} else {
		load_transport = FALSE

		/* No more arguments. Order is for a colony in the same sector as the producing planet. */
		if nampla.x != colony_nampla.x || nampla.y != colony_nampla.y || nampla.z != colony_nampla.z {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Colony and producing planet are not in the same sector.\n")
			return
		}

		num_CUs = nampla.pop_units
		if 2*num_CUs > max_funds_available {
			num_CUs = max_funds_available / 2
		}
	}

	CUs_only = FALSE
	if mining_colony != FALSE {
		if load_transport != FALSE {
			num_CUs = capacity / 2
			if num_CUs > nampla.pop_units {
				fmt.Fprintf(log_file, "! WARNING: %s", input_line)
				fmt.Fprintf(log_file, "! Insufficient available population! %d CUs are needed", num_CUs)
				num_CUs = nampla.pop_units
				fmt.Fprintf(log_file, " to fill ship but only %d can be built.\n", num_CUs)
			}
		}

		num_AUs = 0
		num_IUs = num_CUs
	} else if resort_colony != FALSE {
		if load_transport != FALSE {
			num_CUs = capacity / 2
			if num_CUs > nampla.pop_units {
				fmt.Fprintf(log_file, "! WARNING: %s", input_line)
				fmt.Fprintf(log_file, "! Insufficient available population! %d CUs are needed", num_CUs)
				num_CUs = nampla.pop_units
				fmt.Fprintf(log_file, " to fill ship but only %d can be built.\n", num_CUs)
			}
		}

		num_IUs = 0
		num_AUs = num_CUs
	} else {
		if load_transport != FALSE {
			if colony_production >= capacity {
				/* Colony can build its own IUs and AUs. */
				num_CUs = capacity
				CUs_only = TRUE
			} else {
				/* Build IUs and AUs for the colony. */
				num_CUs = capacity / 2
			}

			if num_CUs > nampla.pop_units {
				fmt.Fprintf(log_file, "! WARNING: %s", input_line)
				fmt.Fprintf(log_file, "! Insufficient available population! %d CUs are needed", num_CUs)
				num_CUs = nampla.pop_units
				fmt.Fprintf(log_file, " to fill ship, but\n!   only %d can be built.\n", num_CUs)
			}
		}

		colony_planet = planet_base[colony_nampla.planet_index]

		i = 100 + colony_planet.mining_difficulty
		num_AUs = ((100 * num_CUs) + (i+1)/2) / i
		num_IUs = num_CUs - num_AUs
	}

	if num_CUs <= 0 {
		return
	}

	/* Make sure there's enough money to pay for it all. */
	if load_transport != FALSE && CUs_only != FALSE {
		amount_to_spend = num_CUs
	} else {
		amount_to_spend = num_CUs + num_IUs + num_AUs
	}

	if check_bounced(amount_to_spend) != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Internal error. Notify GM!\n")
		return
	}

	/* Start logging what happened. */
	if load_transport != FALSE && CUs_only != FALSE {
		start_dev_log(num_CUs, 0, 0)
	} else {
		start_dev_log(num_CUs, num_IUs, num_AUs)
	}

	log_string(" for PL ")
	log_string(colony_nampla.name)

	nampla.pop_units -= num_CUs

	if load_transport != FALSE {
		if CUs_only != FALSE {
			colony_nampla.IUs_needed += num_IUs
			colony_nampla.AUs_needed += num_AUs
		}

		if nampla.x != ship.x || nampla.y != ship.y || nampla.z != ship.z {
			nampla.item_quantity[CU] += num_CUs
			if CUs_only == FALSE {
				nampla.item_quantity[IU] += num_IUs
				nampla.item_quantity[AU] += num_AUs
			}

			log_string(" but will remain on the planet's surface because ")
			log_string(ship_name(ship))
			log_string(" is not in the same sector.")
		} else {
			ship.item_quantity[CU] += num_CUs
			if CUs_only == FALSE {
				ship.item_quantity[IU] += num_IUs
				ship.item_quantity[AU] += num_AUs
			}

			/* C: n = colony_nampla - nampla_base. */
			n = 0
			for i = 0; i < species.num_namplas; i++ {
				if nampla_base[i] == colony_nampla {
					n = i
					break
				}
			}
			if n == 0 {
				/* Home planet. */
				n = 9999
			}
			ship.unloading_point = n

			/* C: n = nampla - nampla_base. */
			n = 0
			for i = 0; i < species.num_namplas; i++ {
				if nampla_base[i] == nampla {
					n = i
					break
				}
			}
			if n == 0 {
				/* Home planet. */
				n = 9999
			}
			ship.loading_point = n

			log_string(" and transferred to ")
			log_string(ship_name(ship))
		}
	} else {
		colony_nampla.item_quantity[CU] += num_CUs
		colony_nampla.item_quantity[IU] += num_IUs
		colony_nampla.item_quantity[AU] += num_AUs

		colony_nampla.auto_IUs += num_IUs
		colony_nampla.auto_AUs += num_AUs

		log_string(" and transferred to PL ")
		log_string(colony_nampla.name)

		check_population(colony_nampla)
	}

	log_string(".\n")
}

func do_DISBAND_command() {
	/* Get the planet. */
	found := get_location()
	if found == FALSE || nampla == nil {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid planet name in DISBAND command.\n")
		return
	}
	/* Make sure planet is not the home planet. */
	if nampla.status&HOME_PLANET != 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! You cannot disband your home planet!\n")
		return
	}
	/* Make sure planet is not under siege. */
	if nampla.siege_eff != 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! You cannot disband a planet that is under siege!\n")
		return
	}
	/* Mark the colony as "disbanded" and convert mining and manufacturing base to CUs, IUs, and AUs. */
	nampla.status |= DISBANDED_COLONY
	nampla.item_quantity[CU] += nampla.mi_base + nampla.ma_base
	nampla.item_quantity[IU] += nampla.mi_base / 2
	nampla.item_quantity[AU] += nampla.ma_base / 2
	nampla.mi_base = 0
	nampla.ma_base = 0
	/* Log the event. */
	log_string("    The colony on PL ")
	log_string(nampla.name)
	log_string(" was ordered to disband.\n")
}

func do_ENEMY_command() {
	var i, array_index, bit_number int
	var bit_mask uint32
	/* See if declaration is for all species. */
	if get_value() != 0 {
		bit_mask = 0
		for i = 0; i < NUM_CONTACT_WORDS; i++ {
			species.enemy[i] = ^bit_mask /* Set all enemy bits. */
			species.ally[i] = bit_mask   /* Clear all ally bits. */
		}
	} else {
		/* Get name of species that is being declared an enemy. */
		if get_species_name() == FALSE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! Invalid or missing argument in ENEMY command.\n")
			return
		}
		/* Get array index and bit mask. */
		array_index = (g_spec_number - 1) / 32
		bit_number = (g_spec_number - 1) % 32
		bit_mask = uint32(1) << uint(bit_number)
		/* Set/clear the appropriate bit. */
		species.enemy[array_index] |= bit_mask /* Set enemy bit. */
		species.ally[array_index] &^= bit_mask /* Clear ally bit. */
	}
	/* Log the result. */
	log_string("    Enmity was declared towards ")
	if bit_mask == 0 {
		log_string("ALL species")
	} else {
		log_string("SP ")
		log_string(g_spec_name)
	}
	log_string(".\n")
}

func do_ESTIMATE_command() {
	var i, max_error, contact_word_number, contact_bit_number int
	var estimate [6]int
	var cost int
	var contact_mask uint32
	var alien *species_data_t

	/* Check if this order was preceded by a PRODUCTION order. */
	if doing_production == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Missing PRODUCTION order!\n")
		return
	}

	/* Get name of alien species. */
	if get_species_name() == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid species name in ESTIMATE command.\n")
		return
	}

	/* Check if we've met this species. */
	contact_word_number = (g_spec_number - 1) / 32
	contact_bit_number = (g_spec_number - 1) % 32
	contact_mask = uint32(1) << uint(contact_bit_number)
	if (species.contact[contact_word_number] & contact_mask) == 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! You can't do an estimate of a species you haven't met.\n")
		return
	}

	/* Check if sufficient funds are available. */
	cost = 25
	if check_bounced(cost) != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Insufficient funds to execute order.\n")
		return
	}

	/* Log the result. */
	if first_pass != FALSE {
		log_string("    An estimate of the technology of SP ")
		log_string(g_spec_name)
		log_string(" was made at a cost of ")
		log_long(cost)
		log_string(".\n")
		return
	}

	/* Make the estimates. */
	alien = &spec_data[g_spec_number-1]
	for i = 0; i < 6; i++ {
		max_error = alien.tech_level[i] - species.tech_level[i]
		if max_error < 1 {
			max_error = 1
		}
		estimate[i] = alien.tech_level[i] + rnd((2*max_error)+1) - (max_error + 1)
		if alien.tech_level[i] == 0 {
			estimate[i] = 0
		}
		if estimate[i] < 0 {
			estimate[i] = 0
		}
	}

	log_string("    Estimate of the technology of SP ")
	log_string(alien.name)
	log_string(" (government name '")
	log_string(alien.govt_name)
	log_string("', government type '")
	log_string(alien.govt_type)
	log_string("'):\n      MI = ")
	log_int(estimate[MI])
	log_string(", MA = ")
	log_int(estimate[MA])
	log_string(", ML = ")
	log_int(estimate[ML])
	log_string(", GV = ")
	log_int(estimate[GV])
	log_string(", LS = ")
	log_int(estimate[LS])
	log_string(", BI = ")
	log_int(estimate[BI])
	log_string(".\n")
}

func do_HIDE_command() {
	var cost int

	/* Check if this order was preceded by a PRODUCTION order. */
	if doing_production == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Missing PRODUCTION order!\n")
		return
	}

	/* Make sure this is not a mining colony or home planet. */
	if nampla.status&HOME_PLANET != 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! You may not HIDE a home planet.\n")
		return
	}
	if nampla.status&RESORT_COLONY != 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! You may not HIDE a resort colony.\n")
		return
	}

	/* Check if planet is under siege. */
	if nampla.siege_eff != 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Besieged planet cannot HIDE!\n")
		return
	}

	/* Check if sufficient funds are available. */
	cost = (nampla.mi_base + nampla.ma_base) / 10
	if nampla.status&MINING_COLONY != 0 {
		if cost > species.econ_units {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! Mining colony does not have sufficient EUs to hide.\n")
			return
		} else {
			species.econ_units -= cost
		}
	} else if check_bounced(cost) != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Insufficient funds to execute order.\n")
		return
	}

	/* Set 'hiding' flag. */
	nampla.hiding = TRUE

	/* Log transaction. */
	log_string("    Spent ")
	log_long(cost)
	log_string(" hiding this colony.\n")
}

func do_INSTALL_command() {
	var item_class, item_count, num_available, do_all_units, recovering_home_planet, alien_index int
	var n, reb int
	var alien_home_nampla *nampla_data_t
	/* Get number of items to install. */
	if get_value() != 0 {
		do_all_units = FALSE
	} else {
		do_all_units = TRUE
		item_count = 0
		item_class = IU
		goto get_planet
	}
	/* Make sure value is meaningful. */
	if value < 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid item count in INSTALL command.\n")
		return
	}
	item_count = value
	/* Get class of item. */
	item_class = get_class_abbr()
	if item_class != ITEM_CLASS || (abbr_index != IU && abbr_index != AU) {
		/* Players sometimes accidentally use "MI" for "IU"
		   or "MA" for "AU". */
		if item_class == TECH_ID && abbr_index == MI {
			abbr_index = IU
		} else if item_class == TECH_ID && abbr_index == MA {
			abbr_index = AU
		} else {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! Invalid item class!\n")
			return
		}
	}
	item_class = abbr_index

get_planet:

	/* Get planet where items are to be installed. */
	if get_location() == FALSE || nampla == nil {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid planet name in INSTALL command.\n")
		return
	}
	/* Make sure this is not someone else's populated homeworld. */
	for alien_index = 0; alien_index < galaxy.num_species; alien_index++ {
		if species_number == alien_index+1 {
			continue
		}
		if data_in_memory[alien_index] == FALSE {
			continue
		}
		alien_home_nampla = namp_data[alien_index][0]
		if alien_home_nampla.x != nampla.x {
			continue
		}
		if alien_home_nampla.y != nampla.y {
			continue
		}
		if alien_home_nampla.z != nampla.z {
			continue
		}
		if alien_home_nampla.pn != nampla.pn {
			continue
		}
		if (alien_home_nampla.status & POPULATED) == 0 {
			continue
		}
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! You may not colonize someone else's populated home planet!\n")
		return
	}
	/* Make sure it's not a healthy home planet. */
	recovering_home_planet = FALSE
	if nampla.status&HOME_PLANET != 0 {
		n = nampla.mi_base + nampla.ma_base + nampla.IUs_to_install + nampla.AUs_to_install
		reb = species.hp_original_base - n
		if reb > 0 {
			recovering_home_planet = TRUE /* HP was bombed. */
		} else {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! Installation not allowed on a healthy home planet!\n")
			return
		}
	}

check_items:

	/* Make sure planet has the specified items. */
	if item_count == 0 {
		item_count = nampla.item_quantity[item_class]
		if nampla.item_quantity[CU] < item_count {
			item_count = nampla.item_quantity[CU]
		}
		if item_count == 0 {
			if do_all_units != FALSE {
				item_count = 0
				item_class = AU
				do_all_units = FALSE
				goto check_items
			} else {
				return
			}
		}
	} else if nampla.item_quantity[item_class] < item_count {
		fmt.Fprintf(log_file, "! WARNING: %s", input_line)
		fmt.Fprintf(log_file, "! Planet does not have %d %ss. Substituting 0 for %d!\n", item_count, item_abbr[item_class],
			item_count)
		item_count = 0
		goto check_items
	}
	if recovering_home_planet != FALSE {
		if item_count > reb {
			item_count = reb
		}
		reb -= item_count
	}
	/* Make sure planet has enough colonist units. */
	num_available = nampla.item_quantity[CU]
	if num_available < item_count {
		if num_available > 0 {
			fmt.Fprintf(log_file, "! WARNING: %s", input_line)
			fmt.Fprintf(log_file, "! Planet does not have %d CUs. Substituting %d for %d!\n",
				item_count, num_available, item_count)
			item_count = num_available
		} else {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! No colonist units on planet for installation.\n")
			return
		}
	}
	/* Start the installation. */
	nampla.item_quantity[CU] -= item_count
	nampla.item_quantity[item_class] -= item_count
	if item_class == IU {
		nampla.IUs_to_install += item_count
	} else {
		nampla.AUs_to_install += item_count
	}
	/* Log result. */
	log_string("    Installation of ")
	log_int(item_count)
	log_char(' ')
	log_string(item_name[item_class])
	if item_count != 1 {
		log_char('s')
	}
	log_string(" began on PL ")
	log_string(nampla.name)
	log_string(".\n")
	if do_all_units != FALSE {
		item_count = 0
		item_class = AU
		do_all_units = FALSE
		goto check_items
	}
	check_population(nampla)
}

func do_INTERCEPT_command() {
	var i, status int
	var cost int

	/* Check if this order was preceded by a PRODUCTION order. */
	if doing_production == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Missing PRODUCTION order!\n")
		return
	}

	/* Get amount to spend. */
	status = get_value()
	if status == 0 || value < 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid or missing amount.\n")
		return
	}
	if value == 0 {
		value = balance
	}
	if value == 0 {
		return
	}
	cost = value

	/* Check if planet is under siege. */
	if nampla.siege_eff != 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Besieged planet cannot INTERCEPT!\n")
		return
	}

	/* Check if sufficient funds are available. */
	if check_bounced(cost) != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Insufficient funds to execute order.\n")
		return
	}

	log_string("    Preparations were made for an interception at a cost of ")
	log_long(cost)
	log_string(".\n")

	if first_pass != FALSE {
		return
	}

	/* Allocate funds. */
	for i = 0; i < num_intercepts; i++ {
		if nampla.x != intercept[i].x {
			continue
		}
		if nampla.y != intercept[i].y {
			continue
		}
		if nampla.z != intercept[i].z {
			continue
		}

		/* This interception was started by another planet in the same star system. */
		intercept[i].amount_spent += cost
		return
	}

	if num_intercepts == MAX_INTERCEPTS {
		fmt.Fprintf(os.Stderr, "\n\tMAX_INTERCEPTS exceeded in do_JUMP_command!\n\n")
		os.Exit(255)
	}

	intercept[num_intercepts].x = nampla.x
	intercept[num_intercepts].y = nampla.y
	intercept[num_intercepts].z = nampla.z
	intercept[num_intercepts].amount_spent = cost

	num_intercepts++
}
