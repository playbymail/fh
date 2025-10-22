package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Experimental struct {
	EconUnits   int  `json:"x-econ-units"`
	MakeBridges bool `json:"x-bridges"`
	MABase      int  `json:"x-ma-base"`
	MIBase      int  `json:"x-mi-base"`
	ShipYards   int  `json:"x-ship-yards"`
	TechBI      int  `json:"x-tech-bi"`
	TechGV      int  `json:"x-tech-gv"`
	TechLS      int  `json:"x-tech-ls"`
	TechMA      int  `json:"x-tech-ma"`
	TechMI      int  `json:"x-tech-mi"`
	TechML      int  `json:"x-tech-ml"`
}

type SpeciesConfig struct {
	Email        string        `json:"email"`
	GovtName     string        `json:"govt-name"`
	GovtType     string        `json:"govt-type"`
	Homeworld    string        `json:"homeworld"`
	Name         string        `json:"name"`
	ML           int           `json:"tech-ml"`
	GV           int           `json:"tech-gv"`
	LS           int           `json:"tech-ls"`
	BI           int           `json:"tech-bi"`
	Experimental *Experimental `json:"experimental,omitempty"`
}

func (s *SpeciesConfig) Validate() error {
	if s.ML < 1 || s.ML > 15 {
		return fmt.Errorf("tech-ml must be between 1 and 15, got %d", s.ML)
	}
	if s.GV < 1 || s.GV > 15 {
		return fmt.Errorf("tech-gv must be between 1 and 15, got %d", s.GV)
	}
	if s.LS < 1 || s.LS > 15 {
		return fmt.Errorf("tech-ls must be between 1 and 15, got %d", s.LS)
	}
	if s.BI < 1 || s.BI > 15 {
		return fmt.Errorf("tech-bi must be between 1 and 15, got %d", s.BI)
	}

	if s.Experimental != nil {
		if s.Experimental.EconUnits > 99999999 {
			return fmt.Errorf("x-econ-units must be between 0 and 99999999, got %d", s.Experimental.EconUnits)
		}
		if s.Experimental.MABase > 99999999 {
			return fmt.Errorf("x-ma-base must be between 0 and 99999999, got %d", s.Experimental.MABase)
		}
		if s.Experimental.MIBase > 99999999 {
			return fmt.Errorf("x-mi-base must be between 0 and 99999999, got %d", s.Experimental.MIBase)
		}
		if s.Experimental.ShipYards > 99 {
			return fmt.Errorf("x-ship-yards must be between 0 and 99, got %d", s.Experimental.ShipYards)
		}
		if s.Experimental.TechBI > 999 {
			return fmt.Errorf("x-tech-bi must be between 0 and 999, got %d", s.Experimental.TechBI)
		}
		if s.Experimental.TechGV > 999 {
			return fmt.Errorf("x-tech-gv must be between 0 and 999, got %d", s.Experimental.TechGV)
		}
		if s.Experimental.TechLS > 999 {
			return fmt.Errorf("x-tech-ls must be between 0 and 999, got %d", s.Experimental.TechLS)
		}
		if s.Experimental.TechMA > 999 {
			return fmt.Errorf("x-tech-ma must be between 0 and 999, got %d", s.Experimental.TechMA)
		}
		if s.Experimental.TechMI > 999 {
			return fmt.Errorf("x-tech-mi must be between 0 and 999, got %d", s.Experimental.TechMI)
		}
		if s.Experimental.TechML > 999 {
			return fmt.Errorf("x-tech-ml must be between 0 and 999, got %d", s.Experimental.TechML)
		}
	}

	return nil
}

func LoadSpecies(filename string) ([]*SpeciesConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to read %s: %w", filename, err)
	}

	var species []*SpeciesConfig
	if err := json.Unmarshal(data, &species); err != nil {
		return nil, fmt.Errorf("%s does not contain valid JSON: %w", filename, err)
	}

	if len(species) == 0 {
		return nil, fmt.Errorf("%s contains no data", filename)
	}

	const maxSpecies = 100
	if len(species) > maxSpecies {
		return nil, fmt.Errorf("%s contains too many species (expect 0..%d, got %d)", filename, maxSpecies, len(species))
	}

	for i, s := range species {
		if err := s.Validate(); err != nil {
			return nil, fmt.Errorf("%s: species %d: %w", filename, i, err)
		}
	}

	return species, nil
}
