// Package loader charge les configurations depuis JSON/ fichiers
package loader

import (
	"encoding/json"
	"os"
)

// ResourceConfig définition d'une ressource depuis JSON
type ResourceConfig struct {
	Type         string   `json:"type"`
	DisplayName  string   `json:"display_name"`
	Stages       []string `json:"stages"`
	BaseValue    int      `json:"base_value"`
	CanPropagate bool     `json:"can_propagate"`
	Element      string   `json:"element"`
	MatchTypes   []string `json:"match_types"`
}

// CreatureConfig définition d'une créature depuis JSON
type CreatureConfig struct {
	Type        string   `json:"type"`
	DisplayName string   `json:"display_name"`
	Behavior    string   `json:"behavior"`
	Aggression  int      `json:"aggression"`
	Speed       int      `json:"speed"`
	Tags        []string `json:"tags"`
}

// GameConfig configuration globale du jeu
type GameConfig struct {
	GridWidth  int              `json:"grid_width"`
	GridHeight int              `json:"grid_height"`
	Resources  []ResourceConfig `json:"resources"`
	Creatures  []CreatureConfig `json:"creatures"`
}

// LoadGameConfig charge la configuration depuis un fichier JSON
func LoadGameConfig(path string) (*GameConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// Retourne une config par défaut si le fichier n'existe pas
		return DefaultConfig(), nil
	}
	
	var config GameConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

// DefaultConfig retourne une configuration par défaut
func DefaultConfig() *GameConfig {
	return &GameConfig{
		GridWidth:  6,
		GridHeight: 6,
		Resources: []ResourceConfig{
			{
				Type:         "dreamberry",
				DisplayName:  "Baie de Rêve",
				Stages:       []string{"bourgeon", "fleur", "fruit", "gâté"},
				BaseValue:    100,
				CanPropagate: true,
				Element:      "ethereal",
				MatchTypes:   []string{"identical"},
			},
			{
				Type:         "moonstone",
				DisplayName:  "Pierre de Lune",
				Stages:       []string{"brute", "taillée", "polie"},
				BaseValue:    200,
				CanPropagate: false,
				Element:      "earth",
				MatchTypes:   []string{"identical"},
			},
		},
		Creatures: []CreatureConfig{
			{
				Type:        "lumifly",
				DisplayName: "Lumifly",
				Behavior:    "pollinating",
				Aggression:  0,
				Speed:       1,
				Tags:        []string{"flying", "passive"},
			},
			{
				Type:        "shadowstalker",
				DisplayName: "Traqueur d'Ombre",
				Behavior:    "hunting",
				Aggression:  80,
				Speed:       2,
				Tags:        []string{"dangerous", "aggressive"},
			},
		},
	}
}
