package game

// Port of intercept.c (and the intercept_t type plus globals from
// intercept.h, which this module owns).

import (
	"fmt"
	"os"
)

const MAX_INTERCEPTS = 1000

const MAX_ENEMY_SHIPS = 400

// intercept_t is from intercept.h.
type intercept_t struct {
	x, y, z      int
	amount_spent int
}

var num_intercepts int

var intercept [MAX_INTERCEPTS]intercept_t

// resetInterceptState restores this module's file-scope globals to their
// initial values. ResetState in vars.go does not call this; tests or
// drivers that re-run combat in one process must call it themselves.
func resetInterceptState() {
	num_intercepts = 0
	intercept = [MAX_INTERCEPTS]intercept_t{}
}

func handle_intercept(intercept_index int) {
	var i, j, n, num_enemy_ships, alien_index, enemy_index, enemy_num, num_ships_left, array_index, bit_number, is_an_enemy, is_distorted int
	var enemy_number [MAX_ENEMY_SHIPS]int
	var bit_mask, cost_to_destroy int
	var alien *species_data_t
	var alien_sh, enemy_sh *ship_data_t
	var enemy_ship [MAX_ENEMY_SHIPS]*ship_data_t

	/* Make a list of all enemy ships that jumped into this system. */
	num_enemy_ships = 0
	for alien_index = 0; alien_index < galaxy.num_species; alien_index++ {
		if data_in_memory[alien_index] == FALSE {
			continue
		}

		if species_number == alien_index+1 {
			continue
		}

		/* Is it an enemy species? */
		array_index = alien_index / 32
		bit_number = alien_index % 32
		bit_mask = 1 << bit_number
		if species.enemy[array_index]&uint32(bit_mask) != 0 {
			is_an_enemy = TRUE
		} else {
			is_an_enemy = FALSE
		}

		/* Find enemy ships, if any, that jumped to this location. */
		alien = &spec_data[alien_index]
		for i = 0; i < alien.num_ships; i++ {
			alien_sh = ship_data[alien_index][i]

			if alien_sh.pn == 99 {
				continue
			}

			/* Did it jump this turn? */
			if alien_sh.just_jumped == FALSE {
				continue
			}
			if alien_sh.just_jumped == 50 {
				continue /* Ship MOVEd. */
			}

			/* Did it enter this star system? */
			if alien_sh.x != intercept[intercept_index].x {
				continue
			}
			if alien_sh.y != intercept[intercept_index].y {
				continue
			}
			if alien_sh.z != intercept[intercept_index].z {
				continue
			}

			/* Is it field-distorted? */
			if alien_sh.item_quantity[FD] == alien_sh.tonnage {
				is_distorted = TRUE
			} else {
				is_distorted = FALSE
			}

			if is_an_enemy == FALSE && is_distorted == FALSE {
				continue
			}

			/* This is an enemy ship that just jumped into the system. */
			if num_enemy_ships == MAX_ENEMY_SHIPS {
				fmt.Fprintf(os.Stderr, "\n\tERROR! Array overflow in handle_intercept!\n\n")
				os.Exit(255)
			}
			enemy_number[num_enemy_ships] = alien_index + 1
			enemy_ship[num_enemy_ships] = alien_sh
			num_enemy_ships++
		}
	}

	if num_enemy_ships == 0 {
		return /* Nothing to intercept. */
	}

	num_ships_left = num_enemy_ships
	for num_ships_left > 0 {
		/* Select ship for interception. */
		enemy_index = rnd(num_enemy_ships) - 1
		if enemy_ship[enemy_index] == nil {
			continue /* We already did this one. */
		}
		enemy_num = enemy_number[enemy_index]
		enemy_sh = enemy_ship[enemy_index]

		/* Are there enough funds to destroy this ship? */
		cost_to_destroy = 100 * enemy_sh.tonnage
		if enemy_sh.class == TR {
			cost_to_destroy /= 10
		}
		if cost_to_destroy > intercept[intercept_index].amount_spent {
			break
		}

		/* Is the ship too large? Check only if ship did NOT arrive via a natural wormhole. */
		if enemy_sh.just_jumped != 99 {
			if enemy_sh.tonnage > 20 {
				break
			}
			if enemy_sh.class != TR && enemy_sh.tonnage > 5 {
				break
			}
		}

		/* Update funds available. */
		intercept[intercept_index].amount_spent -= cost_to_destroy

		/* Log the result for current species. */
		log_string("\n! ")
		n = enemy_sh.item_quantity[FD] /* Show real name. */
		enemy_sh.item_quantity[FD] = 0
		log_string(ship_name(enemy_sh))
		enemy_sh.item_quantity[FD] = n

		/* List cargo destroyed. */
		n = 0
		for j = 0; j < MAX_ITEMS; j++ {
			if enemy_sh.item_quantity[j] > 0 {
				if n == 0 {
					log_string(" (cargo: ")
				} else {
					log_char(',')
				}
				n++
				log_int(enemy_sh.item_quantity[j])
				log_char(' ')
				log_string(item_abbr[j])
			}
		}
		if n > 0 {
			log_char(')')
		}

		log_string(", owned by SP ")
		log_string(spec_data[enemy_num-1].name)
		log_string(", was successfully intercepted and destroyed in sector ")
		log_int(enemy_sh.x)
		log_char(' ')
		log_int(enemy_sh.y)
		log_char(' ')
		log_int(enemy_sh.z)
		log_string(".\n")

		/* Create interspecies transaction so that other player will be notified. */
		if num_transactions == MAX_TRANSACTIONS {
			fmt.Fprintf(os.Stderr, "\n\n\tERROR! num_transactions > MAX_TRANSACTIONS in handle_intercept!\n\n")
			os.Exit(255)
		}

		n = num_transactions
		num_transactions++
		transaction[n].trans_type = SHIP_MISHAP
		transaction[n].value = 1 /* Interception. */
		transaction[n].number1 = enemy_number[enemy_index]
		transaction[n].name1 = ship_name(enemy_sh)

		delete_ship(enemy_sh)

		enemy_ship[enemy_index] = nil /* Don't select this ship again. */

		num_ships_left--
	}
}
