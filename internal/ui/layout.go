package ui

// Layout constants for 1280x720 screen
const (
	ScreenWidth  = 1280
	ScreenHeight = 720
)

// Portrait Holder
const (
	PortraitX = 10
	PortraitY = 10
	PortraitW = 270
	PortraitH = 270

	MenuIconRelativeX = 5
	MenuIconRelativeY = 5
	MenuIconSize      = 43.75

	FullscreenIconRelativeX = 53.75
	FullscreenIconRelativeY = 5
	FullscreenIconSize      = 43.75
)

// Inventory Panel
const (
	InventoryX = 10
	InventoryY = 340
	InventoryW = 270
	InventoryH = 370

	LootSlotSize    = 87.5
	LootSlotPadding = 1.875
	LootSlotsPerRow = 3

	LootCounterRelativeX = 5
	LootCounterRelativeY = 321
	LootCounterSize      = 43.75

	DeleteLootRelativeX = 221
	DeleteLootRelativeY = 321
	DeleteLootSize      = 43.75
)

// Playmat
const (
	PlaymatX = 290
	PlaymatY = 10
	PlaymatW = 700
	PlaymatH = 700

	// Quake effect : le snapshot est plus grand pour éviter les espaces vides lors de la rotation
	QuakePadding = 145 // 700 * sqrt(2) ≈ 990, padding = (990-700)/2 = 145
	QuakeSnapW   = PlaymatW + QuakePadding*2 // 990
	QuakeSnapH   = PlaymatH + QuakePadding*2 // 990

	ActionButtonW = 219.67
	ActionButtonH = 39.17

	// Action buttons relative positions
	ActionBtn1X = 10
	ActionBtn1Y = 10
	ActionBtn2X = 470
	ActionBtn2Y = 10
	ActionBtn3X = 10
	ActionBtn3Y = 651
	ActionBtn4X = 470
	ActionBtn4Y = 651

	ButtonTextRelativeX = 10
	ButtonTextRelativeY = 8.65
	ButtonTextW         = 165.5
	ButtonTextH         = 21.88

	ButtonIconRelativeX = 180.5
	ButtonIconRelativeY = 5
	ButtonIconSize      = 29.17

	// Exits
	ExitNorthX = 262.5
	ExitNorthY = 0
	ExitNorthW = 175
	ExitNorthH = 87.5

	ExitEastX = 612.5
	ExitEastY = 262.5
	ExitEastW = 87.5
	ExitEastH = 175

	ExitSouthX = 262.5
	ExitSouthY = 612.5
	ExitSouthW = 175
	ExitSouthH = 87.5

	ExitWestX = 0
	ExitWestY = 262.5
	ExitWestW = 87.5
	ExitWestH = 175

	// Board
	BoardRelativeX = 87.5
	BoardRelativeY = 87.5
	BoardW         = 525
	BoardH         = 525

	TileSize = 87.5

	// Flip animation characteristics
	FaceSize      = 80.0
	FlipThickness = 4.0
)

// Gauges Holder
const (
	GaugesX = 1000
	GaugesY = 10
	GaugesW = 270
	GaugesH = 370

	GaugeW = 76.67
	GaugeH = 350

	HealthGaugeRelativeX = 10
	HealthGaugeRelativeY = 10

	ManaGaugeRelativeX = 97.65
	ManaGaugeRelativeY = 10

	SanityGaugeRelativeX = 183.32
	SanityGaugeRelativeY = 10
)

// Message Boxes
const (
	MessageBoxXLeft    = 10
	MessageBoxYLeft    = 290
	MessageBoxWLeft    = 270
	MessageBoxHLeft    = 40
	MessageBoxXRight   = 1000
	MessageBoxYRight   = 390
	MessageBoxWRight   = 270
	MessageBoxHRight   = 40
)

// Minimap
const (
	MinimapX = 1000
	MinimapY = 440
	MinimapW = 270
	MinimapH = 270
)
