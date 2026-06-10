package game

// Port of dev_log.c.

func start_dev_log(num_CUs, num_IUs, num_AUs int) {
	log_string("    ")
	log_int(num_CUs)
	log_string(" Colonist Unit")
	if num_CUs != 1 {
		log_char('s')
	}

	if num_IUs+num_AUs != 0 {
		if num_IUs > 0 {
			if num_AUs == 0 {
				log_string(" and ")
			} else {
				log_string(", ")
			}

			log_int(num_IUs)
			log_string(" Colonial Mining Unit")
			if num_IUs != 1 {
				log_char('s')
			}
		}

		if num_AUs > 0 {
			if num_IUs > 0 {
				log_char(',')
			}

			log_string(" and ")

			log_int(num_AUs)
			log_string(" Colonial Manufacturing Unit")
			if num_AUs != 1 {
				log_char('s')
			}
		}
	}

	log_string(" were built")
}
