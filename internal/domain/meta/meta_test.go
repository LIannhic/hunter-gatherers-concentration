package meta

import (
	"testing"
)

func TestNewFamily(t *testing.T) {
	f := NewFamily()
	
	if len(f.Members) != 2 {
		t.Errorf("Expected 2 family members, got %d", len(f.Members))
	}
	
	if len(f.Needs) != 3 {
		t.Errorf("Expected 3 needs, got %d", len(f.Needs))
	}
	
	if f.Day != 1 {
		t.Errorf("Expected day 1, got %d", f.Day)
	}
	
	if f.Debt != 1000 {
		t.Errorf("Expected debt 1000, got %d", f.Debt)
	}
}

func TestFamilyAllocate(t *testing.T) {
	f := NewFamily()
	
	err := f.Allocate("food", 30)
	if err != nil {
		t.Errorf("Failed to allocate: %v", err)
	}
	
	foodNeed, _ := f.GetNeed("food")
	if foodNeed.Current != 80 { // 50 + 30
		t.Errorf("Expected food 80, got %d", foodNeed.Current)
	}
	
	// Can't exceed required
	err = f.Allocate("food", 100)
	if err != nil {
		t.Errorf("Failed to allocate: %v", err)
	}
	
	foodNeed, _ = f.GetNeed("food")
	if foodNeed.Current != 100 { // Should be capped at required
		t.Errorf("Expected food capped at 100, got %d", foodNeed.Current)
	}
	
	// Unknown need
	err = f.Allocate("unknown_need", 10)
	if err == nil {
		t.Error("Should fail for unknown need")
	}
}

func TestFamilyPayDebt(t *testing.T) {
	f := NewFamily()
	
	f.PayDebt(300)
	
	if f.Debt != 700 {
		t.Errorf("Expected debt 700, got %d", f.Debt)
	}
	
	// Pay more than debt
	f.PayDebt(1000)
	
	if f.Debt != 0 {
		t.Errorf("Debt should be 0, got %d", f.Debt)
	}
}

func TestFamilyNextDay(t *testing.T) {
	f := NewFamily()
	
	initialFood := f.Needs[0].Current
	initialHealth := f.Needs[2].Current
	
	f.NextDay()
	
	if f.Day != 2 {
		t.Errorf("Expected day 2, got %d", f.Day)
	}
	
	// Food should decrease by 20
	if f.Needs[0].Current != initialFood-20 {
		t.Errorf("Expected food %d, got %d", initialFood-20, f.Needs[0].Current)
	}
	
	// Health should decrease by 5
	if f.Needs[2].Current != initialHealth-5 {
		t.Errorf("Expected health %d, got %d", initialHealth-5, f.Needs[2].Current)
	}
}

func TestFamilyUrgencyUpdate(t *testing.T) {
	f := NewFamily()
	
	// Set food to critical level
	f.Needs[0].Current = 10 // Below 20% of 100
	f.NextDay() // This should update urgency
	
	foodNeed, _ := f.GetNeed("food")
	if foodNeed.Urgency != 5 {
		t.Errorf("Expected urgency 5 (critical), got %d", foodNeed.Urgency)
	}
}

func TestFamilyIsStable(t *testing.T) {
	f := NewFamily()
	
	// Make all needs stable (urgency < 4)
	for i := range f.Needs {
		f.Needs[i].Urgency = 3
	}
	
	// Now family should be stable
	if !f.IsStable() {
		t.Error("Family should be stable when all needs have urgency < 4")
	}
	
	// Make food critical
	f.Needs[0].Urgency = 5
	
	if f.IsStable() {
		t.Error("Family with critical need should not be stable")
	}
}

func TestNewMetaProgression(t *testing.T) {
	m := NewMetaProgression()
	
	if len(m.UnlockedZones) != 1 {
		t.Errorf("Expected 1 unlocked zone, got %d", len(m.UnlockedZones))
	}
	
	if m.UnlockedZones[0] != "twilight_woods" {
		t.Error("Expected twilight_woods to be unlocked")
	}
	
	if len(m.UnlockedRecipes) != 1 {
		t.Errorf("Expected 1 unlocked recipe, got %d", len(m.UnlockedRecipes))
	}
	
	if m.Reputation != 0 {
		t.Errorf("Expected reputation 0, got %d", m.Reputation)
	}
}

func TestMetaProgressionUnlockZone(t *testing.T) {
	m := NewMetaProgression()
	
	if m.IsZoneUnlocked("dark_cave") {
		t.Error("Dark cave should not be unlocked initially")
	}
	
	m.UnlockZone("dark_cave")
	
	if !m.IsZoneUnlocked("dark_cave") {
		t.Error("Dark cave should be unlocked")
	}
	
	// Duplicate unlock should be safe
	m.UnlockZone("dark_cave")
	
	count := 0
	for _, z := range m.UnlockedZones {
		if z == "dark_cave" {
			count++
		}
	}
	if count != 1 {
		t.Error("Zone should not be duplicated")
	}
}

func TestMetaProgressionKnowledge(t *testing.T) {
	m := NewMetaProgression()
	
	m.AddKnowledge("wolf", 10)
	if m.GetKnowledge("wolf") != 10 {
		t.Errorf("Expected knowledge 10, got %d", m.GetKnowledge("wolf"))
	}
	
	m.AddKnowledge("wolf", 5)
	if m.GetKnowledge("wolf") != 15 {
		t.Errorf("Expected knowledge 15, got %d", m.GetKnowledge("wolf"))
	}
}

func TestNewHub(t *testing.T) {
	h := NewHub()
	
	if h.Family == nil {
		t.Error("Hub should have a family")
	}
	
	if h.Progression == nil {
		t.Error("Hub should have progression")
	}
	
	if h.Inventory == nil {
		t.Error("Hub should have inventory")
	}
}

func TestHubPrepareMission(t *testing.T) {
	h := NewHub()
	
	// Add items to inventory
	h.Inventory["sword"] = 1
	h.Inventory["shield"] = 1
	
	items := h.PrepareMission([]string{"sword"})
	
	if items["sword"] != 1 {
		t.Error("Should have sword")
	}
	
	if items["ration"] != 2 {
		t.Error("Should have 2 rations")
	}
	
	if items["shield"] != 0 {
		t.Error("Should not have shield (not selected)")
	}
	
	// Inventory should be reduced
	if h.Inventory["sword"] != 0 {
		t.Error("Sword should be removed from hub inventory")
	}
}

func TestHubReturnFromMission(t *testing.T) {
	h := NewHub()
	
	loot := map[string]int{
		"gold":   100,
		"herbs":  5,
	}
	
	initialDay := h.Family.Day
	
	h.ReturnFromMission(loot, true)
	
	if h.Inventory["gold"] != 100 {
		t.Errorf("Expected 100 gold in inventory, got %d", h.Inventory["gold"])
	}
	
	if h.Family.Day != initialDay+1 {
		t.Error("Day should advance after successful mission")
	}
}

func TestHubReturnFromMissionFailure(t *testing.T) {
	f := NewFamily()
	h := &Hub{Family: f}
	
	initialFood := f.Needs[0].Current
	
	h.ReturnFromMission(nil, false)
	
	if f.Needs[0].Current != initialFood-30 {
		t.Error("Food should decrease on failed mission")
	}
}

func TestHubAddToInventory(t *testing.T) {
	h := NewHub()
	
	h.AddToInventory("gem", 5)
	
	if h.Inventory["gem"] != 5 {
		t.Errorf("Expected 5 gems, got %d", h.Inventory["gem"])
	}
	
	h.AddToInventory("gem", 3)
	
	if h.Inventory["gem"] != 8 {
		t.Errorf("Expected 8 gems, got %d", h.Inventory["gem"])
	}
}
