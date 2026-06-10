package game

import (
	"fmt"
	"os"
)

// Port of ship.c.

/* Look-up table for ship defensive/offensive power uses ship->tonnage as an index.
 * Each value is equal to 100 * (ship->tonnage)^1.2.
 * The 'power' subroutine uses recursion to calculate values for tonnages over 100. */
var ship_power = [101]int{
	0, /* Zeroth element not used. */
	100, 230, 374, 528, 690, 859, 1033, 1213, 1397, 1585,
	1777, 1973, 2171, 2373, 2578, 2786, 2996, 3209, 3424, 3641,
	3861, 4082, 4306, 4532, 4759, 4988, 5220, 5452, 5687, 5923,
	6161, 6400, 6641, 6883, 7127, 7372, 7618, 7866, 8115, 8365,
	8617, 8870, 9124, 9379, 9635, 9893, 10151, 10411, 10672, 10934,
	11197, 11461, 11725, 11991, 12258, 12526, 12795, 13065, 13336, 13608,
	13880, 14154, 14428, 14703, 14979, 15256, 15534, 15813, 16092, 16373,
	16654, 16936, 17218, 17502, 17786, 18071, 18356, 18643, 18930, 19218,
	19507, 19796, 20086, 20377, 20668, 20960, 21253, 21547, 21841, 22136,
	22431, 22727, 23024, 23321, 23619, 23918, 24217, 24517, 24818, 25119,
}

func delete_ship(ship *ship_data_t) {
	/* Set all bytes of record to zero. */
	*ship = ship_data_t{}
	ship.pn = 99
	ship.name = "Unused"
}

func disbanded_ship(ship *ship_data_t) int {
	for nampla_index := 0; nampla_index < species.num_namplas; nampla_index++ {
		nampla := nampla_base[nampla_index]
		if nampla.x != ship.x {
			continue
		}
		if nampla.y != ship.y {
			continue
		}
		if nampla.z != ship.z {
			continue
		}
		if nampla.pn != ship.pn {
			continue
		}
		if nampla.status&DISBANDED_COLONY == 0 {
			continue
		}
		if ship.ship_type != STARBASE && ship.status == IN_ORBIT {
			continue
		}
		/* This ship is either on the surface of a disbanded colony or is a starbase orbiting a disbanded colony. */
		return TRUE
	}
	return FALSE
}

func power(tonnage int) int {
	if tonnage > 4068 {
		fmt.Fprintf(os.Stderr, "\n\n\tLong integer overflow will occur in call to 'power(tonnage)'!\n")
		fmt.Fprintf(os.Stderr, "\t\tActual call is power(%d).\n\n", tonnage)
		os.Exit(255)
	}
	if tonnage <= 100 {
		return ship_power[tonnage]
	}
	/* Tonnage is not in table.
	 * Break it up into two halves and get approximate result = 1.149 * (x1 + x2), using recursion. */
	t1 := tonnage / 2
	t2 := tonnage - t1
	return 1149 * (power(t1) + power(t2)) / 1000
}

func printMishapChanceToOrders(ship *ship_data_t, destx, desty, destz int) {
	if destx == -1 {
		fmt.Fprintf(orders_file, "Mishap chance = ???")
		return
	}

	stx := destx
	sty := desty
	stz := destz
	temp_distance := ((stx - ship.x) * (stx - ship.x)) +
		((sty - ship.y) * (sty - ship.y)) +
		((stz - ship.z) * (stz - ship.z))

	mishap_age := ship.age
	mishap_GV := species.tech_level[GV]
	var mishap_chance int
	if mishap_GV > 0 {
		mishap_chance = 100 * temp_distance / mishap_GV
	} else {
		mishap_chance = 10000
	}
	if mishap_age > 0 && mishap_chance < 10000 {
		success_chance := 10000 - mishap_chance
		success_chance -= (2 * mishap_age * success_chance) / 100
		mishap_chance = 10000 - success_chance
	}
	if mishap_chance > 10000 {
		mishap_chance = 10000
	}
	fmt.Fprintf(orders_file, "mishap chance = %d.%02d%%", mishap_chance/100, mishap_chance%100)
}

func shipDisplayName(ship *ship_data_t) string {
	if ship.class == TR {
		return fmt.Sprintf("%s%d%s %s",
			ship_abbr[ship.class], ship.tonnage, ship_type[ship.ship_type], ship.name)
	}
	return fmt.Sprintf("%s%s %s",
		ship_abbr[ship.class], ship_type[ship.ship_type], ship.name)
}

/* This routine will return a string containing a complete ship name,
 * including its orbital/landed status and age.
 * If global variable "truncate_name" is TRUE,
 * then orbital/landed status and age will not be included. */
func ship_name(ship *ship_data_t) string {
	var ship_is_distorted int
	if ship.item_quantity[FD] == ship.tonnage {
		ship_is_distorted = TRUE
	} else {
		ship_is_distorted = FALSE
	}
	if ship.status == ON_SURFACE {
		ship_is_distorted = FALSE
	}
	if ignore_field_distorters != FALSE {
		ship_is_distorted = FALSE
	}
	if ship_is_distorted != FALSE {
		if ship.class == TR {
			full_ship_id = fmt.Sprintf("%s%d ???", ship_abbr[ship.class], ship.tonnage)
		} else if ship.class == BA {
			full_ship_id = "BAS ???"
		} else {
			full_ship_id = fmt.Sprintf("%s ???", ship_abbr[ship.class])
		}
	} else if ship.class == TR {
		full_ship_id = fmt.Sprintf("%s%d%s %s",
			ship_abbr[ship.class], ship.tonnage, ship_type[ship.ship_type], ship.name)
	} else {
		full_ship_id = fmt.Sprintf("%s%s %s",
			ship_abbr[ship.class], ship_type[ship.ship_type], ship.name)
	}
	if truncate_name != FALSE {
		return full_ship_id
	}
	full_ship_id += " ("
	effective_age := ship.age
	if effective_age < 0 {
		effective_age = 0
	}
	if ship_is_distorted == FALSE {
		if ship.status != UNDER_CONSTRUCTION {
			/* Do age. */
			full_ship_id += fmt.Sprintf("A%d,", effective_age)
		}
	}
	var temp string
	switch ship.status {
	case UNDER_CONSTRUCTION:
		temp = "C"
	case IN_ORBIT:
		temp = fmt.Sprintf("O%d", ship.pn)
	case ON_SURFACE:
		temp = fmt.Sprintf("L%d", ship.pn)
	case IN_DEEP_SPACE:
		temp = "D"
	case FORCED_JUMP:
		temp = "FJ"
	case JUMPED_IN_COMBAT:
		temp = "WD"
	default:
		temp = "***???***"
		fmt.Fprintf(os.Stderr, "\ndebug: ship name '%s' status %12d\n", ship.name, ship.status)
		fmt.Fprintf(os.Stderr, "\n\tWARNING!!!  Internal error in subroutine 'ship_name'\n\n")
	}
	full_ship_id += temp
	if ship.ship_type == STARBASE {
		full_ship_id += fmt.Sprintf(",%d tons", 10000*ship.tonnage)
	}
	full_ship_id += ")"
	return full_ship_id
}
