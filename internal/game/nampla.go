package game

import "fmt"

// Port of nampla.c.

/* This routine will set or clear the POPULATED bit for a nampla.
 * It will return TRUE if the nampla is populated or FALSE if not.
 * It will also check if a message associated with this planet should be logged. */
func check_population(nampla *nampla_data_t) int {
	var is_now_populated int
	var was_already_populated int

	if nampla.status&POPULATED != 0 {
		was_already_populated = TRUE
	} else {
		was_already_populated = FALSE
	}
	total_pop := nampla.mi_base + nampla.ma_base + nampla.IUs_to_install + nampla.AUs_to_install +
		nampla.item_quantity[PD] + nampla.item_quantity[CU] + nampla.pop_units
	if total_pop > 0 {
		nampla.status |= POPULATED
		is_now_populated = TRUE
	} else {
		nampla.status &= ^(POPULATED | MINING_COLONY | RESORT_COLONY)
		is_now_populated = FALSE
	}
	if is_now_populated != FALSE && was_already_populated == FALSE {
		if nampla.message != 0 {
			/* There is a message that must be logged whenever this planet becomes populated for the first time. */
			filename := fmt.Sprintf("message%d.txt", nampla.message)
			log_message(filename)
		}
	}
	return is_now_populated
}

/* delete_nampla delete a nampla record. not really. */
func delete_nampla(nampla *nampla_data_t) {
	/* Set all bytes of record to zero. */
	*nampla = nampla_data_t{}
	nampla.name = "Unused"
	nampla.pn = 99
}
