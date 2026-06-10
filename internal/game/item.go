package game

// Port of item.c (the tables live in tables.go).

func check_high_tech_items(tech, old_tech_level, new_tech_level int) {
	for i := 0; i < MAX_ITEMS; i++ {
		if item_critical_tech[i] != tech || new_tech_level < item_tech_requirment[i] ||
			old_tech_level >= item_tech_requirment[i] {
			continue
		}
		log_string("  You now have the technology to build ")
		log_string(item_name[i])
		log_string("s.\n")
	}

	/* Check for high tech abilities that are not associated with specific items. */
	if tech == MA && old_tech_level < 25 && new_tech_level >= 25 {
		log_string("  You now have the technology to do interspecies construction.\n")
	}
}
