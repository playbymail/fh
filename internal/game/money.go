package game

import "fmt"

// Port of money.c.

var balance int
var EU_spending_limit int
var production_capacity int
var raw_material_units int

func resetMoneyState() {
	balance = 0
	EU_spending_limit = 0
	production_capacity = 0
	raw_material_units = 0
}

func check_bounced(amount_needed int) int {
	/* Check if we have sufficient funds for this purchase. */
	if amount_needed > balance {
		take_from_EUs := amount_needed - balance

		if take_from_EUs <= EU_spending_limit && take_from_EUs <= species.econ_units {
			species.econ_units -= take_from_EUs
			EU_spending_limit -= take_from_EUs
			balance = amount_needed
		} else {
			return TRUE
		}
	}

	/* Reduce various balances appropriately. */
	if raw_material_units >= amount_needed {
		if production_capacity >= amount_needed {
			/* Enough of both. */
			raw_material_units -= amount_needed
			production_capacity -= amount_needed
		} else {
			/* Enough RMs but not enough PC. */
			raw_material_units -= production_capacity
			production_capacity = 0
		}
	} else {
		if production_capacity >= amount_needed {
			/* Enough PC but not enough RMs. */
			production_capacity -= raw_material_units
			raw_material_units = 0
		} else {
			/* Not enough RMs or PC. */
			limiting_balance := production_capacity
			if raw_material_units <= production_capacity {
				limiting_balance = raw_material_units
			}
			raw_material_units -= limiting_balance
			production_capacity -= limiting_balance
		}
	}

	balance -= amount_needed

	return FALSE
}

func transfer_balance() {
	/* Log end of production.
	 * Do not print ending balance for mining or resort colonies. */
	limiting_amount := 0
	fmt.Fprintf(log_file, "  End of production on PL %s.", nampla.name)
	if nampla.status&(MINING_COLONY|RESORT_COLONY) == 0 {
		if raw_material_units > production_capacity {
			limiting_amount = production_capacity
		} else {
			limiting_amount = raw_material_units
		}
		fmt.Fprintf(log_file, " (Ending balance is %d.)", limiting_amount)
	}
	fmt.Fprintf(log_file, "\n")

	/* Convert unused balance to economic units. */
	species.econ_units += limiting_amount
	raw_material_units -= limiting_amount

	/* Carry over unused raw material units into next turn. */
	nampla.item_quantity[RM] += raw_material_units

	balance = 0
}
