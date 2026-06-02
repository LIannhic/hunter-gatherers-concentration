package board

// BiomeType définit l'identité visuelle et les règles de génération d'une zone.
type BiomeType string

const (
	BiomeDefault BiomeType = "default" // Zones de départ, fin, calmes
	BiomeForest  BiomeType = "forest"  // Uniforme et aléatoire
	BiomeCave    BiomeType = "cave"    // Attraction cardinale vers la cible
	BiomeDesert  BiomeType = "desert"  // Répulsion cardinale depuis la cible
	BiomeSwamp   BiomeType = "swamp"   // Vortex en spirale concentrique
)

// Climate représente les conditions météorologiques dominantes.
type Climate string

const (
	ClimateTemperate Climate = "temperate"
	ClimateHumid     Climate = "humid"
	ClimateArid      Climate = "arid"
)

// Season représente le cycle temporel impactant la maturation des ressources.
type Season int

const (
	SeasonAwakening Season = iota
	SeasonZenith
	SeasonDecay
	SeasonSlumber
)

// SuccessionStage représente l'évolution écologique d'une parcelle.
type SuccessionStage int

const (
	StagePreliminary SuccessionStage = iota
	StagePioneer
	StageClimax
)
