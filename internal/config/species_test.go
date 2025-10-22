package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSpecies(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantErr   bool
		wantCount int
	}{
		{
			name: "valid single species",
			json: `[{
				"email": "test@example.com",
				"name": "Humans",
				"homeworld": "Earth",
				"govt-name": "United Earth",
				"govt-type": "Democracy",
				"tech-ml": 10,
				"tech-gv": 12,
				"tech-ls": 8,
				"tech-bi": 15,
				"experimental": {
					"x-econ-units": 1000,
					"x-bridges": true,
					"x-ma-base": 5000,
					"x-mi-base": 3000,
					"x-ship-yards": 10,
					"x-tech-bi": 150,
					"x-tech-gv": 120,
					"x-tech-ls": 80,
					"x-tech-ma": 100,
					"x-tech-mi": 90,
					"x-tech-ml": 110
				}
			}]`,
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "multiple species",
			json: `[
				{
					"name": "Species A",
					"tech-ml": 5,
					"tech-gv": 5,
					"tech-ls": 5,
					"tech-bi": 5
				},
				{
					"name": "Species B",
					"tech-ml": 10,
					"tech-gv": 10,
					"tech-ls": 10,
					"tech-bi": 10
				}
			]`,
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:    "empty array",
			json:    `[]`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			json:    `{not valid json}`,
			wantErr: true,
		},
		{
			name: "tech value too low",
			json: `[{
				"name": "Test",
				"tech-ml": 0,
				"tech-gv": 5,
				"tech-ls": 5,
				"tech-bi": 5
			}]`,
			wantErr: true,
		},
		{
			name: "tech value too high",
			json: `[{
				"name": "Test",
				"tech-ml": 5,
				"tech-gv": 16,
				"tech-ls": 5,
				"tech-bi": 5
			}]`,
			wantErr: true,
		},
		{
			name: "experimental value out of range",
			json: `[{
				"name": "Test",
				"tech-ml": 5,
				"tech-gv": 5,
				"tech-ls": 5,
				"tech-bi": 5,
				"experimental": {
					"x-tech-bi": 1000
				}
			}]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filename := filepath.Join(tmpDir, "species.json")

			if err := os.WriteFile(filename, []byte(tt.json), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			species, err := LoadSpecies(filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadSpecies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(species) != tt.wantCount {
				t.Errorf("LoadSpecies() returned %d species, want %d", len(species), tt.wantCount)
			}
		})
	}
}

func TestLoadSpecies_FileNotFound(t *testing.T) {
	_, err := LoadSpecies("/nonexistent/file.json")
	if err == nil {
		t.Error("LoadSpecies() should return error for nonexistent file")
	}
}

func TestSpeciesConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  SpeciesConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: SpeciesConfig{
				ML: 5,
				GV: 10,
				LS: 15,
				BI: 8,
			},
			wantErr: false,
		},
		{
			name: "ML too low",
			config: SpeciesConfig{
				ML: 0,
				GV: 5,
				LS: 5,
				BI: 5,
			},
			wantErr: true,
		},
		{
			name: "GV too high",
			config: SpeciesConfig{
				ML: 5,
				GV: 16,
				LS: 5,
				BI: 5,
			},
			wantErr: true,
		},
		{
			name: "experimental EconUnits too high",
			config: SpeciesConfig{
				ML: 5,
				GV: 5,
				LS: 5,
				BI: 5,
				Experimental: &Experimental{
					EconUnits: 100000000,
				},
			},
			wantErr: true,
		},
		{
			name: "experimental ShipYards valid max",
			config: SpeciesConfig{
				ML: 5,
				GV: 5,
				LS: 5,
				BI: 5,
				Experimental: &Experimental{
					ShipYards: 99,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SpeciesConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
