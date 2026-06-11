package game

// Port of do.c, part 3 of 3 (lines 3908-6128): the RECYCLE, REPAIR,
// RESEARCH, SCAN, SEND, SHIPYARD, TEACH, TECH, TELESCOPE, TERRAFORM,
// TRANSFER, UNLOAD, UPGRADE, VISITED, and WORMHOLE order handlers.

import (
	"fmt"
	"os"
)

func do_RECYCLE_command() {
	var i, class, cargo int
	var recycle_value, original_cost, units_available int

	/* Check if this order was preceded by a PRODUCTION order. */
	if doing_production == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Missing PRODUCTION order!\n")
		return
	}

	/* Get number of items to recycle. */
	i = get_value()

	if i == 0 {
		goto recycle_ship
	} /* Not an item. */

	/* Get class of item. */
	class = get_class_abbr()

	if class != ITEM_CLASS {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid item class in RECYCLE command.\n")
		return
	}
	class = abbr_index

	/* Make sure value is meaningful. */
	if value == 0 {
		value = nampla.item_quantity[class]
	}
	if value == 0 {
		return
	}
	if value < 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid item count in RECYCLE command.\n")
		return
	}

	/* Make sure that items exist. */
	units_available = nampla.item_quantity[class]
	if value > units_available {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Attempt to recycle more items than are available.\n")
		return
	}

	/* Determine recycle value. */
	if class == TP {
		recycle_value = (value * item_cost[class]) / (2 * species.tech_level[BI])
	} else if class == RM {
		recycle_value = value / 5
	} else {
		recycle_value = (value * item_cost[class]) / 2
	}

	/* Update inventories. */
	nampla.item_quantity[class] -= value
	if class == PD || class == CU {
		nampla.pop_units += value
	}
	species.econ_units += recycle_value
	if nampla.status&HOME_PLANET != 0 {
		EU_spending_limit += recycle_value
	}

	/* Log what was recycled. */
	log_string("    ")
	log_long(value)
	log_char(' ')
	log_string(item_name[class])

	if value > 1 {
		log_string("s were")
	} else {
		log_string(" was")
	}

	log_string(" recycled, generating ")
	log_long(recycle_value)
	log_string(" economic units.\n")

	return

recycle_ship:

	correct_spelling_required = TRUE
	if get_ship() == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Ship to be recycled does not exist.\n")
		return
	}

	/* Make sure it didn't just jump. */
	if ship.just_jumped != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Ship just jumped and is still in transit.\n")
		return
	}

	/* Make sure item is at producing planet. */
	if ship.x != nampla.x || ship.y != nampla.y || ship.z != nampla.z || ship.pn != nampla.pn {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Ship is not at the production planet.\n")
		return
	}

	/* Calculate recycled value. */
	if ship.class == TR || ship.ship_type == STARBASE {
		original_cost = ship_cost[ship.class] * ship.tonnage
	} else {
		original_cost = ship_cost[ship.class]
	}

	if ship.ship_type == SUB_LIGHT {
		original_cost = (3 * original_cost) / 4
	}

	if ship.status == UNDER_CONSTRUCTION {
		recycle_value = (original_cost - ship.remaining_cost) / 2
	} else {
		recycle_value = (3 * original_cost * (60 - ship.age)) / 200
	}

	species.econ_units += recycle_value
	if nampla.status&HOME_PLANET != 0 {
		EU_spending_limit += recycle_value
	}

	/* Log what was recycled. */
	log_string("    ")
	log_string(ship_name(ship))
	log_string(" was recycled, generating ")
	log_long(recycle_value)
	log_string(" economic units")

	/* Transfer cargo, if any, from ship to planet. */
	cargo = FALSE
	for i = 0; i < MAX_ITEMS; i++ {
		if ship.item_quantity[i] > 0 {
			nampla.item_quantity[i] += ship.item_quantity[i]
			cargo = TRUE
		}
	}

	if cargo != FALSE {
		log_string(". Cargo onboard ")
		log_string(ship_name(ship))
		log_string(" was first transferred to PL ")
		log_string(nampla.name)
	}

	log_string(".\n")

	/* Remove ship from inventory. */
	delete_ship(ship)
}

func do_REPAIR_command() {
	var i, n, x, y, z, age_reduction, num_dr_units int
	var total_dr_units, dr_units_used, max_age, desired_age int
	var original_line_pointer int
	var damaged_ship *ship_data_t

	/* See if this is a "pool" repair. */
	if get_value() != FALSE {
		x = value
		get_value()
		y = value
		get_value()
		z = value
		if get_value() != FALSE {
			desired_age = value
		} else {
			desired_age = 0
		}
		goto pool_repair
	}

	/* Get the ship to be repaired. */
	original_line_pointer = input_line_pointer
	if get_ship() == FALSE {
		/* Check for missing comma or tab after ship name. */
		input_line_pointer = original_line_pointer
		fix_separator()
		if get_ship() == FALSE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Ship to be repaired does not exist.\n")
			return
		}
	}

	if ship.status == UNDER_CONSTRUCTION {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Item to be repaired is still under construction.\n")
		return
	}

	if ship.age < 1 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Ship or starbase is too new to repair.\n")
		return
	}

	/* Get number of damage repair units to use. */
	if get_value() != FALSE {
		if value == 0 {
			num_dr_units = ship.item_quantity[DR]
		} else {
			num_dr_units = value
		}

		age_reduction = (16 * num_dr_units) / ship.tonnage
		if age_reduction > ship.age {
			age_reduction = ship.age
			n = age_reduction * ship.tonnage
			num_dr_units = (n + 15) / 16
		}
	} else {
		age_reduction = ship.age
		n = age_reduction * ship.tonnage
		num_dr_units = (n + 15) / 16
	}

	/* Check if sufficient units are available. */
	if num_dr_units > ship.item_quantity[DR] {
		if ship.item_quantity[DR] == 0 {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Ship does not have any DRs!\n")
			return
		}
		fmt.Fprintf(log_file, "! WARNING: %s", original_line)
		fmt.Fprintf(log_file, "! Ship does not have %d DRs. Substituting %d for %d.\n", num_dr_units,
			ship.item_quantity[DR], num_dr_units)
		num_dr_units = ship.item_quantity[DR]
	}

	/* Check if repair will have any effect. */
	age_reduction = (16 * num_dr_units) / ship.tonnage
	if age_reduction < 1 {
		if value == 0 {
			return
		}
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! %d DRs is not enough to do a repair.\n", num_dr_units)
		return
	}

	/* Log what was repaired. */
	log_string("    ")
	log_string(ship_name(ship))
	log_string(" was repaired using ")
	log_int(num_dr_units)
	log_char(' ')
	log_string(item_name[DR])
	if num_dr_units != 1 {
		log_char('s')
	}
	log_string(". Age went from ")
	log_int(ship.age)
	log_string(" to ")
	ship.age -= age_reduction
	if ship.age < 0 {
		ship.age = 0
	}
	ship.item_quantity[DR] -= num_dr_units
	log_int(ship.age)
	log_string(".\n")

	return

pool_repair:

	/* Get total number of DR units available. */
	total_dr_units = 0
	for i = 0; i < species.num_ships; i++ {
		ship = ship_base[i]
		if ship.pn == 99 {
			continue
		}
		if ship.x != x {
			continue
		}
		if ship.y != y {
			continue
		}
		if ship.z != z {
			continue
		}
		total_dr_units += ship.item_quantity[DR]
		ship.special = 0
	}

	/* Repair ships, starting with the most heavily damaged. */
	dr_units_used = 0
	for total_dr_units > 0 {
		/* Find most heavily damaged ship. */
		max_age = 0
		for i = 0; i < species.num_ships; i++ {
			ship = ship_base[i]
			if ship.pn == 99 {
				continue
			}
			if ship.x != x {
				continue
			}
			if ship.y != y {
				continue
			}
			if ship.z != z {
				continue
			}
			if ship.special != 0 {
				continue
			}
			if ship.status == UNDER_CONSTRUCTION {
				continue
			}
			n = ship.age
			if n > max_age {
				max_age = n
				damaged_ship = ship
			}
		}

		if max_age == 0 {
			break
		}

		damaged_ship.special = 99

		age_reduction = max_age - desired_age
		n = age_reduction * damaged_ship.tonnage
		num_dr_units = (n + 15) / 16

		if num_dr_units > total_dr_units {
			num_dr_units = total_dr_units
			age_reduction = (16 * num_dr_units) / damaged_ship.tonnage
		}

		if age_reduction < 1 {
			continue
		} /* This ship is too big. */

		log_string("    ")
		log_string(ship_name(damaged_ship))
		log_string(" was repaired using ")
		log_int(num_dr_units)
		log_char(' ')
		log_string(item_name[DR])
		if num_dr_units != 1 {
			log_char('s')
		}
		log_string(". Age went from ")
		log_int(damaged_ship.age)
		log_string(" to ")
		damaged_ship.age -= age_reduction
		if damaged_ship.age < 0 {
			damaged_ship.age = 0
		}
		log_int(damaged_ship.age)
		log_string(".\n")

		total_dr_units -= num_dr_units
		dr_units_used += num_dr_units
	}

	if dr_units_used == 0 {
		return
	}

	/* Subtract units used from ships at the location. */
	for i = 0; i < species.num_ships; i++ {
		ship = ship_base[i]
		if ship.pn == 99 {
			continue
		}
		if ship.x != x {
			continue
		}
		if ship.y != y {
			continue
		}
		if ship.z != z {
			continue
		}

		n = ship.item_quantity[DR]
		if n < 1 {
			continue
		}
		if n > dr_units_used {
			n = dr_units_used
		}

		ship.item_quantity[DR] -= n
		dr_units_used -= n

		if dr_units_used == 0 {
			break
		}
	}
}

func do_RESEARCH_command() {
	var status, tech, initial_level, current_level, need_amount_to_spend int
	var cost, amount_spent, cost_for_one_level, funds_remaining, max_funds_available int

	/* Check if this order was preceded by a PRODUCTION order. */
	if doing_production == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Missing PRODUCTION order!\n")
		return
	}

	/* Get amount to spend. */
	status = get_value()
	if status == 0 { /* Sometimes players reverse the arguments. */
		need_amount_to_spend = TRUE
	} else {
		need_amount_to_spend = FALSE
	}

	/* Get technology. */
	if get_class_abbr() != TECH_ID {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid or missing technology.\n")
		return
	}
	tech = abbr_index

	if species.tech_knowledge[tech] == 0 && sp_tech_level[tech] == 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Zero level can only be raised via TECH or TEACH.\n")
		return
	}

	/* Get amount to spend if it was not obtained above. */
	if need_amount_to_spend != FALSE {
		status = get_value()
	}

	if status == 0 || value < 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid or missing amount to spend!\n")
		return
	}

do_cost:

	if value == 0 {
		value = balance
	}
	if value == 0 {
		return
	}
	cost = value

	/* Check if sufficient funds are available. */
	if check_bounced(cost) != FALSE {
		max_funds_available = species.econ_units
		if max_funds_available > EU_spending_limit {
			max_funds_available = EU_spending_limit
		}
		max_funds_available += balance

		if max_funds_available > 0 {
			fmt.Fprintf(log_file, "! WARNING: %s", input_line)
			fmt.Fprintf(log_file, "! Insufficient funds. Substituting %d for %d.\n",
				max_funds_available, cost)
			value = max_funds_available
			goto do_cost
		}

		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Insufficient funds to execute order.\n")
		return
	}

	/* Check if we already have knowledge of this technology. */
	funds_remaining = cost
	amount_spent = 0
	initial_level = sp_tech_level[tech]
	current_level = initial_level
	for current_level < species.tech_knowledge[tech] {
		cost_for_one_level = current_level * current_level
		cost_for_one_level -= cost_for_one_level / 4 /* 25% discount. */
		if funds_remaining < cost_for_one_level {
			break
		}
		funds_remaining -= cost_for_one_level
		amount_spent += cost_for_one_level
		current_level++
	}

	if current_level > initial_level {
		log_string("    Spent ")
		log_long(amount_spent)
		log_string(" raising ")
		log_string(tech_name[tech])
		log_string(" tech level from ")
		log_int(initial_level)
		log_string(" to ")
		log_int(current_level)
		log_string(" using transferred knowledge.\n")

		sp_tech_level[tech] = current_level
	}

	if funds_remaining == 0 {
		return
	}

	/* Increase in experience points is equal to whatever was not spent above. */
	species.tech_eps[tech] += funds_remaining

	/* Log transaction. */
	log_string("    Spent ")
	log_long(funds_remaining)
	log_string(" on ")
	log_string(tech_name[tech])
	log_string(" research.\n")
}

func do_SCAN_command() {
	var x, y, z int

	found := get_ship()
	if found == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid ship name in SCAN command.\n")
		return
	}

	if ship.status == UNDER_CONSTRUCTION {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Ship is still under construction.\n")
		return
	}

	if ship.status == FORCED_JUMP || ship.status == JUMPED_IN_COMBAT {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Ship jumped during combat and is still in transit.\n")
		return
	}

	/* Log the result. */
	if first_pass != FALSE {
		log_string("    A scan will be done by ")
		log_string(ship_name(ship))
		log_string(".\n")
		return
	}

	/* Write scan of ship's location to log file. */
	x = ship.x
	y = ship.y
	z = ship.z

	if test_mode != FALSE {
		fmt.Fprintf(log_file, "\nA scan will be done by %s.\n\n", ship_name(ship))
	} else {
		fmt.Fprintf(log_file, "\nScan done by %s:\n\n", ship_name(ship))
		scan(x, y, z, TRUE)
	}
	fmt.Fprintf(log_file, "\n")
}

func do_SEND_command() {
	var i, n, found, contact_word_number, contact_bit_number int
	var num_available, contact_mask, item_count int

	/* Get number of EUs to transfer. */
	i = get_value()

	/* Make sure value is meaningful. */
	if i == 0 || value < 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid item count in SEND command.\n")
		return
	}
	item_count = value

	num_available = species.econ_units
	if item_count == 0 {
		item_count = num_available
	}
	if item_count == 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! You do not have any EUs available!\n")
		return
	}
	if num_available < item_count {
		if num_available == 0 {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! You do not have any EUs!\n")
			return
		}
		fmt.Fprintf(log_file, "! WARNING: %s", input_line)
		fmt.Fprintf(log_file, "! You do not have %d EUs! Substituting %d for %d.\n",
			item_count, num_available, item_count)
		item_count = num_available
	}

	/* Get destination of transfer. */
	found = get_species_name()
	if found == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid species name in SEND command.\n")
		return
	}
	fmt.Fprintf(log_file, "!!! Order: SEND %d/%d SP%02d %s\n",
		num_available, species.econ_units, g_spec_number, g_spec_name)

	/* Check if we've met this species and make sure it is not an enemy. */
	contact_word_number = (g_spec_number - 1) / 32
	contact_bit_number = (g_spec_number - 1) % 32
	contact_mask = 1 << contact_bit_number
	if species.contact[contact_word_number]&uint32(contact_mask) == 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! You can't SEND to a species you haven't met.\n")
		return
	}
	if species.enemy[contact_word_number]&uint32(contact_mask) != 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! You may not SEND economic units to an ENEMY.\n")
		return
	}

	/* Make the transfer and log the result. */
	log_string("    ")
	log_long(item_count)
	log_string(" economic unit")
	if item_count > 1 {
		log_string("s were")
	} else {
		log_string(" was")
	}
	log_string(" sent to SP ")
	log_string(g_spec_name)
	log_string(".\n")
	species.econ_units -= item_count

	if first_pass != FALSE {
		return
	}

	/* Define this transaction. */
	if num_transactions == MAX_TRANSACTIONS {
		fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS!\n\n")
		os.Exit(255)
	}

	n = num_transactions
	num_transactions++
	transaction[n].trans_type = EU_TRANSFER
	transaction[n].donor = species_number
	transaction[n].recipient = g_spec_number
	transaction[n].value = item_count
	transaction[n].name1 = species.name
	transaction[n].name2 = g_spec_name

	/* Make the transfer to the alien. */
	spec_data[g_spec_number-1].econ_units += item_count
	data_modified[g_spec_number-1] = TRUE
}

func do_SHIPYARD_command() {
	var cost int

	/* Check if this order was preceded by a PRODUCTION order. */
	if doing_production == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Missing PRODUCTION order!\n")
		return
	}

	/* Make sure this is not a mining or resort colony. */
	if (nampla.status&MINING_COLONY != 0) || (nampla.status&RESORT_COLONY != 0) {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! You may not build shipyards on a mining or resort colony!\n")
		return
	}

	/* Check if planet has already built a shipyard. */
	if shipyard_built != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Only one shipyard can be built per planet per turn!\n")
		return
	}

	/* Check if sufficient funds are available. */
	cost = 10 * species.tech_level[MA]
	if check_bounced(cost) != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Insufficient funds to execute order.\n")
		return
	}

	nampla.shipyards++

	shipyard_built = TRUE

	/* Log transaction. */
	log_string("    Spent ")
	log_long(cost)
	log_string(" to increase shipyard capacity by 1.\n")
}

func do_TEACH_command() {
	var i, tech, contact_word_number, contact_bit_number, max_level_specified, need_technology int
	var temp_ptr int
	var max_tech_level int
	var contact_mask int

	/* Get technology. */
	temp_ptr = input_line_pointer
	if get_class_abbr() != TECH_ID {
		need_technology = TRUE /* Sometimes players accidentally reverse the arguments. */
		input_line_pointer = temp_ptr
	} else {
		need_technology = FALSE
		tech = abbr_index
	}

	/* See if a maximum tech level was specified. */
	max_level_specified = get_value()
	if max_level_specified != FALSE {
		max_tech_level = value
		if max_tech_level > species.tech_level[tech] {
			max_tech_level = species.tech_level[tech]
		}
	} else {
		max_tech_level = species.tech_level[tech]
	}

	/* Get the technology now if it wasn't obtained above. */
	if need_technology != FALSE {
		if get_class_abbr() != TECH_ID {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! Invalid or missing technology!\n")
			return
		}
		tech = abbr_index
	}

	/* Get species to transfer knowledge to. */
	if get_species_name() == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid species name in TEACH command.\n")
		return
	}

	/* Check if we've met this species and make sure it is not an enemy. */
	contact_word_number = (g_spec_number - 1) / 32
	contact_bit_number = (g_spec_number - 1) % 32
	contact_mask = 1 << contact_bit_number
	if species.contact[contact_word_number]&uint32(contact_mask) == 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! You can't TEACH a species you haven't met.\n")
		return
	}

	if species.enemy[contact_word_number]&uint32(contact_mask) != 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! You can't TEACH an ENEMY.\n")
		return
	}

	if first_pass != FALSE {
		return
	}

	/* Define this transaction and add to list of transactions. */
	if num_transactions == MAX_TRANSACTIONS {
		fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS!\n\n")
		os.Exit(255)
	}

	i = num_transactions
	num_transactions++
	transaction[i].trans_type = KNOWLEDGE_TRANSFER
	transaction[i].donor = species_number
	transaction[i].recipient = g_spec_number
	transaction[i].value = tech
	transaction[i].name1 = species.name
	transaction[i].number3 = max_tech_level
}

func do_TECH_command() {
	var i, tech, contact_word_number, contact_bit_number, max_level_specified, max_tech_level, max_cost_specified, need_technology int
	var contact_mask, max_cost int

	/* See if a maximum cost was specified. */
	max_cost_specified = get_value()
	if max_cost_specified != FALSE {
		max_cost = value
	} else {
		max_cost = 0
	}

	/* Get technology. */
	if get_class_abbr() != TECH_ID {
		need_technology = TRUE /* Sometimes players accidentally reverse the arguments. */
	} else {
		need_technology = FALSE
		tech = abbr_index
	}

	/* See if a maximum tech level was specified. */
	max_level_specified = get_value()
	max_tech_level = value

	/* Get species to transfer tech to. */
	if get_species_name() == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid species name in TECH command.\n")
		return
	}

	/* Check if we've met this species and make sure it is not an enemy. */
	contact_word_number = (g_spec_number - 1) / 32
	contact_bit_number = (g_spec_number - 1) % 32
	contact_mask = 1 << contact_bit_number
	if species.contact[contact_word_number]&uint32(contact_mask) == 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! You can't transfer tech to a species you haven't met.\n")
		return
	}
	if species.enemy[contact_word_number]&uint32(contact_mask) != 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! You can't transfer tech to an ENEMY.\n")
		return
	}

	/* Get the technology now if it wasn't obtained above. */
	if need_technology != FALSE {
		if get_class_abbr() != TECH_ID {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", input_line)
			fmt.Fprintf(log_file, "!!! Invalid or missing technology!\n")
			return
		}
		tech = abbr_index
	}

	/* Make sure there isn't already a transfer of the same technology from the same donor species to the same recipient species. */
	for i = 0; i < num_transactions; i++ {
		if transaction[i].trans_type != TECH_TRANSFER {
			continue
		}
		if transaction[i].value != tech {
			continue
		}
		if transaction[i].number1 != species_number {
			continue
		}
		if transaction[i].number2 != g_spec_number {
			continue
		}

		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! You can't transfer the same tech to the same species more than once!\n")
		return
	}

	/* Log the result. */
	log_string("    Will attempt to transfer ")
	log_string(tech_name[tech])
	log_string(" technology to SP ")
	log_string(g_spec_name)
	log_string(".\n")

	if first_pass != FALSE {
		return
	}

	/* Define this transaction and add to list of transactions. */
	if num_transactions == MAX_TRANSACTIONS {
		fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS!\n\n")
		os.Exit(255)
	}

	i = num_transactions
	num_transactions++
	transaction[i].trans_type = TECH_TRANSFER
	transaction[i].donor = species_number
	transaction[i].recipient = g_spec_number
	transaction[i].value = tech
	transaction[i].name1 = species.name
	transaction[i].number1 = max_cost
	transaction[i].name2 = g_spec_name
	if max_level_specified != FALSE && (max_tech_level < species.tech_level[tech]) {
		transaction[i].number3 = max_tech_level
	} else {
		transaction[i].number3 = species.tech_level[tech]
	}
}

func do_TELESCOPE_command() {
	var i, n, found, range_in_parsecs, max_range, alien_index, alien_number, alien_nampla_index, alien_ship_index, location_printed, industry, detection_chance, num_obs_locs, alien_name_printed, loc_index, success_chance, something_found int
	var x, y, z, max_distance, max_distance_squared, delta_x, delta_y, delta_z, distance_squared int
	var planet_type string
	var obs_x, obs_y, obs_z [MAX_OBS_LOCS]int
	var alien *species_data_t
	var alien_nampla *nampla_data_t
	var starbase, alien_ship *ship_data_t

	found = get_ship()
	if found == FALSE || ship.ship_type != STARBASE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid starbase name in TELESCOPE command.\n")
		return
	}
	starbase = ship

	/* Make sure starbase does not get more than one TELESCOPE order per turn. */
	if starbase.dest_z != 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! A starbase may only be given one TELESCOPE order per turn.\n")
		return
	}
	starbase.dest_z = 99

	/* Get range of telescope. */
	range_in_parsecs = starbase.item_quantity[GT] / 2
	if range_in_parsecs < 1 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Starbase is not carrying enough gravitic telescope units.\n")
		return
	}

	/* Log the result. */
	if first_pass != FALSE {
		log_string("    A gravitic telescope at ")
		log_int(starbase.x)
		log_char(' ')
		log_int(starbase.y)
		log_char(' ')
		log_int(starbase.z)
		log_string(" will be operated by ")
		log_string(ship_name(starbase))
		log_string(".\n")
		return
	}

	/* Define range parameters. */
	max_range = species.tech_level[GV] / 10
	if range_in_parsecs > max_range {
		range_in_parsecs = max_range
	}

	x = starbase.x
	y = starbase.y
	z = starbase.z

	max_distance = range_in_parsecs
	max_distance_squared = max_distance * max_distance

	/* First pass. Simply create a list of X Y Z locations that have observable aliens. */
	num_obs_locs = 0
	for alien_index = 0; alien_index < galaxy.num_species; alien_index++ {
		if data_in_memory[alien_index] == FALSE {
			continue
		}

		alien_number = alien_index + 1
		if alien_number == species_number {
			continue
		}

		alien = &spec_data[alien_index]

		for alien_nampla_index = 0; alien_nampla_index < alien.num_namplas; alien_nampla_index++ {
			alien_nampla = namp_data[alien_index][alien_nampla_index]

			if (alien_nampla.status & POPULATED) == 0 {
				continue
			}

			delta_x = x - alien_nampla.x
			delta_y = y - alien_nampla.y
			delta_z = z - alien_nampla.z
			distance_squared = (delta_x * delta_x) + (delta_y * delta_y) + (delta_z * delta_z)

			if distance_squared == 0 {
				continue
			} /* Same loc as telescope. */
			if distance_squared > max_distance_squared {
				continue
			}

			found = FALSE
			for i = 0; i < num_obs_locs; i++ {
				if alien_nampla.x != obs_x[i] {
					continue
				}
				if alien_nampla.y != obs_y[i] {
					continue
				}
				if alien_nampla.z != obs_z[i] {
					continue
				}

				found = TRUE
				break
			}
			if found == FALSE {
				if num_obs_locs == MAX_OBS_LOCS {
					fmt.Fprintf(os.Stderr, "\n\nInternal error! MAX_OBS_LOCS exceeded in do_TELESCOPE_command!\n\n")
					os.Exit(255)
				}
				obs_x[num_obs_locs] = alien_nampla.x
				obs_y[num_obs_locs] = alien_nampla.y
				obs_z[num_obs_locs] = alien_nampla.z

				num_obs_locs++
			}
		}

		for alien_ship_index = 0; alien_ship_index < alien.num_ships; alien_ship_index++ {
			alien_ship = ship_data[alien_index][alien_ship_index]

			if alien_ship.status == UNDER_CONSTRUCTION {
				continue
			}
			if alien_ship.status == ON_SURFACE {
				continue
			}
			if alien_ship.item_quantity[FD] == alien_ship.tonnage {
				continue
			}

			delta_x = x - alien_ship.x
			delta_y = y - alien_ship.y
			delta_z = z - alien_ship.z
			distance_squared = (delta_x * delta_x) + (delta_y * delta_y) + (delta_z * delta_z)

			if distance_squared == 0 {
				continue
			} /* Same loc as telescope. */
			if distance_squared > max_distance_squared {
				continue
			}

			found = FALSE
			for i = 0; i < num_obs_locs; i++ {
				if alien_ship.x != obs_x[i] {
					continue
				}
				if alien_ship.y != obs_y[i] {
					continue
				}
				if alien_ship.z != obs_z[i] {
					continue
				}

				found = TRUE
				break
			}
			if found == FALSE {
				if num_obs_locs == MAX_OBS_LOCS {
					fmt.Fprintf(os.Stderr, "\n\nInternal error! MAX_OBS_LOCS exceeded in do_TELESCOPE_command!\n\n")
					os.Exit(255)
				}
				obs_x[num_obs_locs] = alien_ship.x
				obs_y[num_obs_locs] = alien_ship.y
				obs_z[num_obs_locs] = alien_ship.z

				num_obs_locs++
			}
		}
	}

	/* Operate the gravitic telescope. */
	log_string("\n  Results of operation of gravitic telescope by ")
	log_string(ship_name(starbase))
	log_string(" (location = ")
	log_int(starbase.x)
	log_char(' ')
	log_int(starbase.y)
	log_char(' ')
	log_int(starbase.z)
	log_string(", max range = ")
	log_int(range_in_parsecs)
	log_string(" parsecs):\n")

	something_found = FALSE

	for loc_index = 0; loc_index < num_obs_locs; loc_index++ {
		x = obs_x[loc_index]
		y = obs_y[loc_index]
		z = obs_z[loc_index]

		location_printed = FALSE

		for alien_index = 0; alien_index < galaxy.num_species; alien_index++ {
			if data_in_memory[alien_index] == FALSE {
				continue
			}

			alien_number = alien_index + 1
			if alien_number == species_number {
				continue
			}

			alien = &spec_data[alien_index]

			alien_name_printed = FALSE

			for alien_nampla_index = 0; alien_nampla_index < alien.num_namplas; alien_nampla_index++ {
				alien_nampla = namp_data[alien_index][alien_nampla_index]

				if (alien_nampla.status & POPULATED) == 0 {
					continue
				}
				if alien_nampla.x != x {
					continue
				}
				if alien_nampla.y != y {
					continue
				}
				if alien_nampla.z != z {
					continue
				}

				industry = alien_nampla.mi_base + alien_nampla.ma_base

				success_chance = species.tech_level[GV]
				success_chance += starbase.item_quantity[GT]
				success_chance += (industry - 500) / 20
				if alien_nampla.hiding != FALSE || alien_nampla.hidden != FALSE {
					success_chance /= 10
				}

				if rnd(100) > success_chance {
					continue
				}

				if industry < 100 {
					industry = (industry + 5) / 10
				} else {
					industry = ((industry + 50) / 100) * 10
				}

				if alien_nampla.status&HOME_PLANET != 0 {
					planet_type = "Home planet"
				} else if alien_nampla.status&RESORT_COLONY != 0 {
					planet_type = "Resort colony"
				} else if alien_nampla.status&MINING_COLONY != 0 {
					planet_type = "Mining colony"
				} else {
					planet_type = "Colony"
				}

				if alien_name_printed == FALSE {
					if location_printed == FALSE {
						fmt.Fprintf(log_file, "\n    %d%3d%3d:\n", x, y, z)
						location_printed = TRUE
						something_found = TRUE
					}
					fmt.Fprintf(log_file, "      SP %s:\n", alien.name)
					alien_name_printed = TRUE
				}

				fmt.Fprintf(log_file, "\t#%d: %s PL %s (%d)\n",
					alien_nampla.pn, planet_type, alien_nampla.name, industry)
			}

			for alien_ship_index = 0; alien_ship_index < alien.num_ships; alien_ship_index++ {
				alien_ship = ship_data[alien_index][alien_ship_index]

				if alien_ship.x != x {
					continue
				}
				if alien_ship.y != y {
					continue
				}
				if alien_ship.z != z {
					continue
				}
				if alien_ship.status == UNDER_CONSTRUCTION {
					continue
				}
				if alien_ship.status == ON_SURFACE {
					continue
				}
				if alien_ship.item_quantity[FD] == alien_ship.tonnage {
					continue
				}

				success_chance = species.tech_level[GV]
				success_chance += starbase.item_quantity[GT]
				success_chance += alien_ship.tonnage - 10
				if alien_ship.ship_type == STARBASE {
					success_chance *= 2
				}
				if alien_ship.class == TR {
					success_chance = (3 * success_chance) / 2
				}
				if rnd(100) > success_chance {
					continue
				}

				if alien_name_printed == FALSE {
					if location_printed == FALSE {
						fmt.Fprintf(log_file, "\n    %d%3d%3d:\n", x, y, z)
						location_printed = TRUE
						something_found = TRUE
					}
					fmt.Fprintf(log_file, "      SP %s:\n", alien.name)
					alien_name_printed = TRUE
				}

				truncate_name = FALSE
				fmt.Fprintf(log_file, "\t%s", ship_name(alien_ship))
				truncate_name = TRUE

				/* See if alien detected that it is being observed. */
				if alien_ship.ship_type == STARBASE {
					detection_chance = 2 * alien_ship.item_quantity[GT]
					if detection_chance > 0 {
						fmt.Fprintf(log_file, " <- %d GTs installed!",
							alien_ship.item_quantity[GT])
					}
				} else {
					detection_chance = 0
				}

				fmt.Fprintf(log_file, "\n")

				detection_chance += 2 * (alien.tech_level[GV] - species.tech_level[GV])

				if rnd(100) > detection_chance {
					continue
				}

				/* Define this transaction. */
				if num_transactions == MAX_TRANSACTIONS {
					fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS!\n\n")
					os.Exit(255)
				}

				n = num_transactions
				num_transactions++
				transaction[n].trans_type = TELESCOPE_DETECTION
				transaction[n].x = starbase.x
				transaction[n].y = starbase.y
				transaction[n].z = starbase.z
				transaction[n].number1 = alien_number
				transaction[n].name1 = ship_name(alien_ship)
			}
		}
	}

	if something_found != FALSE {
		log_char('\n')
	} else {
		log_string("    No alien ships or planets were detected.\n\n")
	}
}

func do_TERRAFORM_command() {
	var i, j, ls_needed, num_plants, got_required_gas, correct_percentage int
	var home_planet, colony_planet *planet_data_t

	/* Get number of TPs to use. */
	if get_value() != FALSE {
		num_plants = value
	} else {
		num_plants = 0
	}

	/* Get planet where terraforming is to be done. */
	if get_location() == FALSE || nampla == nil {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid planet name in TERRAFORM command.\n")
		return
	}

	/* Make sure planet is not a home planet. */
	if nampla.status&HOME_PLANET != 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Terraforming may not be done on a home planet.\n")
		return
	}

	/* Find out how many terraforming plants are needed. */
	colony_planet = planet_base[nampla.planet_index]
	home_planet = planet_base[nampla_base[0].planet_index]

	ls_needed = life_support_needed(species, home_planet, colony_planet)

	if ls_needed == 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Colony does not need to be terraformed.\n")
		return
	}

	if num_plants == 0 {
		num_plants = nampla.item_quantity[TP]
	}
	if num_plants > ls_needed {
		num_plants = ls_needed
	}
	num_plants = num_plants / 3
	num_plants *= 3

	if num_plants < 3 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! At least three TPs are needed to terraform.\n")
		return
	}

	if num_plants > nampla.item_quantity[TP] {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! PL %s doesn't have that many TPs!\n",
			nampla.name)
		return
	}

	/* Log results. */
	log_string("    PL ")
	log_string(nampla.name)
	log_string(" was terraformed using ")
	log_int(num_plants)
	log_string(" Terraforming Unit")
	if num_plants != 1 {
		log_char('s')
	}
	log_string(".\n")

	nampla.item_quantity[TP] -= num_plants
	planet_data_modified = TRUE

	/* Terraform the planet. */
	for num_plants > 1 {
		got_required_gas = FALSE
		correct_percentage = FALSE
		for j = 0; j < 4; j++ { /* Check gases on planet. */
			for i = 0; i < 6; i++ { /* Compare with poisonous gases. */
				if colony_planet.gas[j] == species.required_gas {
					got_required_gas = j + 1

					if colony_planet.gas_percent[j] >= species.required_gas_min &&
						colony_planet.gas_percent[j] <= species.required_gas_max {
						correct_percentage = TRUE
					}
				}

				if species.poison_gas[i] == colony_planet.gas[j] {
					colony_planet.gas[j] = 0
					colony_planet.gas_percent[j] = 0

					/* Make sure percentages add up to 100%. */
					fix_gases(colony_planet)

					goto next_change
				}
			}
		}

		if got_required_gas != FALSE && correct_percentage != FALSE {
			goto do_temp
		}

		j = 0 /* If all 4 gases are neutral gases, replace the first one. */

		if got_required_gas != FALSE {
			j = got_required_gas - 1
		} else {
			for i = 0; i < 4; i++ {
				if colony_planet.gas_percent[i] == 0 {
					j = i
					break
				}
			}
		}

		colony_planet.gas[j] = species.required_gas
		i = species.required_gas_max - species.required_gas_min
		colony_planet.gas_percent[j] = species.required_gas_min + rnd(i)

		/* Make sure percentages add up to 100%. */
		fix_gases(colony_planet)

		goto next_change

	do_temp:

		if colony_planet.temperature_class != home_planet.temperature_class {
			if colony_planet.temperature_class > home_planet.temperature_class {
				colony_planet.temperature_class--
			} else {
				colony_planet.temperature_class++
			}

			goto next_change
		}

		if colony_planet.pressure_class != home_planet.pressure_class {
			if colony_planet.pressure_class > home_planet.pressure_class {
				colony_planet.pressure_class--
			} else {
				colony_planet.pressure_class++
			}
		}

	next_change:

		num_plants -= 3
	}
}

func do_TRANSFER_command() {
	var i, n, item_class, item_count, capacity, transfer_type int
	var attempt_during_siege, siege_1_chance, siege_2_chance int
	var alien_number, first_try, both_args_present, need_destination int
	var c byte
	var x1, x2, y1, y2, z1, z2 int
	var original_line_pointer, temp_ptr int
	var already_notified [MAX_SPECIES]int
	var num_available, original_count int
	var nampla1, nampla2, temp_nampla *nampla_data_t
	var ship1, ship2 *ship_data_t

	/* Get number of items to transfer. */
	i = get_value()

	/* Make sure value is meaningful. */
	if i == 0 || value < 0 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Invalid item count in TRANSFER command.\n")
		return
	}
	original_count = value
	item_count = value

	/* Get class of item. */
	item_class = get_class_abbr()

	if item_class != ITEM_CLASS {
		/* Players sometimes accidentally use "MI" for "IU" or "MA" for "AU". */
		if item_class == TECH_ID && abbr_index == MI {
			abbr_index = IU
		} else if item_class == TECH_ID && abbr_index == MA {
			abbr_index = AU
		} else {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Invalid item class!\n")
			return
		}
	}
	item_class = abbr_index

	/* Get source of transfer. */
	nampla1 = nil
	nampla2 = nil
	original_line_pointer = input_line_pointer
	if get_transfer_point() == FALSE {
		/* Check for missing comma or tab after source name. */
		input_line_pointer = original_line_pointer
		fix_separator()
		if get_transfer_point() == FALSE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Invalid source location in TRANSFER command.\n")
			return
		}
	}

	/* Test if the order has both a source and a destination.
	 * Sometimes, the player will accidentally omit the source if it's "obvious". */
	temp_ptr = input_line_pointer
	both_args_present = FALSE
	for {
		c = at(input_line, temp_ptr)
		temp_ptr++
		if c == ';' || c == '\n' || c == 0 { /* End of order. */
			break
		}
		if isalpha(c) {
			both_args_present = TRUE
			break
		}
	}

	need_destination = TRUE

	/* Make sure everything makes sense. */
	if abbr_type == SHIP_CLASS {
		ship1 = ship

		if ship1.status == UNDER_CONSTRUCTION {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! %s is still under construction!\n", ship_name(ship1))
			return
		}

		if ship1.status == FORCED_JUMP || ship1.status == JUMPED_IN_COMBAT {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Ship jumped during combat and is still in transit.\n")
			return
		}

		x1 = ship1.x
		y1 = ship1.y
		z1 = ship1.z

		num_available = ship1.item_quantity[item_class]

	check_ship_items:

		if item_count == 0 {
			item_count = num_available
		}
		if item_count == 0 {
			return
		}

		if num_available < item_count {
			if both_args_present != FALSE { /* Change item count to "0". */
				if num_available == 0 {
					fmt.Fprintf(log_file, "!!! Order ignored:\n")
					fmt.Fprintf(log_file, "!!! %s", original_line)
					fmt.Fprintf(log_file, "!!! %s does not have specified item(s)!\n", ship_name(ship1))
					return
				}

				fmt.Fprintf(log_file, "! WARNING: %s", original_line)
				fmt.Fprintf(log_file, "! Ship does not have %d units. Substituting %d for %d!\n",
					item_count, num_available, item_count)
				item_count = 0
				goto check_ship_items
			}

			/* Check if ship is at a planet that has the items.
			 * If so, we'll assume that the planet is the source and the ship is the destination.
			 * We'll look first for a planet that the ship is actually landed on or orbiting.
			 * If that fails, then we'll look for a planet in the same sector. */
			first_try = TRUE

		next_ship_try:

			for i = 0; i < species.num_namplas; i++ {
				nampla1 = nampla_base[i]
				if nampla1.x != ship1.x {
					continue
				}
				if nampla1.y != ship1.y {
					continue
				}
				if nampla1.z != ship1.z {
					continue
				}
				if first_try != FALSE {
					if nampla1.pn != ship1.pn {
						continue
					}
				}

				num_available = nampla1.item_quantity[item_class]
				if num_available < item_count {
					continue
				}

				ship = ship1           /* Destination. */
				transfer_type = 1      /* Source = planet. */
				abbr_type = SHIP_CLASS /* Destination type. */

				need_destination = FALSE

				goto get_destination
			}

			if first_try != FALSE {
				first_try = FALSE
				goto next_ship_try
			}

			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! %s does not have specified item(s)!\n", ship_name(ship1))
			return
		}

		transfer_type = 0 /* Source = ship. */
	} else {
		/* Source is a planet. */
		nampla1 = nampla

		x1 = nampla1.x
		y1 = nampla1.y
		z1 = nampla1.z

		num_available = nampla1.item_quantity[item_class]

	check_planet_items:

		if item_count == 0 {
			item_count = num_available
		}
		if item_count == 0 {
			return
		}

		if num_available < item_count {
			if both_args_present != FALSE {
				/* Change item count to "0". */
				if num_available == 0 {
					fmt.Fprintf(log_file, "!!! Order ignored:\n")
					fmt.Fprintf(log_file, "!!! %s", original_line)
					fmt.Fprintf(log_file, "!!! PL %s does not have specified item(s)!\n", nampla1.name)
					return
				}

				fmt.Fprintf(log_file, "! WARNING: %s", original_line)
				fmt.Fprintf(log_file, "! Planet does not have %d units. Substituting %d for %d!\n",
					item_count, num_available, item_count)
				item_count = 0
				goto check_planet_items
			}

			/* Check if another planet in the same sector has the items.
			 * If so, we'll assume that it is the source and that the named planet is the destination. */
			for i = 0; i < species.num_namplas; i++ {
				temp_nampla = nampla_base[i]
				if temp_nampla.x != nampla1.x {
					continue
				}
				if temp_nampla.y != nampla1.y {
					continue
				}
				if temp_nampla.z != nampla1.z {
					continue
				}

				num_available = temp_nampla.item_quantity[item_class]
				if num_available < item_count {
					continue
				}

				nampla = nampla1      /* Destination. */
				nampla1 = temp_nampla /* Source. */
				transfer_type = 1     /* Source = planet. */
				abbr_type = PLANET_ID /* Destination type. */

				need_destination = FALSE

				goto get_destination
			}

			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! PL %s does not have specified item(s)!\n", nampla1.name)
			return
		}

		transfer_type = 1 /* Source = planet. */
	}

get_destination:

	/* Get destination of transfer. */
	if need_destination != FALSE {
		if get_transfer_point() == FALSE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Invalid destination location.\n")
			return
		}
	}

	/* Make sure everything makes sense. */
	if abbr_type == SHIP_CLASS {
		ship2 = ship

		if ship2.status == UNDER_CONSTRUCTION {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! %s is still under construction!\n", ship_name(ship2))
			return
		}

		if ship2.status == FORCED_JUMP || ship2.status == JUMPED_IN_COMBAT {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Ship jumped during combat and is still in transit.\n")
			return
		}

		/* Check if destination ship has sufficient carrying capacity. */
		if ship2.class == TR {
			capacity = (10 + (ship2.tonnage / 2)) * ship2.tonnage
		} else if ship2.class == BA {
			capacity = 10 * ship2.tonnage
		} else {
			capacity = ship2.tonnage
		}

		for i = 0; i < MAX_ITEMS; i++ {
			capacity -= ship2.item_quantity[i] * item_carry_capacity[i]
		}

	do_capacity:

		if original_count == 0 {
			i = capacity / item_carry_capacity[item_class]
			if i < item_count {
				item_count = i
			}
			if item_count == 0 {
				return
			}
		}

		if capacity < item_count*item_carry_capacity[item_class] {
			fmt.Fprintf(log_file, "! WARNING: %s", original_line)
			fmt.Fprintf(log_file, "! %s does not have sufficient carrying capacity!",
				ship_name(ship2))
			fmt.Fprintf(log_file, " Changed %d to 0.\n", original_count)
			original_count = 0
			goto do_capacity
		}

		x2 = ship2.x
		y2 = ship2.y
		z2 = ship2.z
	} else {
		nampla2 = nampla

		x2 = nampla2.x
		y2 = nampla2.y
		z2 = nampla2.z

		transfer_type |= 2

		/* If this is the post-arrival phase, then make sure the planet is populated. */
		if post_arrival_phase != FALSE && ((nampla2.status & POPULATED) == 0) {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Destination planet must be populated for post-arrival TRANSFERs.\n")
			return
		}
	}

	/* Check if source and destination are in same system. */
	if x1 != x2 || y1 != y2 || z1 != z2 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Source and destination are not at same 'x y z' in TRANSFER command.\n")
		return
	}

	/* Check for siege. */
	siege_1_chance = 0
	siege_2_chance = 0
	if transfer_type == 3 && /* Planet to planet. */
		(nampla1.siege_eff != 0 || nampla2.siege_eff != 0) {
		if nampla1.siege_eff >= 0 {
			siege_1_chance = nampla1.siege_eff
		} else {
			siege_1_chance = -nampla1.siege_eff
		}

		if nampla2.siege_eff >= 0 {
			siege_2_chance = nampla2.siege_eff
		} else {
			siege_2_chance = -nampla2.siege_eff
		}

		attempt_during_siege = TRUE
	} else {
		attempt_during_siege = FALSE
	}

	/* Make the transfer and log the result. */
	log_string("    ")

	if attempt_during_siege != FALSE && first_pass != FALSE {
		log_string("An attempt will be made to transfer ")
	}

	log_int(item_count)
	log_char(' ')
	log_string(item_name[item_class])

	if attempt_during_siege != FALSE && first_pass != FALSE {
		if item_count > 1 {
			log_char('s')
		}
		log_char(' ')
	} else {
		if item_count > 1 {
			log_string("s were transferred from ")
		} else {
			log_string(" was transferred from ")
		}
	}

	switch transfer_type {
	case 0: /* Ship to ship. */
		ship1.item_quantity[item_class] -= item_count
		ship2.item_quantity[item_class] += item_count
		log_string(ship_name(ship1))
		log_string(" to ")
		log_string(ship_name(ship2))
		log_char('.')

	case 1: /* Planet to ship. */
		nampla1.item_quantity[item_class] -= item_count
		ship2.item_quantity[item_class] += item_count
		if item_class == CU {
			if nampla1 == nampla_base[0] {
				ship2.loading_point = 9999 /* Home planet. */
			} else {
				/* C: ship2->loading_point = (nampla1 - nampla_base); */
				for i = 0; i < species.num_namplas; i++ {
					if nampla_base[i] == nampla1 {
						ship2.loading_point = i
						break
					}
				}
			}
		}
		log_string("PL ")
		log_string(nampla1.name)
		log_string(" to ")
		log_string(ship_name(ship2))
		log_char('.')

	case 2: /* Ship to planet. */
		ship1.item_quantity[item_class] -= item_count
		nampla2.item_quantity[item_class] += item_count
		log_string(ship_name(ship1))
		log_string(" to PL ")
		log_string(nampla2.name)
		log_char('.')

	case 3: /* Planet to planet. */
		nampla1.item_quantity[item_class] -= item_count
		nampla2.item_quantity[item_class] += item_count

		log_string("PL ")
		log_string(nampla1.name)
		log_string(" to PL ")
		log_string(nampla2.name)
		if attempt_during_siege != FALSE {
			log_string(" despite the siege")
		}
		log_char('.')

		if first_pass != FALSE {
			break
		}

		/* Check if either planet is under siege and if transfer
		   was detected by the besiegers. */
		if rnd(100) > siege_1_chance && rnd(100) > siege_2_chance {
			break
		}

		log_string(" However, the transfer was detected by the besiegers and the items were destroyed!!!")
		nampla2.item_quantity[item_class] -= item_count

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
			transaction[n].value = 4 /* Transfer of items. */
			transaction[n].number1 = item_count
			transaction[n].number2 = item_class
			if siege_1_chance > siege_2_chance {
				/* Besieged planet is the source of the transfer. */
				transaction[n].value = 4
				transaction[n].name1 = nampla1.name
				transaction[n].name2 = nampla2.name
			} else {
				/* Besieged planet is the destination of the transfer. */
				transaction[n].value = 5
				transaction[n].name1 = nampla2.name
				transaction[n].name2 = nampla1.name
			}
			transaction[n].name3 = species.name
			transaction[n].number3 = alien_number

			already_notified[alien_number-1] = TRUE
		}

	default: /* Internal error. */
		fmt.Fprintf(os.Stderr, "\n\n\tInternal error: transfer type!\n\n")
		os.Exit(255)
	}

	log_char('\n')

	if nampla1 != nil {
		check_population(nampla1)
	}
	if nampla2 != nil {
		check_population(nampla2)
	}
}

func do_UNLOAD_command() {
	var i, found, item_count, recovering_home_planet, alien_index int
	var n, reb int
	var alien_home_nampla *nampla_data_t

	/* Get the ship. */
	if get_ship() == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid ship name in UNLOAD command.\n")
		return
	}

	if ship.status == UNDER_CONSTRUCTION {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Ship is still under construction.\n")
		return
	}

	if ship.status == FORCED_JUMP || ship.status == JUMPED_IN_COMBAT {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Ship jumped during combat and is still in transit.\n")
		return
	}

	/* Find which planet the ship is at. */
	found = FALSE
	for i = 0; i < species.num_namplas; i++ {
		nampla = nampla_base[i]
		if ship.x != nampla.x {
			continue
		}
		if ship.y != nampla.y {
			continue
		}
		if ship.z != nampla.z {
			continue
		}
		if ship.pn != nampla.pn {
			continue
		}
		found = TRUE
		break
	}

	if found == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Ship is not at a named planet.\n")
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
		n = nampla.mi_base + nampla.ma_base + nampla.IUs_to_install +
			nampla.AUs_to_install
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

	/* Transfer the items from the ship to the planet. */
	log_string("    ")

	item_count = ship.item_quantity[CU]
	nampla.item_quantity[CU] += item_count
	log_int(item_count)
	log_char(' ')
	log_string(item_abbr[CU])
	if item_count != 1 {
		log_char('s')
	}
	ship.item_quantity[CU] = 0

	item_count = ship.item_quantity[IU]
	nampla.item_quantity[IU] += item_count
	log_string(", ")
	log_int(item_count)
	log_char(' ')
	log_string(item_abbr[IU])
	if item_count != 1 {
		log_char('s')
	}
	ship.item_quantity[IU] = 0

	item_count = ship.item_quantity[AU]
	nampla.item_quantity[AU] += item_count
	log_string(", and ")
	log_int(item_count)
	log_char(' ')
	log_string(item_abbr[AU])
	if item_count != 1 {
		log_char('s')
	}
	ship.item_quantity[AU] = 0

	log_string(" were transferred from ")
	log_string(ship_name(ship))
	log_string(" to PL ")
	log_string(nampla.name)
	log_string(". ")

	/* Do the installation. */
	item_count = nampla.item_quantity[CU]
	if item_count > nampla.item_quantity[IU] {
		item_count = nampla.item_quantity[IU]
	}
	if recovering_home_planet != FALSE {
		if item_count > reb {
			item_count = reb
		}
		reb -= item_count
	}

	nampla.item_quantity[CU] -= item_count
	nampla.item_quantity[IU] -= item_count
	nampla.IUs_to_install += item_count

	log_string("Installation of ")
	log_int(item_count)
	log_char(' ')
	log_string(item_abbr[IU])
	if item_count != 1 {
		log_char('s')
	}

	item_count = nampla.item_quantity[CU]
	if item_count > nampla.item_quantity[AU] {
		item_count = nampla.item_quantity[AU]
	}
	if recovering_home_planet != FALSE {
		if item_count > reb {
			item_count = reb
		}
		reb -= item_count
	}

	nampla.item_quantity[CU] -= item_count
	nampla.item_quantity[AU] -= item_count
	nampla.AUs_to_install += item_count

	log_string(" and ")
	log_int(item_count)
	log_char(' ')
	log_string(item_abbr[AU])
	if item_count != 1 {
		log_char('s')
	}
	log_string(" began on the planet.\n")

	check_population(nampla)
}

func do_UPGRADE_command() {
	var age_reduction, value_specified int
	var original_line_pointer int
	var amount_to_spend, original_cost, max_funds_available int

	/* Check if this order was preceded by a PRODUCTION order. */
	if doing_production == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Missing PRODUCTION order!\n")
		return
	}

	/* Get the ship to be upgraded. */
	original_line_pointer = input_line_pointer
	if get_ship() == FALSE {
		/* Check for missing comma or tab after ship name. */
		input_line_pointer = original_line_pointer
		fix_separator()
		if get_ship() == FALSE {
			fmt.Fprintf(log_file, "!!! Order ignored:\n")
			fmt.Fprintf(log_file, "!!! %s", original_line)
			fmt.Fprintf(log_file, "!!! Ship to be upgraded does not exist.\n")
			return
		}
	}

	/* Make sure it didn't just jump. */
	if ship.just_jumped != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Ship just jumped and is still in transit.\n")
		return
	}

	/* Make sure it's in the same sector as the producing planet. */
	if ship.x != nampla.x || ship.y != nampla.y || ship.z != nampla.z {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Item to be upgraded is not in the same sector as the production planet.\n")
		return
	}

	if ship.status == UNDER_CONSTRUCTION {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Item to be upgraded is still under construction.\n")
		return
	}

	if ship.age < 1 {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Ship or starbase is too new to upgrade.\n")
		return
	}

	/* Calculate the original cost of the ship. */
	if ship.class == TR || ship.ship_type == STARBASE {
		original_cost = ship_cost[ship.class] * ship.tonnage
	} else {
		original_cost = ship_cost[ship.class]
	}

	if ship.ship_type == SUB_LIGHT {
		original_cost = (3 * original_cost) / 4
	}

	/* Get amount to be spent. */
	value_specified = get_value()
	if value_specified != FALSE {
		if value == 0 {
			amount_to_spend = balance
		} else {
			amount_to_spend = value
		}

		age_reduction = (40 * amount_to_spend) / original_cost
	} else {
		age_reduction = ship.age
	}

try_again:

	if age_reduction < 1 {
		if value == 0 {
			return
		}
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! Amount specified is not enough to do an upgrade.\n")
		return
	}

	if age_reduction > ship.age {
		age_reduction = ship.age
	}

	/* Check if sufficient funds are available. */
	amount_to_spend = ((age_reduction * original_cost) + 39) / 40
	if check_bounced(amount_to_spend) != FALSE {
		max_funds_available = species.econ_units
		if max_funds_available > EU_spending_limit {
			max_funds_available = EU_spending_limit
		}
		max_funds_available += balance

		if max_funds_available > 0 {
			if value_specified != FALSE {
				fmt.Fprintf(log_file, "! WARNING: %s", input_line)
				fmt.Fprintf(log_file, "! Insufficient funds. Substituting %d for %d.\n", max_funds_available, value)
			}
			amount_to_spend = max_funds_available
			age_reduction = (40 * amount_to_spend) / original_cost
			goto try_again
		}

		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Insufficient funds to execute order.\n")
		return
	}

	/* Log what was upgraded. */
	log_string("    ")
	log_string(ship_name(ship))
	log_string(" was upgraded from age ")
	log_int(ship.age)
	log_string(" to age ")
	ship.age -= age_reduction
	log_int(ship.age)
	log_string(" at a cost of ")
	log_long(amount_to_spend)
	log_string(".\n")
}

func do_VISITED_command() {
	/* Get x y z coordinates. */
	found := get_location()
	if found == FALSE || nampla != nil {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! Invalid coordinates in VISITED command.\n")
		return
	}

	found = star_visited(x, y, z)

	if found == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", input_line)
		fmt.Fprintf(log_file, "!!! There is no star system at these coordinates.\n")
		return
	}

	/* Log result. */
	log_string("    The star system at ")
	log_int(x)
	log_char(' ')
	log_int(y)
	log_char(' ')
	log_int(z)
	log_string(" was marked as visited.\n")
}

func do_WORMHOLE_command() {
	var i int
	var star *star_data_t

	/* Get ship making the jump. */
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
			fmt.Fprintf(log_file, "!!! Invalid ship name in WORMHOLE command.\n")
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

	/* Make sure ship can jump. */
	if ship.status == UNDER_CONSTRUCTION {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! %s is still under construction!\n", ship_name(ship))
		return
	}

	/* Check if JUMP, MOVE, or WORMHOLE was already done for this ship. */
	if ship.just_jumped != FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! %s already jumped or moved this turn!\n", ship_name(ship))
		return
	}

	/* Find star. */
	found = FALSE
	for i = 0; i < num_stars; i++ {
		star = star_base[i]
		if star.x == ship.x && star.y == ship.y && star.z == ship.z {
			found = star.worm_here
			break
		}
	}

	if found == FALSE {
		fmt.Fprintf(log_file, "!!! Order ignored:\n")
		fmt.Fprintf(log_file, "!!! %s", original_line)
		fmt.Fprintf(log_file, "!!! There is no wormhole at ship's location!\n")
		return
	}

	/* Get the destination planet, if any. */
	get_location()
	if nampla != nil {
		if nampla.x != star.worm_x || nampla.y != star.worm_y || nampla.z != star.worm_z {
			fmt.Fprintf(log_file, "!!! WARNING - Destination planet is not at other end of wormhole!\n")
			nampla = nil
		}
	}

	/* Do the jump. */
	log_string("    ")
	log_string(ship_name(ship))
	log_string(" will jump via natural wormhole at ")
	log_int(ship.x)
	log_char(' ')
	log_int(ship.y)
	log_char(' ')
	log_int(ship.z)
	ship.pn = 0
	ship.status = IN_DEEP_SPACE

	if nampla != nil {
		log_string(" to PL ")
		log_string(nampla.name)
		ship.pn = nampla.pn
		ship.status = IN_ORBIT
	}
	log_string(".\n")
	ship.x = star.worm_x
	ship.y = star.worm_y
	ship.z = star.worm_z
	ship.just_jumped = 99 /* 99 indicates that a wormhole was used. */

	if first_pass == FALSE {
		star_visited(ship.x, ship.y, ship.z)
	}
}
