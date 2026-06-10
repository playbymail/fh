package game

// Types, constants, and file-scope globals from combat.h and the top of
// combat.c, shared by the combat port (combat1.go, combat2.go). The C
// action_data field "char *fighting_unit[MAX_SHIPS]" is a type-punned
// pointer to either a ship_data or a nampla_data record discriminated by
// unit_type; the Go port carries one pointer array per kind.

const (
	MAX_BATTLES        = 50  /* Maximum number of battle locations for all players. */
	MAX_SHIPS          = 200 /* Maximum number of ships at a single battle. */
	MAX_ENGAGE_OPTIONS = 20  /* Maximum number of engagement options that a player may specify for a single battle. */
)

/* Types of combatants. */
const (
	SHIP            = 1
	NAMPLA          = 2
	GENOCIDE_NAMPLA = 3
	BESIEGED_NAMPLA = 4
)

/* Types of special targets. */
const (
	TARGET_WARSHIPS   = 1
	TARGET_TRANSPORTS = 2
	TARGET_STARBASES  = 3
	TARGET_PDS        = 4
)

/* Types of actions. */
const (
	DEFENSE_IN_PLACE   = 0
	DEEP_SPACE_DEFENSE = 1
	PLANET_DEFENSE     = 2
	DEEP_SPACE_FIGHT   = 3
	PLANET_ATTACK      = 4
	PLANET_BOMBARDMENT = 5
	GERM_WARFARE       = 6
	SIEGE              = 7
)

/* Special types. */
const (
	NON_COMBATANT = 1
)

type battle_data_t struct {
	x, y, z                   int
	num_species_here          int
	spec_num                  [MAX_SPECIES]int
	summary_only              [MAX_SPECIES]int
	transport_withdraw_age    [MAX_SPECIES]int
	warship_withdraw_age      [MAX_SPECIES]int
	fleet_withdraw_percentage [MAX_SPECIES]int
	haven_x                   [MAX_SPECIES]int
	haven_y                   [MAX_SPECIES]int
	haven_z                   [MAX_SPECIES]int
	special_target            [MAX_SPECIES]int
	hijacker                  [MAX_SPECIES]int
	can_be_surprised          [MAX_SPECIES]int
	enemy_mine                [MAX_SPECIES][MAX_SPECIES]int
	num_engage_options        [MAX_SPECIES]int
	engage_option             [MAX_SPECIES][MAX_ENGAGE_OPTIONS]int
	engage_planet             [MAX_SPECIES][MAX_ENGAGE_OPTIONS]int
	ambush_amount             [MAX_SPECIES]int
}

type action_data_t struct {
	num_units_fighting     int
	fighting_species_index [MAX_SHIPS]int
	num_shots              [MAX_SHIPS]int
	shots_left             [MAX_SHIPS]int
	weapon_damage          [MAX_SHIPS]int
	shield_strength        [MAX_SHIPS]int
	shield_strength_left   [MAX_SHIPS]int
	original_age_or_PDs    [MAX_SHIPS]int
	bomb_damage            [MAX_SHIPS]int
	surprised              [MAX_SHIPS]int
	unit_type              [MAX_SHIPS]int
	fighting_unit_ship     [MAX_SHIPS]*ship_data_t   /* C: char *fighting_unit[], unit_type == SHIP */
	fighting_unit_nampla   [MAX_SHIPS]*nampla_data_t /* C: char *fighting_unit[], other unit types */
}

// File-scope globals from combat.c.
var ambush_took_place int
var append_log [MAX_SPECIES]int
var attacking_ML int
var battle_base []*battle_data_t
var c_nampla [MAX_SPECIES][]*nampla_data_t
var c_ship [MAX_SPECIES][]*ship_data_t
var c_species [MAX_SPECIES]*species_data_t
var combat_location [1000]int
var combat_option [1000]int
var deep_space_defense int
var defending_ML int
var field_distorted [MAX_SPECIES]int
var first_battle = TRUE
var germ_bombs_used [MAX_SPECIES][MAX_SPECIES]int
var make_enemy [MAX_SPECIES][MAX_SPECIES]int
var num_combat_options int
var strike_phase = FALSE
var x_attacked_y [MAX_SPECIES][MAX_SPECIES]int

func resetCombatState() {
	ambush_took_place = 0
	append_log = [MAX_SPECIES]int{}
	attacking_ML = 0
	battle_base = nil
	c_nampla = [MAX_SPECIES][]*nampla_data_t{}
	c_ship = [MAX_SPECIES][]*ship_data_t{}
	c_species = [MAX_SPECIES]*species_data_t{}
	combat_location = [1000]int{}
	combat_option = [1000]int{}
	deep_space_defense = 0
	defending_ML = 0
	field_distorted = [MAX_SPECIES]int{}
	first_battle = TRUE
	germ_bombs_used = [MAX_SPECIES][MAX_SPECIES]int{}
	make_enemy = [MAX_SPECIES][MAX_SPECIES]int{}
	num_combat_options = 0
	strike_phase = FALSE
	x_attacked_y = [MAX_SPECIES][MAX_SPECIES]int{}
}
