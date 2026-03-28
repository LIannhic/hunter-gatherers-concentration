package meta

type DifficultyLevel string

const (
	LevelEasy   DifficultyLevel = "Easy"   // Révèle tout au début
	LevelNormal DifficultyLevel = "Normal" // Révèle une tuile sur deux
	LevelHard   DifficultyLevel = "Hard"   // Rien n'est révélé
	LevelInsane DifficultyLevel = "Insane" // Révèle, mais se recache très vite
)

type DifficultySettings struct {
	Level           DifficultyLevel
	PreviewDuration float64 // Temps avant que les tuiles ne se recachent
	PreviewRatio    float64 // Pourcentage de tuiles à montrer (1.0 = 100%)
}

func GetSettings(level DifficultyLevel) DifficultySettings {
	switch level {
	case LevelEasy:
		return DifficultySettings{Level: LevelEasy, PreviewDuration: 5.0, PreviewRatio: 1.0}
	case LevelNormal:
		return DifficultySettings{Level: LevelNormal, PreviewDuration: 3.0, PreviewRatio: 0.5}
	case LevelHard:
		return DifficultySettings{Level: LevelHard, PreviewDuration: 0.0, PreviewRatio: 0.0}
	case LevelInsane:
		return DifficultySettings{Level: LevelInsane, PreviewDuration: 1.0, PreviewRatio: 1.0}
	default:
		return GetSettings(LevelNormal)
	}
}
