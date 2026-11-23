package locales

import "embed"

// FS embeds all JSON locale files in internal/infrastructure/assets/locales.
//go:embed *.json
var FS embed.FS
