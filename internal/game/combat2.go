package game

import (
	"fmt"
	"os"
)

// Port of combat.c lines 2029-3601: do_bombardment, do_germ_warfare,
// do_round, do_siege, fighting_params, forced_jump_units_used,
// regenerate_shields, withdrawal_check.
//
// NOTE on the C type-pun: combat.c stores both ship_data and nampla_data
// pointers in act->fighting_unit[] (a char* array) discriminated by
// act->unit_type[]. The Go port carries act.fighting_unit_ship[] and
// act.fighting_unit_nampla[]. During PLANET_BOMBARDMENT, GERM_WARFARE and
// SIEGE simulation rounds, do_round retags ANY targeted defender (ships
// included) to GENOCIDE_NAMPLA/BESIEGED_NAMPLA; the C code then reads such
// a ship record through a nampla_data* cast. Where do_round can reach those
// reads, the exact C struct-layout aliasing is reproduced here (both
// structs are all-int with name[32] at the same offset):
//
//	nampla field (int offset)        ship field at same offset
//	name (1..8)                      name
//	status (13)                      status
//	item_quantity[PD] (29)           item_quantity[FS]
//	item_quantity[GW] (42)           item_quantity[SG6]
//	item_quantity[32=GU9] (60)       age
//	item_quantity[36] (64)           special

func do_bombardment(unit_index int, act *action_data_t) {
	var i, new_mi, new_ma, defending_species int
	var n, total_bomb_damage, CS_bomb_damage, new_pop, initial_base, total_pop, percent_damage int
	var attacked_nampla *nampla_data_t
	var sh *ship_data_t

	attacked_nampla = act.fighting_unit_nampla[unit_index]
	/* C assigns planet = planet_base + attacked_nampla->planet_index here, but never uses it. */

	initial_base = attacked_nampla.mi_base + attacked_nampla.ma_base
	total_pop = initial_base

	if attacked_nampla.item_quantity[CU] > 0 {
		total_pop += 1
	}
	if total_pop < 1 {
		log_string("        The planet is completely uninhabited. There is nothing to bomb!\n")
		return
	}

	/* Total damage done by ten strike cruisers (ML = 50) in ten rounds
	 * is 100 x 4 x the power value for a single ship. To eliminate the
	 * chance of overflow, the algorithm has been carefully chosen. */
	CS_bomb_damage = 400 * power(ship_tonnage[CS]) /* Should be 400 * 4759 = 1,903,600. */

	total_bomb_damage = act.bomb_damage[unit_index]

	/* Keep about 2 significant digits. */
	for total_bomb_damage > 1000 {
		total_bomb_damage /= 10
		CS_bomb_damage /= 10
	}

	if CS_bomb_damage == 0 {
		percent_damage = 101
	} else {
		percent_damage = ((total_bomb_damage * 250000) / CS_bomb_damage) / total_pop
	}
	if percent_damage > 100 {
		percent_damage = 101
	}

	new_mi = attacked_nampla.mi_base - (percent_damage*attacked_nampla.mi_base)/100
	new_ma = attacked_nampla.ma_base - (percent_damage*attacked_nampla.ma_base)/100
	new_pop = attacked_nampla.pop_units - (percent_damage*attacked_nampla.pop_units)/100

	if new_mi == attacked_nampla.mi_base && new_ma == attacked_nampla.ma_base &&
		new_pop == attacked_nampla.pop_units {
		log_string("        Damage due to bombardment was insignificant.\n")
		return
	}

	defending_species = act.fighting_species_index[unit_index]
	if attacked_nampla.status&HOME_PLANET != 0 {
		n = attacked_nampla.mi_base + attacked_nampla.ma_base
		if c_species[defending_species].hp_original_base < n {
			c_species[defending_species].hp_original_base = n
		}
	}

	if new_mi <= 0 && new_ma <= 0 && new_pop <= 0 {
		log_string("        Everyone and everything was completely wiped out!\n")

		attacked_nampla.mi_base = 0
		attacked_nampla.ma_base = 0
		attacked_nampla.pop_units = 0
		attacked_nampla.siege_eff = 0
		attacked_nampla.shipyards = 0
		attacked_nampla.hiding = 0
		attacked_nampla.hidden = 0
		attacked_nampla.use_on_ambush = 0

		/* Reset status. */
		if attacked_nampla.status&HOME_PLANET != 0 {
			attacked_nampla.status = HOME_PLANET
		} else {
			attacked_nampla.status = COLONY
		}

		for i = 0; i < MAX_ITEMS; i++ {
			attacked_nampla.item_quantity[i] = 0
		}

		/* Delete any ships that were under construction on the planet. */
		for i = 0; i < c_species[defending_species].num_ships; i++ {
			sh = c_ship[defending_species][i]
			if sh.x != attacked_nampla.x {
				continue
			}
			if sh.y != attacked_nampla.y {
				continue
			}
			if sh.z != attacked_nampla.z {
				continue
			}
			if sh.pn != attacked_nampla.pn {
				continue
			}
			delete_ship(sh)
		}

		return
	}

	log_string("        Mining base of PL ")
	log_string(attacked_nampla.name)
	log_string(" went from ")
	log_int(attacked_nampla.mi_base / 10)
	log_char('.')
	log_int(attacked_nampla.mi_base % 10)
	log_string(" to ")
	attacked_nampla.mi_base = new_mi
	log_int(new_mi / 10)
	log_char('.')
	log_int(new_mi % 10)
	log_string(".\n")

	log_string("        Manufacturing base of PL ")
	log_string(attacked_nampla.name)
	log_string(" went from ")
	log_int(attacked_nampla.ma_base / 10)
	log_char('.')
	log_int(attacked_nampla.ma_base % 10)
	log_string(" to ")
	attacked_nampla.ma_base = new_ma
	log_int(new_ma / 10)
	log_char('.')
	log_int(new_ma % 10)
	log_string(".\n")

	attacked_nampla.pop_units = new_pop

	for i = 0; i < MAX_ITEMS; i++ {
		n = (percent_damage * attacked_nampla.item_quantity[i]) / 100
		if n > 0 {
			attacked_nampla.item_quantity[i] -= n
			log_string("        ")
			log_long(n)
			log_char(' ')
			log_string(item_name[i])
			if n > 1 {
				log_string("s were")
			} else {
				log_string(" was")
			}
			log_string(" destroyed.\n")
		}
	}

	n = (percent_damage * attacked_nampla.shipyards) / 100
	if n > 0 {
		attacked_nampla.shipyards -= n
		log_string("        ")
		log_long(n)
		log_string(" shipyard")
		if n > 1 {
			log_string("s were")
		} else {
			log_string(" was")
		}
		log_string(" also destroyed.\n")
	}

	check_population(attacked_nampla)
}

func do_germ_warfare(attacking_species, defending_species, defender_index int, bat *battle_data_t, act *action_data_t) {
	var i, attacker_BI, defender_BI, success_chance, num_bombs, success int
	var econ_units_from_looting int
	var attacked_nampla *nampla_data_t
	var sh *ship_data_t

	attacker_BI = c_species[attacking_species].tech_level[BI]
	defender_BI = c_species[defending_species].tech_level[BI]
	attacked_nampla = act.fighting_unit_nampla[defender_index]
	/* C assigns planet = planet_base + attacked_nampla->planet_index here, but never uses it. */

	success_chance = 50 + (2 * (attacker_BI - defender_BI))
	success = FALSE
	num_bombs = germ_bombs_used[attacking_species][defending_species]

	for i = 0; i < num_bombs; i++ {
		if rnd(100) <= success_chance {
			success = TRUE
			break
		}
	}

	if success != FALSE {
		log_string("        Unfortunately")
	} else {
		log_string("        Fortunately")
	}
	log_string(" for the ")
	log_string(c_species[defending_species].name)
	log_string(" defenders of PL ")
	log_string(attacked_nampla.name)
	log_string(", the ")
	i = bat.spec_num[attacking_species]
	if field_distorted[attacking_species] != FALSE {
		log_int(distorted(i))
	} else {
		log_string(c_species[attacking_species].name)
	}
	log_string(" attackers ")

	if success == FALSE {
		log_string("failed")
		if num_bombs <= 0 {
			log_string(" because they didn't have any germ warfare bombs")
		}
		log_string("!\n")
		return
	}

	log_string("succeeded, using ")
	log_int(num_bombs)
	log_string(" germ warfare bombs. The defenders were wiped out!\n")

	/* Take care of looting. */
	econ_units_from_looting = attacked_nampla.mi_base + attacked_nampla.ma_base

	if attacked_nampla.status&HOME_PLANET != 0 {
		if c_species[defending_species].hp_original_base < econ_units_from_looting {
			c_species[defending_species].hp_original_base = econ_units_from_looting
		}
		econ_units_from_looting *= 5
	}

	if econ_units_from_looting > 0 {
		/* Check if there's enough memory for a new interspecies transaction. */
		if num_transactions == MAX_TRANSACTIONS {
			fmt.Fprintf(os.Stderr, "\nRan out of memory! MAX_TRANSACTIONS is too small!\n\n")
			os.Exit(255)
		}
		i = num_transactions
		num_transactions++

		/* Define this transaction. */
		transaction[i].trans_type = LOOTING_EU_TRANSFER
		transaction[i].donor = bat.spec_num[defending_species]
		transaction[i].recipient = bat.spec_num[attacking_species]
		transaction[i].value = econ_units_from_looting
		transaction[i].name1 = c_species[defending_species].name
		transaction[i].name2 = c_species[attacking_species].name
		transaction[i].name3 = attacked_nampla.name
	}

	/* Finish off defenders. */
	attacked_nampla.mi_base = 0
	attacked_nampla.ma_base = 0
	attacked_nampla.IUs_to_install = 0
	attacked_nampla.AUs_to_install = 0
	attacked_nampla.pop_units = 0
	attacked_nampla.siege_eff = 0
	attacked_nampla.shipyards = 0
	attacked_nampla.hiding = 0
	attacked_nampla.hidden = 0
	attacked_nampla.use_on_ambush = 0

	for i = 0; i < MAX_ITEMS; i++ {
		attacked_nampla.item_quantity[i] = 0
	}

	/* Reset status word. */
	if attacked_nampla.status&HOME_PLANET != 0 {
		attacked_nampla.status = HOME_PLANET
	} else {
		attacked_nampla.status = COLONY
	}

	/* Delete any ships that were under construction on the planet. */
	for i = 0; i < c_species[defending_species].num_ships; i++ {
		sh = c_ship[defending_species][i]
		if sh.x != attacked_nampla.x {
			continue
		}
		if sh.y != attacked_nampla.y {
			continue
		}
		if sh.z != attacked_nampla.z {
			continue
		}
		if sh.pn != attacked_nampla.pn {
			continue
		}
		delete_ship(sh)
	}
}

/* The following routine will return TRUE if a round of combat actually occurred. Otherwise, it will return false. */
func do_round(option, round_number int, bat *battle_data_t, act *action_data_t) int {
	var i, j, n, unit_index, combat_occurred, total_shots int
	var attacker_index, defender_index, chance_to_hit int
	var attacker_ml, attacker_gv, defender_ml int
	var target_index [MAX_SHIPS]int
	var num_targets, header_printed, num_sp, fj_chance, shields_up int
	var FDs_were_destroyed int
	var di [3]int
	var start_unit, current_species, this_is_a_hijacking int
	var units_destroyed, percent_decrease int
	var damage_done, damage_to_ship, damage_to_shields, op1, op2 int
	var original_cost, recycle_value, economic_units int
	var attacker_name, defender_name string
	var attacking_species, defending_species *species_data_t
	var sh, attacking_ship, defending_ship *ship_data_t
	var attacking_nampla, defending_nampla *nampla_data_t

	/* Clear out x_attacked_y and germ_bombs_used arrays.  They will be used to log who bombed who, or how many GWs were used. */
	num_sp = bat.num_species_here
	for i = 0; i < num_sp; i++ {
		for j = 0; j < num_sp; j++ {
			x_attacked_y[i][j] = FALSE
			germ_bombs_used[i][j] = 0
		}
	}

	/* If a species has ONLY non-combatants left, then let them fight. */
	start_unit = 0
	total_shots = 0
	current_species = act.fighting_species_index[0]
	for unit_index = 0; unit_index < act.num_units_fighting; unit_index++ {
		if act.fighting_species_index[unit_index] != current_species {
			if total_shots == 0 {
				/* Convert all non-combatants, if any, to combatants. */
				for i = start_unit; i < unit_index; i++ {
					if act.unit_type[i] == SHIP {
						sh = act.fighting_unit_ship[i]
						sh.special = 0
					}
				}
			}
			start_unit = unit_index
			total_shots = 0
		}

		n = act.num_shots[unit_index]
		if act.surprised[unit_index] != FALSE {
			n = 0
		}
		if act.unit_type[unit_index] == SHIP {
			sh = act.fighting_unit_ship[unit_index]
			if sh.special == NON_COMBATANT {
				n = 0
			}
		}
		total_shots += n
	}

	/* Determine total number of shots for all species present. */
	total_shots = 0
	for unit_index = 0; unit_index < act.num_units_fighting; unit_index++ {
		n = act.num_shots[unit_index]
		if act.surprised[unit_index] != FALSE {
			n = 0
		}
		if act.unit_type[unit_index] == SHIP {
			sh = act.fighting_unit_ship[unit_index]
			if sh.special == NON_COMBATANT {
				n = 0
			}
		}
		act.shots_left[unit_index] = n
		total_shots += n
	}

	/* Handle all shots. */
	header_printed = FALSE
	combat_occurred = FALSE
	//int infiniteShotsGuard = total_shots;
	for total_shots > 0 {
		/* check to make sure we arent in infinite loop
		 * that usually happens when there are shots remaining
		 * but the side with the shots has no more ships left*/
		for i = 0; i < act.num_units_fighting; i++ {
			/* C casts every fighting_unit to ship_data here, even namplas.
			   For a nampla record, the C struct layout aliases:
			     ship->status  -> nampla->status (same offset)
			     ship->age     -> nampla->item_quantity[32] (GU9)
			     ship->special -> nampla->item_quantity[36] (unused item)
			   Reproduce those reads exactly. The pointer arrays keep their
			   original kind even if the unit was retagged GENOCIDE_NAMPLA
			   or BESIEGED_NAMPLA, so discriminate on the pointer kind. */
			var guard_age, guard_status, guard_special int
			if act.fighting_unit_ship[i] != nil {
				guard_age = act.fighting_unit_ship[i].age
				guard_status = act.fighting_unit_ship[i].status
				guard_special = act.fighting_unit_ship[i].special
			} else {
				guard_age = act.fighting_unit_nampla[i].item_quantity[32]
				guard_status = act.fighting_unit_nampla[i].status
				guard_special = act.fighting_unit_nampla[i].item_quantity[36]
			}
			if guard_age > 49 || guard_status == FORCED_JUMP ||
				guard_status == JUMPED_IN_COMBAT ||
				(guard_special == NON_COMBATANT && option != GERM_WARFARE) {
				total_shots -= act.shots_left[i]
				act.shots_left[i] = 0
			}
		}
		//// second test to prevent infinite loop due to the shot counter not being decremented.
		//if (total_shots > infiniteShotsGuard) {
		//    total_shots = infiniteShotsGuard;
		//}
		//infiniteShotsGuard--;

		/* Determine who fires next. */
		attacker_index = rnd(act.num_units_fighting) - 1
		if act.unit_type[attacker_index] == SHIP {
			attacking_ship = act.fighting_unit_ship[attacker_index]
			i = act.fighting_species_index[attacker_index]
			if field_distorted[i] == FALSE {
				ignore_field_distorters = TRUE
			} else {
				ignore_field_distorters = FALSE
			}
			attacker_name = ship_name(attacking_ship)
			ignore_field_distorters = FALSE

			/* Check if ship can fight. */
			if attacking_ship.age > 49 {
				continue
			}
			if attacking_ship.status == FORCED_JUMP {
				continue
			}
			if attacking_ship.status == JUMPED_IN_COMBAT {
				continue
			}
			if attacking_ship.special == NON_COMBATANT &&
				option != GERM_WARFARE {
				continue
			}
		} else {
			attacking_nampla = act.fighting_unit_nampla[attacker_index]
			if attacking_nampla != nil {
				attacker_name = fmt.Sprintf("PL %s", attacking_nampla.name)
				/* Check if planet still has defenses. */
				if attacking_nampla.item_quantity[PD] == 0 {
					continue
				}
			} else {
				/* This is a ship that was retagged GENOCIDE_NAMPLA or
				   BESIEGED_NAMPLA earlier this round. C reads the ship
				   record through a nampla cast: the name is at the same
				   offset, and nampla->item_quantity[PD] aliases the
				   ship's item_quantity[FS]. */
				attacker_name = fmt.Sprintf("PL %s", act.fighting_unit_ship[attacker_index].name)
				if act.fighting_unit_ship[attacker_index].item_quantity[FS] == 0 {
					continue
				}
			}
		}

		/* Make sure attacker is not someone who is being taken by surprise this round. */
		if act.surprised[attacker_index] != FALSE {
			continue
		}

		/* Find an enemy. */
		num_targets = 0
		i = act.fighting_species_index[attacker_index]
		attacker_ml = c_species[i].tech_level[ML]
		attacker_gv = c_species[i].tech_level[GV]
		for defender_index = 0; defender_index < act.num_units_fighting; defender_index++ {
			j = act.fighting_species_index[defender_index]
			if bat.enemy_mine[i][j] == FALSE {
				continue
			}

			if act.unit_type[defender_index] == SHIP {
				defending_ship = act.fighting_unit_ship[defender_index]
				if defending_ship.age > 49 {
					/* Already destroyed. */
					continue
				}
				if defending_ship.status == FORCED_JUMP {
					continue
				}
				if defending_ship.status == JUMPED_IN_COMBAT {
					continue
				}
				if defending_ship.special == NON_COMBATANT {
					continue
				}
			} else {
				defending_nampla = act.fighting_unit_nampla[defender_index]

				/* C reads defending_nampla->item_quantity[PD] before testing
				   the option; the tests are reordered here (identical
				   outcome) so that a ship retagged to GENOCIDE_NAMPLA or
				   BESIEGED_NAMPLA (nil nampla pointer; only possible when
				   option is not PLANET_ATTACK) is not dereferenced. */
				if option == PLANET_ATTACK && defending_nampla.item_quantity[PD] == 0 {
					continue
				}
			}

			target_index[num_targets] = defender_index
			num_targets++
		}

		if num_targets == 0 {
			/* Attacker has no enemies left. */
			total_shots -= act.shots_left[attacker_index]
			act.shots_left[attacker_index] = 0
			continue
		}

		/* Randomly choose a target. Choose the toughest of four. */
		defender_index = target_index[rnd(num_targets)-1]
		op1 = act.num_shots[defender_index] * act.weapon_damage[defender_index]
		di[0] = target_index[rnd(num_targets)-1]
		di[1] = target_index[rnd(num_targets)-1]
		di[2] = target_index[rnd(num_targets)-1]
		for i = 0; i < 3; i++ {
			op2 = act.num_shots[di[i]] * act.weapon_damage[di[i]]
			if op2 > op1 {
				defender_index = di[i]
				op1 = op2
			}
		}

		j = act.fighting_species_index[defender_index]
		defender_ml = c_species[j].tech_level[ML]

		if act.unit_type[defender_index] == SHIP {
			defending_ship = act.fighting_unit_ship[defender_index]
			if field_distorted[j] == FALSE {
				ignore_field_distorters = TRUE
			} else {
				ignore_field_distorters = FALSE
			}
			defender_name = ship_name(defending_ship)
			ignore_field_distorters = FALSE
		} else {
			defending_nampla = act.fighting_unit_nampla[defender_index]
			if defending_nampla != nil {
				defender_name = fmt.Sprintf("PL %s", defending_nampla.name)
			} else {
				/* Retagged ship; C reads the ship's name through the
				   nampla cast (same offset). */
				defender_name = fmt.Sprintf("PL %s", act.fighting_unit_ship[defender_index].name)
			}
		}

		/* Print round number. */
		if header_printed == FALSE {
			log_string("      Now doing round ")
			log_int(round_number)
			log_string(":\n")
			header_printed = TRUE
		}
		attackerGvMl := attacker_gv + attacker_ml
		if attackerGvMl <= 0 {
			attackerGvMl = 1
		}
		/* Check if attacker has any forced jump units.
		 * The attacker will place more emphasis on the use of these devices if he emphasizes gravitics technology over military technology. */
		fj_chance = 50 * attacker_gv / attackerGvMl
		if rnd(100) < fj_chance &&
			act.unit_type[attacker_index] == SHIP &&
			act.unit_type[defender_index] == SHIP {
			if forced_jump_units_used(attacker_index, defender_index, &total_shots, bat, act) != FALSE {
				combat_occurred = TRUE
				continue
			}
		}

		if act.shots_left[attacker_index] == 0 {
			continue
		}

		/* Since transports generally avoid combat, there is only a 10% chance that they will be targeted, unless they are being explicitly targeted. */
		i = act.fighting_species_index[attacker_index]
		j = act.fighting_species_index[defender_index]
		if act.unit_type[defender_index] == SHIP && defending_ship.class == TR &&
			bat.special_target[i] != TARGET_TRANSPORTS && rnd(10) != 5 {
			continue
		}

		/* If a special target has been specified, then there is a 75% chance that it will be attacked if it is available. */
		if bat.special_target[i] != 0 && rnd(100) < 76 {
			if bat.special_target[i] == TARGET_PDS {
				if act.unit_type[defender_index] != SHIP {
					goto fire
				} else {
					continue
				}
			}

			if act.unit_type[defender_index] != SHIP {
				continue
			}

			if bat.special_target[i] == TARGET_STARBASES && defending_ship.class != BA {
				continue
			}
			if bat.special_target[i] == TARGET_TRANSPORTS && defending_ship.class != TR {
				continue
			}
			if bat.special_target[i] == TARGET_WARSHIPS {
				if defending_ship.class == TR {
					continue
				}
				if defending_ship.class == BA {
					continue
				}
			}
		}

	fire:
		/* Update counts. */
		act.shots_left[attacker_index]--
		total_shots--

		/* Since transports generally avoid combat, there is only a 10% chance that they will attack. */
		if act.unit_type[attacker_index] == SHIP && attacking_ship.class == TR && option != GERM_WARFARE &&
			rnd(10) != 5 {
			continue
		}

		/* Fire! */
		combat_occurred = TRUE
		log_string("        ")
		log_string(attacker_name)
		log_string(" fires on ")
		log_string(defender_name)
		if act.unit_type[defender_index] == NAMPLA {
			log_string(" defenses")
		}

		combinedMl := attacker_ml + defender_ml
		if combinedMl <= 0 {
			combinedMl = 1
		}
		/* Get hit probability.
		 * The basic chance to hit is 1.5 times attackers ML over the sum of attacker's and defender's ML.
		 * Double this value if defender is surprised. */
		chance_to_hit = (150 * attacker_ml) / combinedMl
		if act.surprised[defender_index] != FALSE {
			chance_to_hit *= 2
			shields_up = FALSE
		} else {
			shields_up = TRUE
		}

		/* If defending ship is field-distorted, chance-to-hit is reduced by 25%. */
		j = act.fighting_species_index[defender_index]
		if act.unit_type[defender_index] == SHIP && field_distorted[j] != FALSE &&
			defending_ship.item_quantity[FD] == defending_ship.tonnage {
			chance_to_hit = (3 * chance_to_hit) / 4
		}
		if chance_to_hit > 98 {
			chance_to_hit = 98
		}
		if chance_to_hit < 2 {
			chance_to_hit = 2
		}

		/* Adjust for age. */
		if act.unit_type[attacker_index] == SHIP {
			chance_to_hit -= (2 * attacking_ship.age * chance_to_hit) / 100
		}

		/* Calculate damage that shot will do if it hits. */
		damage_done = act.weapon_damage[attacker_index]
		damage_done += ((26 - rnd(51)) * damage_done) / 100

		/* Take care of attempted annihilation and sieges. */
		if option == PLANET_BOMBARDMENT || option == GERM_WARFARE || option == SIEGE {
			/* Indicate the action that was attempted against this nampla. */
			if option == SIEGE {
				act.unit_type[defender_index] = BESIEGED_NAMPLA
			} else {
				act.unit_type[defender_index] = GENOCIDE_NAMPLA
			}

			/* Indicate who attacked who. */
			i = act.fighting_species_index[attacker_index]
			j = act.fighting_species_index[defender_index]
			x_attacked_y[i][j] = TRUE

			/* Update bombardment damage. */
			if option == PLANET_BOMBARDMENT {
				act.bomb_damage[defender_index] += damage_done
			} else if option == GERM_WARFARE {
				if act.unit_type[attacker_index] == SHIP {
					germ_bombs_used[i][j] += attacking_ship.item_quantity[GW]
					attacking_ship.item_quantity[GW] = 0
				} else if attacking_nampla != nil {
					germ_bombs_used[i][j] += attacking_nampla.item_quantity[GW]
					attacking_nampla.item_quantity[GW] = 0
				} else {
					/* Retagged ship attacker; C reads/writes the ship
					   record through the nampla cast:
					   nampla->item_quantity[GW] aliases the ship's
					   item_quantity[SG6]. */
					germ_bombs_used[i][j] += act.fighting_unit_ship[attacker_index].item_quantity[SG6]
					act.fighting_unit_ship[attacker_index].item_quantity[SG6] = 0
				}
			}

			continue
		}

		/* Check if shot hit. */
		if rnd(100) <= chance_to_hit {
			log_string(" and hits!\n")
		} else {
			log_string(" and misses!\n")
			continue
		}

		/* Subtract damage from defender's shields, if they're up. */
		damage_to_ship = 0
		if shields_up != FALSE {
			if act.unit_type[defender_index] == SHIP {
				damage_to_shields = (defending_ship.dest_y * damage_done) / 100
				damage_to_ship = damage_done - damage_to_shields
				act.shield_strength_left[defender_index] -= damage_to_shields

				/* Calculate percentage of shields left. */
				if act.shield_strength_left[defender_index] > 0 {
					defenderShieldStrength := act.shield_strength[defender_index]
					if defenderShieldStrength <= 0 {
						defenderShieldStrength = 1
					}
					defending_ship.dest_y =
						(100 * act.shield_strength_left[defender_index]) / defenderShieldStrength
				} else {
					defending_ship.dest_y = 0
				}
			} else {
				/* Planetary defenses. */
				act.shield_strength_left[defender_index] -= damage_done
			}
		}

		/* See if it got through shields. */
		units_destroyed = 0
		percent_decrease = 0
		if shields_up == FALSE || act.shield_strength_left[defender_index] < 0 || damage_to_ship > 0 {
			/* Get net damage to ship or PDs. */
			if shields_up != FALSE {
				if act.unit_type[defender_index] == SHIP {
					/* Total damage to ship is direct damage plus damage that shields could not absorb. */
					damage_done = damage_to_ship
					if act.shield_strength_left[defender_index] < 0 {
						damage_done -= act.shield_strength_left[defender_index]
					}
				} else {
					damage_done = -act.shield_strength_left[defender_index]
				}
			}

			defenderShieldStrength := act.shield_strength[defender_index]
			if defenderShieldStrength <= 0 {
				defenderShieldStrength = 1
			}

			percent_decrease = (50 * damage_done) / defenderShieldStrength

			percent_decrease += ((rnd(51) - 26) * percent_decrease) / 100
			if percent_decrease > 100 {
				percent_decrease = 100
			}

			if act.unit_type[defender_index] == SHIP {
				defending_ship.age += percent_decrease / 2
				if defending_ship.age > 49 {
					units_destroyed = 1
				} else {
					units_destroyed = 0
				}
			} else {
				units_destroyed = (percent_decrease * act.original_age_or_PDs[defender_index]) / 100
				if units_destroyed > defending_nampla.item_quantity[PD] {
					units_destroyed = defending_nampla.item_quantity[PD]
				}
				if units_destroyed < 1 {
					units_destroyed = 1
				}
				defending_nampla.item_quantity[PD] -= units_destroyed
			}

			if act.shield_strength_left[defender_index] < 0 {
				act.shield_strength_left[defender_index] = 0
			}
		}

		/* See if this is a hijacking. */
		i = act.fighting_species_index[attacker_index]
		j = act.fighting_species_index[defender_index]
		if bat.enemy_mine[i][j] == 2 && (option == DEEP_SPACE_FIGHT || option == PLANET_ATTACK) {
			this_is_a_hijacking = TRUE
		} else {
			this_is_a_hijacking = FALSE
		}

		attacking_species = c_species[i]
		defending_species = c_species[j]

		/* Report if anything was destroyed. */
		FDs_were_destroyed = FALSE
		if units_destroyed != 0 {
			if act.unit_type[defender_index] == SHIP {
				log_summary = TRUE
				log_string("        ")
				log_string(defender_name)
				if this_is_a_hijacking != FALSE {
					log_string(" was successfully hijacked and will generate ")

					if defending_ship.class == TR || defending_ship.ship_type == STARBASE {
						original_cost = ship_cost[defending_ship.class] * defending_ship.tonnage
					} else {
						original_cost = ship_cost[defending_ship.class]
					}

					if defending_ship.ship_type == SUB_LIGHT {
						original_cost = (3 * original_cost) / 4
					}

					if defending_ship.status == UNDER_CONSTRUCTION {
						recycle_value = (original_cost - defending_ship.remaining_cost) / 2
					} else {
						recycle_value = (3 * original_cost * (60 - act.original_age_or_PDs[defender_index])) / 200
					}

					economic_units = recycle_value

					for i = 0; i < MAX_ITEMS; i++ {
						j = defending_ship.item_quantity[i]
						if j > 0 {
							if i == TP {
								techLevel_2x := 2 * defending_species.tech_level[BI]
								if techLevel_2x <= 0 {
									techLevel_2x = 1
								}
								recycle_value = (j * item_cost[i]) / techLevel_2x
							} else if i == RM {
								recycle_value = j / 5
							} else {
								recycle_value = (j * item_cost[i]) / 2
							}

							economic_units += recycle_value
						}
					}

					attacking_species.econ_units += economic_units

					log_long(economic_units)
					log_string(" economic units for the hijackers.\n")
				} else {
					log_string(" was destroyed.\n")
				}

				for i = 0; i < MAX_ITEMS; i++ {
					if defending_ship.item_quantity[i] > 0 {
						/* If this is a hijacking of a field-distorted ship, we want the true name of the hijacked species to be announced, but we don't want any cargo to be destroyed. */
						if i == FD {
							FDs_were_destroyed = TRUE
						}
						if this_is_a_hijacking == FALSE {
							defending_ship.item_quantity[FD] = 0
						}
					}
				}
				log_to_file = FALSE
				if this_is_a_hijacking != FALSE {
					log_string("          The hijacker was ")
				} else {
					log_string("          The killing blow was delivered by ")
				}
				log_string(attacker_name)
				log_string(".\n")
				log_to_file = TRUE
				log_summary = FALSE

				total_shots -= act.shots_left[defender_index]
				act.shots_left[defender_index] = 0
				act.num_shots[defender_index] = 0
			} else {
				log_summary = TRUE
				log_string("        ")
				log_int(units_destroyed)
				if units_destroyed > 1 {
					log_string(" PDs on PL ")
				} else {
					log_string(" PD on PL ")
				}
				log_string(defending_nampla.name)
				if units_destroyed > 1 {
					log_string(" were destroyed by ")
				} else {
					log_string(" was destroyed by ")
				}

				log_string(attacker_name)
				log_string(".\n")

				if defending_nampla.item_quantity[PD] == 0 {
					total_shots -= act.shots_left[defender_index]
					act.shots_left[defender_index] = 0
					act.num_shots[defender_index] = 0
					log_string("        All planetary defenses have been destroyed on ")
					log_string(defender_name)
					log_string("!\n")
				}
				log_summary = FALSE
			}
		} else if percent_decrease > 0 && this_is_a_hijacking == FALSE && act.unit_type[defender_index] == SHIP {
			/* See if anything carried by the ship was also destroyed. */
			for i = 0; i < MAX_ITEMS; i++ {
				j = defending_ship.item_quantity[i]
				if j > 0 {
					j = (percent_decrease * j) / 100
					if j > 0 {
						defending_ship.item_quantity[i] -= j
						if i == FD {
							FDs_were_destroyed = TRUE
						}
					}
				}
			}
		}

		j = act.fighting_species_index[defender_index]
		if FDs_were_destroyed != FALSE && field_distorted[j] != FALSE && defending_ship.dest_x == 0 {
			/* Reveal the true name of the ship and the owning species. */
			log_summary = TRUE
			if this_is_a_hijacking != FALSE {
				log_string("        Hijacking of ")
			} else {
				log_string("        Damage to ")
			}
			log_string(defender_name)
			log_string(" caused collapse of distortion field. Real name of ship is ")
			log_string(ship_name(defending_ship))
			log_string(" owned by SP ")
			log_string(defending_species.name)
			log_string(".\n")
			log_summary = FALSE
			defending_ship.dest_x = 127 /* Ship is now exposed. */
		}
	}

	/* No more surprises. */
	for i = 0; i < act.num_units_fighting; i++ {
		act.surprised[i] = FALSE
	}

	return combat_occurred
}

func do_siege(bat *battle_data_t, act *action_data_t) {
	var a, d, i, attacker_index, defender_index, attacking_species_number, defending_species_number int
	var defending_nampla *nampla_data_t
	var attacking_ship *ship_data_t
	var defending_species, attacking_species *species_data_t

	for defender_index = 0; defender_index < act.num_units_fighting; defender_index++ {
		if act.unit_type[defender_index] == BESIEGED_NAMPLA {
			defending_nampla = act.fighting_unit_nampla[defender_index]
			defending_nampla.siege_eff = TRUE
			d = act.fighting_species_index[defender_index]
			defending_species = c_species[d]
			defending_species_number = bat.spec_num[d]
			for attacker_index = 0; attacker_index < act.num_units_fighting; attacker_index++ {
				if act.unit_type[attacker_index] == SHIP {
					attacking_ship = act.fighting_unit_ship[attacker_index]
					a = act.fighting_species_index[attacker_index]
					if x_attacked_y[a][d] != FALSE {
						attacking_species = c_species[a]
						attacking_species_number = bat.spec_num[a]
						/* Check if there's enough memory for a new interspecies transaction. */
						if num_transactions == MAX_TRANSACTIONS {
							fmt.Fprintf(os.Stderr, "\nRan out of memory! MAX_TRANSACTIONS is too small!\n\n")
							os.Exit(255)
						}
						i = num_transactions
						num_transactions++
						/* Define this transaction. */
						transaction[i].trans_type = BESIEGE_PLANET
						transaction[i].x = defending_nampla.x
						transaction[i].y = defending_nampla.y
						transaction[i].z = defending_nampla.z
						transaction[i].pn = defending_nampla.pn
						transaction[i].number1 = attacking_species_number
						transaction[i].name1 = attacking_species.name
						transaction[i].number2 = defending_species_number
						transaction[i].name2 = defending_species.name
						transaction[i].name3 = attacking_ship.name
					}
				}
			}
		}
	}
	log_string("      Only those ships that actually remain in the system will take part in the siege.\n")
}

/*
The following routine will fill "act" with ship and nampla data necessary

	for an action; i.e., number of shots per round, damage done per shot,
	total shield power, etc. Note that this routine always restores shields
	completely. It is assumed that a sufficient number of rounds passes
	between actions of a battle to completely regenerate shields.

	The routine will return TRUE if the action can take place, otherwise FALSE.
*/
func fighting_params(option, location int, bat *battle_data_t, act *action_data_t) int {
	var x, y, z int
	var i, j, found, utype, num_sp, unit_index int
	var species_index int
	var ship_index int
	var nampla_index int
	var sp1, sp2, use_this_ship, n_shots int
	var engage_option, engage_location, attacking_ships_here int
	var defending_ships_here, attacking_pds_here, defending_pds_here int
	var num_fighting_units int
	var tons int
	var ml, ls, unit_power, offensive_power, defensive_power int
	var sh *ship_data_t
	var nam *nampla_data_t

	/* Add fighting units to "act" arrays. At the same time, check if
	   a fight of the current option type will occur at the current
	   location. */
	num_fighting_units = 0
	x = bat.x
	y = bat.y
	z = bat.z
	attacking_ML = 0
	defending_ML = 0
	attacking_ships_here = FALSE
	defending_ships_here = FALSE
	attacking_pds_here = FALSE
	defending_pds_here = FALSE
	deep_space_defense = FALSE
	num_sp = bat.num_species_here

	for species_index = 0; species_index < num_sp; species_index++ {
		/* Check which ships can take part in fight. */
		for ship_index = 0; ship_index < c_species[species_index].num_ships; ship_index++ {
			sh = c_ship[species_index][ship_index]
			use_this_ship = FALSE

			if sh.x != x {
				continue
			}
			if sh.y != y {
				continue
			}
			if sh.z != z {
				continue
			}
			if sh.pn == 99 {
				continue
			}
			if sh.age > 49 {
				continue
			}
			if sh.status == UNDER_CONSTRUCTION {
				continue
			}
			if sh.status == FORCED_JUMP {
				continue
			}
			if sh.status == JUMPED_IN_COMBAT {
				continue
			}
			if sh.class == TR && sh.pn != location && option != GERM_WARFARE {
				continue
			}
			if disbanded_species_ship(species_index, sh) != FALSE {
				continue
			}
			if option == SIEGE || option == PLANET_BOMBARDMENT {
				if sh.special == NON_COMBATANT {
					continue
				}
			}

			for i = 0; i < bat.num_engage_options[species_index]; i++ {
				engage_option = bat.engage_option[species_index][i]
				engage_location = bat.engage_planet[species_index][i]

				switch engage_option {
				case DEFENSE_IN_PLACE:
					if sh.pn != location {
						break
					}
					defending_ships_here = TRUE
					use_this_ship = TRUE

				case DEEP_SPACE_DEFENSE:
					if option != DEEP_SPACE_FIGHT {
						break
					}
					if sh.class == BA && sh.pn != 0 {
						break
					}
					defending_ships_here = TRUE
					use_this_ship = TRUE
					deep_space_defense = TRUE
					if c_species[species_index].tech_level[ML] > defending_ML {
						defending_ML = c_species[species_index].tech_level[ML]
					}

				case PLANET_DEFENSE:
					if location != engage_location {
						break
					}
					if sh.class == BA && sh.pn != location {
						break
					}
					defending_ships_here = TRUE
					use_this_ship = TRUE

				case DEEP_SPACE_FIGHT:
					if option != DEEP_SPACE_FIGHT {
						break
					}
					if sh.class == BA && sh.pn != 0 {
						break
					}
					if c_species[species_index].tech_level[ML] > defending_ML {
						defending_ML = c_species[species_index].tech_level[ML]
					}
					defending_ships_here = TRUE
					attacking_ships_here = TRUE
					use_this_ship = TRUE

				case PLANET_ATTACK, PLANET_BOMBARDMENT, GERM_WARFARE, SIEGE:
					if sh.class == BA && sh.pn != location {
						break
					}
					if sh.class == TR && option == SIEGE {
						break
					}
					if option == DEEP_SPACE_FIGHT {
						/* There are two possibilities here: 1. outsiders
						   are attacking locals, or 2. locals are attacking
						   locals. If (1), we want outsiders to first fight
						   in deep space. If (2), locals will not first
						   fight in deep space (unless other explicit
						   orders were given). The case is (2) if current
						   species has a planet here. */

						found = FALSE
						for nampla_index = 0; nampla_index < c_species[species_index].num_namplas; nampla_index++ {
							nam = c_nampla[species_index][nampla_index]

							if nam.x != x {
								continue
							}
							if nam.y != y {
								continue
							}
							if nam.z != z {
								continue
							}
							if (nam.status & POPULATED) == 0 {
								continue
							}

							found = TRUE
							break
						}

						if found == FALSE {
							attacking_ships_here = TRUE
							use_this_ship = TRUE
							if c_species[species_index].tech_level[ML] > attacking_ML {
								attacking_ML = c_species[species_index].tech_level[ML]
							}
							break
						}
					}
					if option != engage_option &&
						option != PLANET_ATTACK {
						break
					}
					if location != engage_location {
						break
					}
					attacking_ships_here = TRUE
					use_this_ship = TRUE

				default:
					fmt.Fprintf(os.Stderr, "\n\n\tInternal error #1 in fighting_params - invalid engage option!\n\n")
					os.Exit(255)
				}
			}

			/* add_ship: */
			if use_this_ship != FALSE {
				/* Add data for this ship to action array. */
				act.fighting_species_index[num_fighting_units] = species_index
				act.unit_type[num_fighting_units] = SHIP
				act.fighting_unit_ship[num_fighting_units] = sh
				act.fighting_unit_nampla[num_fighting_units] = nil
				act.original_age_or_PDs[num_fighting_units] = sh.age
				num_fighting_units++
			}
		}

		/* Check which namplas can take part in fight. */
		for nampla_index = 0; nampla_index < c_species[species_index].num_namplas; nampla_index++ {
			nam = c_nampla[species_index][nampla_index]

			if nam.x != x {
				continue
			}
			if nam.y != y {
				continue
			}
			if nam.z != z {
				continue
			}
			if nam.pn != location {
				continue
			}
			if (nam.status & POPULATED) == 0 {
				continue
			}
			if nam.status&DISBANDED_COLONY != 0 {
				continue
			}

			/* This planet has been targeted for some kind of attack. In
			   most cases, one species will attack a planet inhabited by
			   another species. However, it is also possible for two or
			   more species to have colonies on the SAME planet, and for
			   one to attack the other. */

			for i = 0; i < bat.num_engage_options[species_index]; i++ {
				engage_option = bat.engage_option[species_index][i]
				engage_location = bat.engage_planet[species_index][i]
				if engage_location != location {
					continue
				}

				switch engage_option {
				case DEFENSE_IN_PLACE, DEEP_SPACE_DEFENSE, PLANET_DEFENSE, DEEP_SPACE_FIGHT:

				case PLANET_ATTACK, PLANET_BOMBARDMENT, GERM_WARFARE, SIEGE:
					if option != engage_option &&
						option != PLANET_ATTACK {
						break
					}
					if nam.item_quantity[PD] > 0 {
						attacking_pds_here = TRUE
					}

				default:
					fmt.Fprintf(os.Stderr, "\n\n\tInternal error #2 in fighting_params - invalid engage option!\n\n")
					os.Exit(255)
				}
			}

			if nam.item_quantity[PD] > 0 {
				defending_pds_here = TRUE
			}

			/* Add data for this nampla to action array. */
			act.fighting_species_index[num_fighting_units] = species_index
			act.unit_type[num_fighting_units] = NAMPLA
			act.fighting_unit_ship[num_fighting_units] = nil
			act.fighting_unit_nampla[num_fighting_units] = nam
			act.original_age_or_PDs[num_fighting_units] = nam.item_quantity[PD]
			num_fighting_units++
		}
	}

	/* Depending on option, see if the right combination of combatants
	   are present. */
	switch option {
	case DEEP_SPACE_FIGHT:
		if attacking_ships_here == FALSE || defending_ships_here == FALSE {
			return FALSE
		}

	case PLANET_ATTACK, PLANET_BOMBARDMENT:
		if attacking_ships_here == FALSE && attacking_pds_here == FALSE {
			return FALSE
		}

	case SIEGE, GERM_WARFARE:
		if attacking_ships_here == FALSE {
			return FALSE
		}

	default:
		fmt.Fprintf(os.Stderr, "\n\n\tInternal error #3 in fighting_params - invalid engage option!\n\n")
		os.Exit(255)
	}
	_ = defending_pds_here

	/* There is at least one attacker and one defender here. See if they
	   are enemies. */
	for i = 0; i < num_fighting_units; i++ {
		sp1 = act.fighting_species_index[i]
		for j = 0; j < num_fighting_units; j++ {
			sp2 = act.fighting_species_index[j]
			if bat.enemy_mine[sp1][sp2] != FALSE {
				goto next_step
			}
		}
	}

	return FALSE

next_step:

	act.num_units_fighting = num_fighting_units

	/* Determine number of shots, shield power and weapons power for
	   all combatants. */
	for unit_index = 0; unit_index < act.num_units_fighting; unit_index++ {
		utype = act.unit_type[unit_index]
		if utype == SHIP {
			sh = act.fighting_unit_ship[unit_index]
			tons = sh.tonnage
		} else {
			nam = act.fighting_unit_nampla[unit_index]
			tons = nam.item_quantity[PD] / 200
			if tons < 1 && nam.item_quantity[PD] > 0 {
				tons = 1
			}
		}

		species_index = act.fighting_species_index[unit_index]

		unit_power = power(tons)
		offensive_power = unit_power
		defensive_power = unit_power

		if utype == SHIP {
			if sh.class == TR {
				/* Transports are not designed for combat. */
				offensive_power /= 10
				defensive_power /= 10
			} else if sh.class != BA {
				/* Add auxiliary shield generator contribution, if any. */
				tons = 5
				for i = SG1; i <= SG9; i++ {
					if sh.item_quantity[i] > 0 {
						defensive_power += sh.item_quantity[i] * power(tons)
					}
					tons += 5
				}

				/* Add auxiliary gun unit contribution, if any. */
				tons = 5
				for i = GU1; i <= GU9; i++ {
					if sh.item_quantity[i] > 0 {
						offensive_power += sh.item_quantity[i] * power(tons)
					}
					tons += 5
				}
			}

			/* Adjust for ship aging. */
			offensive_power -= (sh.age * offensive_power) / 50
			defensive_power -= (sh.age * defensive_power) / 50
		}

		/* Adjust values for tech levels. */
		ml = c_species[species_index].tech_level[ML]
		ls = c_species[species_index].tech_level[LS]
		offensive_power += (ml * offensive_power) / 50
		defensive_power += (ls * defensive_power) / 50

		/* Adjust values if this species is hijacking anyone. */
		if bat.hijacker[species_index] != FALSE && (option == DEEP_SPACE_FIGHT ||
			option == PLANET_ATTACK) {
			offensive_power /= 4
			defensive_power /= 4
		}

		/* Get number of shots per round. */
		n_shots = (offensive_power / 1500) + 1
		if ml == 0 || offensive_power == 0 {
			n_shots = 0
		}
		if n_shots > 5 {
			n_shots = 5
		}
		act.num_shots[unit_index] = n_shots
		act.shots_left[unit_index] = n_shots

		/* Get damage per shot. */
		if n_shots > 0 {
			act.weapon_damage[unit_index] = (2 * offensive_power) / n_shots
		} else {
			act.weapon_damage[unit_index] = 0
		}

		/* Do defensive shields. */
		act.shield_strength[unit_index] = defensive_power
		if utype == SHIP {
			/* Adjust for results of previous action, if any. "dest_y"
			   contains the percentage of shields that remained at end
			   of last action. */
			defensive_power = (sh.dest_y * defensive_power) / 100
		}
		act.shield_strength_left[unit_index] = defensive_power

		/* Set bomb damage to zero in case this is planet bombardment or
		   germ warfare. */
		act.bomb_damage[unit_index] = 0

		/* Set flag for individual unit if species can be surprised. */
		if bat.can_be_surprised[species_index] != FALSE {
			act.surprised[unit_index] = TRUE
		} else {
			act.surprised[unit_index] = FALSE
		}
	}

	return TRUE /* There will be a fight here. */
}

/* This routine will return TRUE if forced jump or misjump units are used, even if they fail.
 * It will return FALSE if the attacker has none or not enough. */
func forced_jump_units_used(attacker_index, defender_index int, total_shots *int, bat *battle_data_t, act *action_data_t) int {
	var i, att_sp_index, def_sp_index, attacker_gv, defender_gv int
	var item_type, fj_num, fm_num, number, success_chance, failure int
	var x, y, z int
	var attacking_ship, defending_ship *ship_data_t

	/* Make sure attacking unit is a starbase. */
	attacking_ship = act.fighting_unit_ship[attacker_index]
	if attacking_ship.ship_type != STARBASE {
		return FALSE
	}
	/* See if attacker has any forced jump units. */
	fj_num = attacking_ship.item_quantity[FJ]
	fm_num = attacking_ship.item_quantity[FM]
	if fj_num == 0 && fm_num == 0 {
		return FALSE
	}
	/* If both types are being carried, choose one randomly. */
	if fj_num > 0 && fm_num > 0 {
		if rnd(2) == 1 {
			item_type = FJ
			number = fj_num
		} else {
			item_type = FM
			number = fm_num
		}
	} else if fj_num > 0 {
		item_type = FJ
		number = fj_num
	} else {
		item_type = FM
		number = fm_num
	}

	/* Get gravitics tech levels. */
	att_sp_index = act.fighting_species_index[attacker_index]
	attacker_gv = c_species[att_sp_index].tech_level[GV]

	def_sp_index = act.fighting_species_index[defender_index]
	defender_gv = c_species[def_sp_index].tech_level[GV]

	/* Check if sufficient units are available. */
	defending_ship = act.fighting_unit_ship[defender_index]
	if number < defending_ship.tonnage {
		return FALSE
	}

	/* Make sure defender is not a starbase. */
	if defending_ship.ship_type == STARBASE {
		return FALSE
	}

	/* Calculate percent chance of success. */
	success_chance = 2 * ((number - defending_ship.tonnage) + (attacker_gv - defender_gv))

	/* See if it worked. */
	if rnd(100) > success_chance {
		failure = TRUE
	} else {
		failure = FALSE
	}

	if failure != FALSE {
		log_summary = FALSE
	} else {
		log_summary = TRUE
	}

	log_string("        ")
	log_string(ship_name(attacking_ship))
	log_string(" attempts to use ")
	log_string(item_name[item_type])
	log_string("s against ")

	if field_distorted[def_sp_index] == FALSE {
		ignore_field_distorters = TRUE
	} else {
		ignore_field_distorters = FALSE
	}
	log_string(ship_name(defending_ship))
	ignore_field_distorters = FALSE

	if failure != FALSE {
		log_string(", but fails.\n")
		return TRUE
	}

	log_string(", and succeeds!\n")
	log_summary = FALSE

	/* Determine destination. */
	if item_type == FM {
		/* Destination is totally random. */
		x = rnd(100) - 1
		y = rnd(100) - 1
		z = rnd(100) - 1
	} else {
		/* Random location close to battle. */
		i = 3
		for i == 3 {
			i = rnd(5)
		}
		x = bat.x + i - 3
		if x < 0 {
			x = 0
		}

		i = 3
		for i == 3 {
			i = rnd(5)
		}
		y = bat.y + i - 3
		if y < 0 {
			y = 0
		}

		i = 3
		for i == 3 {
			i = rnd(5)
		}
		z = bat.z + i - 3
		if z < 0 {
			z = 0
		}
	}
	defending_ship.dest_x = x
	defending_ship.dest_y = y
	defending_ship.dest_z = z

	/* Make sure this ship can no longer take part in the battle. */
	defending_ship.status = FORCED_JUMP
	defending_ship.pn = -1
	*total_shots -= act.shots_left[defender_index]
	act.shots_left[defender_index] = 0
	act.num_shots[defender_index] = 0

	return TRUE
}

func regenerate_shields(act *action_data_t) {
	var species_index, unit_index int
	var ls, max_shield_strength, percent int
	/* Shields are regenerated by 5 + LS/10 percent per round. */
	for unit_index = 0; unit_index < act.num_units_fighting; unit_index++ {
		species_index = act.fighting_species_index[unit_index]
		ls = c_species[species_index].tech_level[LS]
		max_shield_strength = act.shield_strength[unit_index]
		percent = (ls / 10) + 5
		act.shield_strength_left[unit_index] += (percent * max_shield_strength) / 100
		if act.shield_strength_left[unit_index] > max_shield_strength {
			act.shield_strength_left[unit_index] = max_shield_strength
		}
	}
}

/* This routine will check all fighting ships and see if any wish to
 * withdraw. If so, it will set the ship's status to JUMPED_IN_COMBAT.
 * The actual jump will be handled by the Jump program. */
func withdrawal_check(bat *battle_data_t, act *action_data_t) {
	var i, old_trunc int
	var ship_index int
	var species_index int
	var percent_loss int
	var num_ships_gone, num_ships_total [MAX_SPECIES]int
	var withdraw_age int
	var sh *ship_data_t

	for i = 0; i < MAX_SPECIES; i++ {
		num_ships_gone[i] = 0
		num_ships_total[i] = 0
	}

	old_trunc = truncate_name /* Show age of ship here. */
	truncate_name = FALSE

	/* Compile statistics and handle individual ships that must leave. */
	for ship_index = 0; ship_index < act.num_units_fighting; ship_index++ {
		if act.unit_type[ship_index] != SHIP {
			continue
		}

		sh = act.fighting_unit_ship[ship_index]
		species_index = act.fighting_species_index[ship_index]
		num_ships_total[species_index]++

		if sh.status == JUMPED_IN_COMBAT {
			/* Already withdrawn. */
			num_ships_gone[species_index]++
			continue
		}
		if sh.status == FORCED_JUMP {
			/* Forced to leave. */
			num_ships_gone[species_index]++
			continue
		}
		if sh.age > 49 {
			/* Already destroyed. */
			num_ships_gone[species_index]++
			continue
		}
		if sh.ship_type != FTL {
			continue
		} /* Ship can't jump. */

		if sh.class == TR {
			withdraw_age = bat.transport_withdraw_age[species_index]
			if withdraw_age == 0 {
				/* Transports will withdraw only when entire fleet withdraws. */
				continue
			}
		} else {
			withdraw_age = bat.warship_withdraw_age[species_index]
		}

		if sh.age > withdraw_age {
			act.num_shots[ship_index] = 0
			act.shots_left[ship_index] = 0
			sh.pn = 0

			if field_distorted[species_index] == FALSE {
				ignore_field_distorters = TRUE
			} else {
				ignore_field_distorters = FALSE
			}

			fmt.Fprintf(log_file, "        %s jumps away from the battle.\n", ship_name(sh))
			fmt.Fprintf(summary_file, "        %s jumps away from the battle.\n", ship_name(sh))

			ignore_field_distorters = FALSE

			sh.dest_x = bat.haven_x[species_index]
			sh.dest_y = bat.haven_y[species_index]
			sh.dest_z = bat.haven_z[species_index]

			sh.status = JUMPED_IN_COMBAT

			num_ships_gone[species_index]++
		}
	}

	/* Now check if a fleet has reached its limit. */
	for ship_index = 0; ship_index < act.num_units_fighting; ship_index++ {
		if act.unit_type[ship_index] != SHIP {
			continue
		}

		sh = act.fighting_unit_ship[ship_index]
		species_index = act.fighting_species_index[ship_index]

		if sh.ship_type != FTL {
			/* Ship can't jump. */
			continue
		}
		if sh.status == JUMPED_IN_COMBAT {
			/* Already withdrawn. */
			continue
		}
		if sh.status == FORCED_JUMP {
			/* Already gone. */
			continue
		}
		if sh.age > 49 {
			/* Already destroyed. */
			continue
		}

		if bat.fleet_withdraw_percentage[species_index] == 0 {
			percent_loss = 101 /* Always withdraw immediately. */
		} else {
			percent_loss = (100 * num_ships_gone[species_index]) / num_ships_total[species_index]
		}

		if percent_loss > bat.fleet_withdraw_percentage[species_index] {
			act.num_shots[ship_index] = 0
			act.shots_left[ship_index] = 0
			sh.pn = 0

			if field_distorted[species_index] == FALSE {
				ignore_field_distorters = TRUE
			} else {
				ignore_field_distorters = FALSE
			}

			fmt.Fprintf(log_file, "        %s jumps away from the battle.\n", ship_name(sh))
			fmt.Fprintf(summary_file, "        %s jumps away from the battle.\n", ship_name(sh))

			ignore_field_distorters = FALSE

			sh.dest_x = bat.haven_x[species_index]
			sh.dest_y = bat.haven_y[species_index]
			sh.dest_z = bat.haven_z[species_index]

			sh.status = JUMPED_IN_COMBAT
		}
	}

	truncate_name = old_trunc
}
