package game

import "strings"

// Port of command.c — the order-file parser.
//
// The C code walks global char buffers with pointers (input_line_pointer
// is a char* into input_line). Here input_line_pointer is an index into
// the input_line string, and reading at/past len(input_line) yields 0
// (the C NUL terminator); see the at helper below.

// at mimics reading a C char buffer: indexes at/past the end of the
// string (or before the start) read as NUL.
func at(s string, i int) byte {
	if i < 0 || i >= len(s) {
		return 0
	}
	return s[i]
}

// C ctype helpers, ASCII only (matching the C engine's locale).
func isdigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isalpha(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isalnum(c byte) bool {
	return isalpha(c) || isdigit(c)
}

func toupper(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - ('a' - 'A')
	}
	return c
}

// upcase returns an upper case (ASCII only) copy of s, mirroring the C
// idiom of copying a name buffer through toupper() one char at a time.
func upcase(s string) string {
	b := []byte(s)
	for i := 0; i < len(b); i++ {
		b[i] = toupper(b[i])
	}
	return string(b)
}

// set_input_line_char overwrites one byte of the global input_line,
// mirroring C code that stores through a pointer into the buffer.
func set_input_line_char(i int, c byte) {
	input_line = input_line[:i] + string(c) + input_line[i+1:]
}

// strchr_from is C strchr(s + from, c) returning an index, or -1 for NULL.
func strchr_from(s string, from int, c byte) int {
	if from < 0 {
		from = 0
	}
	if from > len(s) {
		return -1
	}
	i := strings.IndexByte(s[from:], c)
	if i < 0 {
		return -1
	}
	return from + i
}

/* The following routine will check that the next argument in the current command line is followed by a comma or tab.
 * If not present, it will try to insert a comma in the proper position.
 * This routine should be called only AFTER an error has been detected. */
func fix_separator() {
	var n, first_class, fix_made, num_commas int
	var c byte
	var temp_ptr, temp2_ptr, first_comma int

	skip_whitespace()
	if isdigit(at(input_line, input_line_pointer)) {
		/* Nothing can be done. */
		return
	}
	if strchr_from(input_line, input_line_pointer, ' ') < 0 {
		/* Ditto. */
		return
	}
	fix_made = FALSE

	/* Look for a ship, planet, or species abbreviation after the first one.
	 * If it is preceeded by a space, convert the space to a comma. */
	temp_ptr = input_line_pointer
	first_class = get_class_abbr() /* Skip first one but remember what it was. */
	for {
		skip_whitespace()
		temp2_ptr = input_line_pointer - 1
		if at(input_line, input_line_pointer) == '\n' {
			break
		}
		if at(input_line, input_line_pointer) == ';' {
			break
		}
		if at(input_line, input_line_pointer) == 0 {
			/* End of buffer; the C code relies on a terminating newline. */
			break
		}
		/* The following is to prevent an infinite loop. */
		if !isalnum(at(input_line, input_line_pointer)) {
			input_line_pointer++
			continue
		}

		n = get_class_abbr()
		if n == SHIP_CLASS || n == PLANET_ID || n == SPECIES_ID {
			/* Convert space preceeding abbreviation to a comma. */
			if at(input_line, temp2_ptr) == ' ' {
				set_input_line_char(temp2_ptr, ',')
				fix_made = TRUE
			}
		}
	}
	input_line_pointer = temp_ptr

	if fix_made != FALSE {
		return
	}

	/* Look for a space followed by a digit.
	 * If found, convert the space to a comma.
	 * If exactly two or four commas are added, re-convert the first one back to a space;
	 * e.g. Jump TR1 Seeker,7,99,99,99 or Build TR1 Seeker,7,50. */
	num_commas = 0
	for {
		c = at(input_line, temp_ptr)
		temp_ptr++
		if c == '\n' {
			break
		}
		if c == ';' {
			break
		}
		if c == 0 {
			/* End of buffer; the C code relies on a terminating newline. */
			break
		}
		if c != ' ' {
			continue
		}
		if isdigit(at(input_line, temp_ptr)) {
			temp_ptr-- /* Convert space to a comma. */
			set_input_line_char(temp_ptr, ',')
			if num_commas == 0 {
				first_comma = temp_ptr
			}
			num_commas++
			temp_ptr++
			fix_made = TRUE
		}
	}

	if fix_made != FALSE {
		if num_commas == 2 || num_commas == 4 {
			set_input_line_char(first_comma, ' ')
		}
		return
	}

	/* Now's the time for wild guesses. */
	temp_ptr = input_line_pointer

	/* If first word is a valid abbreviation, put a comma after the second word. */
	if first_class == SHIP_CLASS || first_class == PLANET_ID || first_class == SPECIES_ID {
		temp_ptr = strchr_from(input_line, temp_ptr, ' ') + 1
		temp_ptr = strchr_from(input_line, temp_ptr, ' ')
		if temp_ptr >= 0 {
			set_input_line_char(temp_ptr, ',')
		}
		return
	}

	/* First word is not a valid abbreviation.  Put a comma after it. */
	temp_ptr = strchr_from(input_line, temp_ptr, ' ')
	if temp_ptr >= 0 {
		set_input_line_char(temp_ptr, ',')
	}
}

/* get_class_abbr will return 0 if the item found was not of the appropriate type, and 1 or greater if an item of the correct type was found. */
/* Get a class abbreviation and return TECH_ID, ITEM_CLASS, SHIP_CLASS,
   PLANET_ID, SPECIES_ID or ALLIANCE_ID as appropriate, or UNKNOWN if it
   cannot be identified. Also, set "abbr_type" to this value. If it is
   TECH_ID, ITEM_CLASS or SHIP_CLASS, "abbr_index" will contain the
   abbreviation index. If it is a ship, "tonnage" will contain tonnage/10,000,
   and "sub_light" will be TRUE or FALSE. (Tonnage value returned is based
   ONLY on abbreviation.) */
func get_class_abbr() int {
	var i int
	var digit_start int

	skip_whitespace()

	abbr_type = UNKNOWN

	if !isalnum(at(input_line, input_line_pointer)) {
		return UNKNOWN
	}
	abbr := []byte{toupper(at(input_line, input_line_pointer))}
	input_line_pointer++

	if !isalnum(at(input_line, input_line_pointer)) {
		return UNKNOWN
	}
	abbr = append(abbr, toupper(at(input_line, input_line_pointer)))
	input_line_pointer++

	input_abbr = string(abbr)

	/* Check for IDs that are followed by one or more digits or letters. */
	digit_start = input_line_pointer
	for isalnum(at(input_line, input_line_pointer)) {
		abbr = append(abbr, at(input_line, input_line_pointer))
		input_line_pointer++
		input_abbr = string(abbr)
	}

	/* Check tech ID. */
	for i = 0; i < 6; i++ {
		if input_abbr == tech_abbr[i] {
			abbr_index = i
			abbr_type = TECH_ID
			return abbr_type
		}
	}

	/* Check item abbreviations. */
	for i = 0; i < MAX_ITEMS; i++ {
		if input_abbr == item_abbr[i] {
			abbr_index = i
			abbr_type = ITEM_CLASS
			return abbr_type
		}
	}

	/* Check ship abbreviations. */
	for i = 0; i < NUM_SHIP_CLASSES; i++ {
		if input_abbr[:2] == ship_abbr[i] { /* strncmp(input_abbr, ship_abbr[i], 2) */
			input_line_pointer = digit_start
			abbr_index = i
			tonnage = ship_tonnage[i]
			if i == TR {
				tonnage = 0
				for isdigit(at(input_line, input_line_pointer)) {
					tonnage = (10 * tonnage) + int(at(input_line, input_line_pointer)-'0')
					input_line_pointer++
				}
			}
			if toupper(at(input_line, input_line_pointer)) == 'S' {
				sub_light = TRUE
				input_line_pointer++
			} else {
				sub_light = FALSE
			}
			if isalnum(at(input_line, input_line_pointer)) {
				/* Garbage. */
				break
			}
			abbr_type = SHIP_CLASS
			return abbr_type
		}
	}

	/* Check for planet name. */
	if input_abbr == "PL" {
		abbr_type = PLANET_ID
		return abbr_type
	}

	/* Check for species name. */
	if input_abbr == "SP" {
		abbr_type = SPECIES_ID
		return abbr_type
	}

	abbr_type = UNKNOWN
	return abbr_type
}

/*
Get a class abbreviation and return TECH_ID, ITEM_CLASS, SHIP_CLASS,

	PLANET_ID, SPECIES_ID or ALLIANCE_ID as appropriate, or UNKNOWN if it
	cannot be identified. Also, set "abbr_type" to this value. If it is
	TECH_ID, ITEM_CLASS or SHIP_CLASS, "abbr_index" will contain the
	abbreviation index. If it is a ship, "tonnage" will contain tonnage/10,000,
	and "sub_light" will be TRUE or FALSE. (Tonnage value returned is based
	ONLY on abbreviation.)
*/
func get_class_abbr_from_arg(arg string) int {
	var i int
	var digit_start int

	p := 0 /* index into arg (the C code advances the arg pointer) */

	abbr_type = UNKNOWN

	if !isalnum(at(arg, p)) {
		return UNKNOWN
	}
	abbr := []byte{toupper(at(arg, p))}
	p++

	if !isalnum(at(arg, p)) {
		return UNKNOWN
	}
	abbr = append(abbr, toupper(at(arg, p)))
	p++

	input_abbr = string(abbr)

	/* Check for IDs that are followed by one or more digits or letters. */
	digit_start = p
	for isalnum(at(arg, p)) {
		abbr = append(abbr, at(arg, p))
		p++
		input_abbr = string(abbr)
	}

	/* Check tech ID. */
	for i = 0; i < 6; i++ {
		if input_abbr == tech_abbr[i] {
			abbr_index = i
			abbr_type = TECH_ID
			return abbr_type
		}
	}

	/* Check item abbreviations. */
	for i = 0; i < MAX_ITEMS; i++ {
		if input_abbr == item_abbr[i] {
			abbr_index = i
			abbr_type = ITEM_CLASS
			return abbr_type
		}
	}

	/* Check ship abbreviations. */
	for i = 0; i < NUM_SHIP_CLASSES; i++ {
		if input_abbr[:2] == ship_abbr[i] { /* strncmp(input_abbr, ship_abbr[i], 2) */
			p = digit_start
			abbr_index = i
			tonnage = ship_tonnage[i]
			if i == TR {
				tonnage = 0
				for isdigit(at(arg, p)) {
					tonnage = (10 * tonnage) + int(at(arg, p)-'0')
					p++
				}
			}
			if toupper(at(arg, p)) == 'S' {
				sub_light = TRUE
				p++
			} else {
				sub_light = FALSE
			}
			if isalnum(at(arg, p)) {
				/* Garbage. */
				break
			}
			abbr_type = SHIP_CLASS
			return abbr_type
		}
	}

	/* Check for planet name. */
	if input_abbr == "PL" {
		abbr_type = PLANET_ID
		return abbr_type
	}

	/* Check for species name. */
	if input_abbr == "SP" {
		abbr_type = SPECIES_ID
		return abbr_type
	}

	abbr_type = UNKNOWN
	return abbr_type
}

/* get_command will return 0 if the item found was not of the appropriate type, and 1 or greater if an item of the correct type was found. */
/* Get a command and return its index. */
func get_command() int {
	var i int
	var cmd_n int
	var c byte
	var cmd_s [3]byte

	skip_junk()
	if end_of_file != FALSE {
		return -1
	}

	c = at(input_line, input_line_pointer)
	/* Get first three characters of command word. */
	for i = 0; i < 3; i++ {
		if !isalpha(c) {
			return 0
		}
		cmd_s[i] = toupper(c)
		input_line_pointer++
		c = at(input_line, input_line_pointer)
	}

	/* Skip everything after third character of command word. */
skip_loop:
	for {
		switch c {
		case '\t', '\n', ' ', ',', ';':
			break skip_loop
		case 0:
			/* End of buffer; the C code relies on a terminating newline. */
			break skip_loop
		default:
			input_line_pointer++
			c = at(input_line, input_line_pointer)
		}
	}

	/* find_cmd: */

	/* Find corresponding string in list. */
	cmd_n = UNKNOWN
	for i = 1; i < NUM_COMMANDS; i++ {
		if string(cmd_s[:]) == command_abbr[i] {
			cmd_n = i
			break
		}
	}

	return cmd_n
}

func get_jump_portal() int {
	var found int
	var start_x, start_y, start_z int

	/* See if specified starbase is owned by the current species. */
	original_line_pointer := input_line_pointer
	temp_ship := ship
	found = get_ship()
	portal := ship
	ship = temp_ship
	using_alien_portal = FALSE

	if found != FALSE {
		if portal.ship_type != STARBASE {
			return FALSE
		}
		jump_portal_age = portal.age
		jump_portal_gv = species.tech_level[GV]
		jump_portal_units = portal.item_quantity[JP]
		jump_portal_name = ship_name(portal)
		return TRUE
	}

	start_x = ship.x
	start_y = ship.y
	start_z = ship.z

	if abbr_type != SHIP_CLASS {
		goto check_for_bad_spelling
	}
	if abbr_index != BA {
		goto check_for_bad_spelling
	}

	/* It IS the name of a starbase.  See if another species has given permission to use their starbase. */
	for other_species_number = 1; other_species_number <= galaxy.num_species; other_species_number++ {
		if data_in_memory[other_species_number-1] == FALSE {
			continue
		}
		if other_species_number == species_number {
			continue
		}

		other_species = &spec_data[other_species_number-1]

		found = FALSE

		/* Check if other species has declared this species as an ally. */
		array_index := (species_number - 1) / 32
		bit_number := (species_number - 1) % 32
		bit_mask := uint32(1) << uint(bit_number)
		if (other_species.ally[array_index] & bit_mask) == 0 {
			continue
		}

		/* See if other species has a starbase with the specified name at the start location. */
		for j := 0; j < other_species.num_ships; j++ {
			alien_portal = ship_data[other_species_number-1][j]
			if alien_portal.ship_type != STARBASE {
				continue
			}
			if alien_portal.x != start_x {
				continue
			}
			if alien_portal.y != start_y {
				continue
			}
			if alien_portal.z != start_z {
				continue
			}
			if alien_portal.pn == 99 {
				continue
			}
			/* Make upper case copy of ship name. */
			upper_ship_name := upcase(alien_portal.name)
			/* Compare names. */
			if upper_ship_name == upper_name {
				found = TRUE
				break
			}
		}

		if found != FALSE {
			jump_portal_units = alien_portal.item_quantity[JP]
			jump_portal_age = alien_portal.age
			jump_portal_gv = other_species.tech_level[GV]
			jump_portal_name = ship_name(alien_portal)
			jump_portal_name += " owned by SP "
			jump_portal_name += other_species.name
			using_alien_portal = TRUE
			break
		}
	}

	if found != FALSE {
		return TRUE
	}

check_for_bad_spelling:

	/* Try again, but allow spelling errors. */
	{
		original_ship := ship
		original_ship_base := ship_base
		original_species := species

		for other_species_number = 1; other_species_number <= galaxy.num_species; other_species_number++ {
			if data_in_memory[other_species_number-1] == FALSE {
				continue
			}
			if other_species_number == species_number {
				continue
			}
			species = &spec_data[other_species_number-1]

			/* Check if other species has declared this species as an ally. */
			array_index := (species_number - 1) / 32
			bit_number := (species_number - 1) % 32
			bit_mask := uint32(1) << uint(bit_number)
			if (species.ally[array_index] & bit_mask) == 0 {
				continue
			}
			input_line_pointer = original_line_pointer
			ship_base = ship_data[other_species_number-1]
			found = get_ship()
			if found != FALSE {
				found = FALSE
				if ship.ship_type != STARBASE {
					continue
				}
				if ship.x != start_x {
					continue
				}
				if ship.y != start_y {
					continue
				}
				if ship.z != start_z {
					continue
				}
				if ship.pn == 99 {
					continue
				}
				found = TRUE
				break
			}
		}

		if found != FALSE {
			jump_portal_units = ship.item_quantity[JP]
			jump_portal_age = ship.age
			jump_portal_gv = species.tech_level[GV]
			jump_portal_name = ship_name(ship)
			jump_portal_name += " owned by SP "
			jump_portal_name += species.name
			using_alien_portal = TRUE
		}

		species = original_species
		ship = original_ship
		ship_base = original_ship_base
	}

	return found
}

/* This routine will assign values to global variables x, y, z, pn, star and nampla.
 * If the location is not a named planet, then nampla will be set to NULL.
 * If planet is not specified, pn will be set to zero.
 * If location is valid, TRUE will be returned, otherwise FALSE will be returned. */
func get_location() int {
	var i, n, temp_nampla_index, first_try, name_length int
	var best_score, next_best_score, best_nampla_index int
	var minimum_score int
	var upper_nampla_name string
	var temp1_ptr, temp2_ptr int
	var temp_nampla *nampla_data_t

	/* Check first if x, y, z are specified. */
	nampla = nil
	skip_whitespace()

	if get_value() == 0 {
		goto get_planet
	}
	x = value

	if get_value() == 0 {
		return FALSE
	}
	y = value

	if get_value() == 0 {
		return FALSE
	}
	z = value

	if get_value() == 0 {
		pn = 0
	} else {
		pn = value
	}

	if pn == 0 {
		return TRUE
	}

	/* Get star. Check if planet exists. */
	/* found = FALSE; -- the C code initializes this here, but it is never read. */
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
		if pn > star.num_planets {
			return FALSE
		} else {
			return TRUE
		}
	}

	return FALSE

get_planet:

	/* Save pointers in case of error. */
	temp1_ptr = input_line_pointer

	get_class_abbr()

	temp2_ptr = input_line_pointer

	first_try = TRUE

again:

	input_line_pointer = temp2_ptr

	if abbr_type != PLANET_ID && first_try == FALSE {
		/* Assume abbreviation was accidentally omitted. */
		input_line_pointer = temp1_ptr
	}

	/* Get planet name. */
	get_name()

	/* Search all temp_namplas for name. */
	for temp_nampla_index = 0; temp_nampla_index < species.num_namplas; temp_nampla_index++ {
		temp_nampla = nampla_base[temp_nampla_index]
		if temp_nampla.pn == 99 {
			continue
		}
		/* Make upper case copy of temp_nampla name. */
		upper_nampla_name = upcase(temp_nampla.name)
		/* Compare names. */
		if upper_nampla_name == upper_name {
			goto done
		}
	}

	if first_try != FALSE {
		first_try = FALSE
		goto again
	}

	/* Possibly a spelling error.  Find the best match that is approximately the same. */
	first_try = TRUE

yet_again:

	input_line_pointer = temp2_ptr

	if abbr_type != PLANET_ID && first_try == FALSE {
		/* Assume abbreviation was accidentally omitted. */
		input_line_pointer = temp1_ptr
	}

	/* Get planet name. */
	get_name()

	best_nampla_index = 0
	best_score = -9999
	next_best_score = -9999
	for temp_nampla_index = 0; temp_nampla_index < species.num_namplas; temp_nampla_index++ {
		temp_nampla = nampla_base[temp_nampla_index]
		if temp_nampla.pn == 99 {
			continue
		}
		/* Make upper case copy of temp_nampla name. */
		upper_nampla_name = upcase(temp_nampla.name)
		/* Compare names. */
		n = agrep_score(upper_nampla_name, upper_name)
		if n > best_score {
			best_score = n /* Best match so far. */
			best_nampla_index = temp_nampla_index
		} else if n > next_best_score {
			next_best_score = n
		}
	}

	temp_nampla = nampla_base[best_nampla_index]
	name_length = len(temp_nampla.name)
	minimum_score = name_length - ((name_length / 7) + 1)

	if best_score < minimum_score /* Score too low. */ ||
		name_length < 5 /* No errors allowed. */ ||
		best_score == next_best_score /* Another name with equal score. */ {
		if first_try != FALSE {
			first_try = FALSE
			goto yet_again
		} else {
			return FALSE
		}
	}

done:

	abbr_type = PLANET_ID

	x = temp_nampla.x
	y = temp_nampla.y
	z = temp_nampla.z
	pn = temp_nampla.pn
	nampla = temp_nampla

	return TRUE
}

/* get_name will return 0 if the item found was not of the appropriate type, and 1 or greater if an item of the correct type was found. */
/* Get a name and copy original version to "original_name" and upper case version to "upper_name". Return length of name. */
func get_name() int {
	name_length := 0
	var name_buf []byte

	skip_whitespace()
	for {
		c := at(input_line, input_line_pointer)
		if c == ';' {
			break
		}
		if c == 0 {
			/* End of buffer; the C code relies on a terminating newline. */
			break
		}
		input_line_pointer++
		if c == ',' || c == '\t' || c == '\n' {
			break
		}
		if name_length < 31 {
			name_buf = append(name_buf, c)
			name_length++
		}
	}

	/* Remove any final spaces in name. */
	for name_length > 0 {
		c := name_buf[name_length-1]
		if c != ' ' {
			break
		}
		name_length--
	}
	/* Terminate strings. */
	original_name = string(name_buf[:name_length])
	upper_name = upcase(original_name)
	return name_length
}

/* The following routine will return TRUE and set global variables "ship" and "ship_index" if a valid ship designation is found.
 * Otherwise, it will return FALSE.
 * The algorithm employed allows minor spelling errors, as well as accidental deletion of a ship abbreviation. */
func get_ship() int {
	var n, name_length, best_score, next_best_score, best_ship_index, first_try, minimum_score int
	var upper_ship_name string
	var temp1_ptr, temp2_ptr int
	var best_ship *ship_data_t

	/* Save in case of an error. */
	temp1_ptr = input_line_pointer

	/* Get ship abbreviation. */
	if get_class_abbr() == PLANET_ID {
		input_line_pointer = temp1_ptr
		return FALSE
	}

	temp2_ptr = input_line_pointer

	first_try = TRUE

again:

	input_line_pointer = temp2_ptr

	if abbr_type != SHIP_CLASS && first_try == FALSE {
		/* Assume abbreviation was accidentally omitted. */
		input_line_pointer = temp1_ptr
	}

	/* Get ship name. */
	name_length = get_name()

	/* Search all ships for name. */
	for ship_index = 0; ship_index < species.num_ships; ship_index++ {
		ship = ship_base[ship_index]
		if ship.pn == 99 {
			continue
		}
		/* Make upper case copy of ship name. */
		upper_ship_name = upcase(ship.name)
		/* Compare names. */
		if upper_ship_name == upper_name {
			abbr_type = SHIP_CLASS
			abbr_index = ship.class
			correct_spelling_required = FALSE
			return TRUE
		}
	}

	if first_try != FALSE {
		first_try = FALSE
		goto again
	}

	if correct_spelling_required != FALSE {
		correct_spelling_required = FALSE
		return FALSE
	}

	/* Possibly a spelling error.  Find the best match that is approximately the same. */
	first_try = TRUE

yet_again:

	input_line_pointer = temp2_ptr

	if abbr_type != SHIP_CLASS && first_try == FALSE {
		/* Assume abbreviation was accidentally omitted. */
		input_line_pointer = temp1_ptr
	}

	/* Get ship name. */
	name_length = get_name()

	best_score = -9999
	next_best_score = -9999
	for ship_index = 0; ship_index < species.num_ships; ship_index++ {
		ship = ship_base[ship_index]

		if ship.pn == 99 {
			continue
		}

		/* Make upper case copy of ship name. */
		upper_ship_name = upcase(ship.name)

		n = agrep_score(upper_ship_name, upper_name)
		if n > best_score {
			/* Best match so far. */
			best_score = n
			best_ship = ship
			best_ship_index = ship_index
		} else if n > next_best_score {
			next_best_score = n
		}
	}

	if best_ship == nil {
		return FALSE
	}
	name_length = len(best_ship.name)
	minimum_score = name_length - ((name_length / 7) + 1)

	if best_score < minimum_score /* Score too low. */ ||
		name_length < 5 /* No errors allowed. */ ||
		best_score == next_best_score /* Another name with equal score. */ {
		if first_try != FALSE {
			first_try = FALSE
			goto yet_again
		} else {
			correct_spelling_required = FALSE
			return FALSE
		}
	}

	ship = best_ship
	ship_index = best_ship_index
	abbr_type = SHIP_CLASS
	abbr_index = ship.class
	correct_spelling_required = FALSE
	return TRUE
}

/* This routine will get a species name and return TRUE if found and if it is valid.
 * It will also set global values "g_spec_number" and "g_spec_name".
 * The algorithm employed allows minor spelling errors, as well as accidental deletion of the SP abbreviation. */
func get_species_name() int {
	var n, species_index, best_score, best_species_index, next_best_score, first_try, minimum_score, name_length int
	var sp_name string
	var temp1_ptr, temp2_ptr int
	var sp *species_data_t

	g_spec_number = 0
	/* Save pointers in case of error. */
	temp1_ptr = input_line_pointer
	get_class_abbr()
	temp2_ptr = input_line_pointer

	first_try = TRUE

again:

	input_line_pointer = temp2_ptr

	if abbr_type != SPECIES_ID && first_try == FALSE {
		/* Assume abbreviation was accidentally omitted. */
		input_line_pointer = temp1_ptr
	}

	/* Get species name. */
	get_name()

	for species_index = 0; species_index < galaxy.num_species; species_index++ {
		if data_in_memory[species_index] == FALSE {
			continue
		}
		sp = &spec_data[species_index]

		/* Copy name to g_spec_name and convert it to upper case. */
		g_spec_name = sp.name
		sp_name = upcase(g_spec_name)
		if sp_name == upper_name {
			g_spec_number = species_index + 1
			abbr_type = SPECIES_ID
			return TRUE
		}
	}

	if first_try != FALSE {
		first_try = FALSE
		goto again
	}

	/* Possibly a spelling error.  Find the best match that is approximately the same. */
	first_try = TRUE

yet_again:

	input_line_pointer = temp2_ptr

	if abbr_type != SPECIES_ID && first_try == FALSE {
		/* Assume abbreviation was accidentally omitted. */
		input_line_pointer = temp1_ptr
	}

	/* Get species name. */
	get_name()

	best_score = -9999
	next_best_score = -9999
	best_species_index = -1
	for species_index = 0; species_index < galaxy.num_species; species_index++ {
		if data_in_memory[species_index] == FALSE {
			continue
		}
		sp = &spec_data[species_index]
		/* Convert name to upper case. */
		sp_name = upcase(sp.name)
		n = agrep_score(sp_name, upper_name)
		if n > best_score {
			/* Best match so far. */
			best_score = n
			best_species_index = species_index
		} else if n > next_best_score {
			next_best_score = n
		}
	}

	/* No in-memory species was scored, so there is nothing to match against. */
	if best_species_index < 0 {
		return FALSE
	}

	sp = &spec_data[best_species_index]
	name_length = len(sp.name)
	minimum_score = name_length - ((name_length / 7) + 1)

	if best_score < minimum_score /* Score too low. */ ||
		name_length < 5 /* No errors allowed. */ ||
		best_score == next_best_score /* Another name with equal score. */ {
		if first_try != FALSE {
			first_try = FALSE
			goto yet_again
		} else {
			return FALSE
		}
	}

	/* Copy name to g_spec_name. */
	g_spec_name = sp.name
	g_spec_number = best_species_index + 1
	abbr_type = SPECIES_ID
	return TRUE
}

func get_transfer_point() int {
	/* Find out if it is a ship or a planet. First try for a correctly spelled ship name. */
	temp_ptr := input_line_pointer
	correct_spelling_required = TRUE
	if get_ship() != FALSE {
		return TRUE
	}
	/* Probably not a ship. See if it's a planet. */
	input_line_pointer = temp_ptr
	if get_location() != FALSE {
		if nampla != nil {
			return TRUE
		}
		return FALSE
	}
	/* Now check for an incorrectly spelled ship name. */
	input_line_pointer = temp_ptr
	if get_ship() != FALSE {
		return TRUE
	}
	return FALSE
}

/* Read a long decimal and place its value in 'value'. */
func get_value() int {
	skip_whitespace()
	/* Emulate n = sscanf(input_line_pointer, "%ld", &value). */
	i := input_line_pointer
	c := at(input_line, i)
	negative := false
	if c == '+' || c == '-' {
		negative = c == '-'
		i++
		c = at(input_line, i)
	}
	if !isdigit(c) {
		/* Not a numeric value. */
		return 0
	}
	v := 0
	for isdigit(at(input_line, i)) {
		v = (10 * v) + int(at(input_line, i)-'0')
		i++
	}
	if negative {
		v = -v
	}
	value = v
	/* Skip numeric string. */
	input_line_pointer++ /* Skip first sign or digit. */
	for isdigit(at(input_line, input_line_pointer)) {
		input_line_pointer++
	}
	return 1
}

/* Skip white space and comments. */
func skip_junk() {
	var ok bool

again:

	/* Read next line. */
	input_line, ok = readln(input_file, 256)
	if !ok {
		end_of_file = TRUE
		return
	}
	input_line_pointer = 0

	if just_opened_file != FALSE {
		/* Skip mail header, if any. */
		if at(input_line, 0) == '\n' {
			goto again
		}
		just_opened_file = FALSE
		if strings.HasPrefix(input_line, "From ") { /* strncmp(input_line, "From ", 5) == 0 */
			/* This is a mail header. */
			for {
				input_line, ok = readln(input_file, 256)
				if !ok {
					end_of_file = TRUE /* Weird. */
					return
				}
				input_line_pointer = 0
				if at(input_line, 0) == '\n' {
					/* End of header. */
					break
				}
			}
			goto again
		}
	}

	original_line = input_line /* Make a copy. */

	/* Skip white space and comments. */
	for {
		switch at(input_line, input_line_pointer) {
		case ';', '\n': /* Semi-colon, newline. */
			goto again
		case '\t', ' ', ',': /* Tab, space, comma. */
			input_line_pointer++
			continue
		default:
			return
		}
	}
}

func skip_whitespace() {
	for {
		switch at(input_line, input_line_pointer) {
		case '\t', ' ', ',': /* Tab, space, comma. */
			input_line_pointer++
		default:
			return
		}
	}
}
