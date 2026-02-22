package player

import (
	"testing"
)

func TestNewPlayer(t *testing.T) {
	p := New("player1")
	
	if p.ID != "player1" {
		t.Errorf("Expected ID 'player1', got '%s'", p.ID)
	}
	
	if p.Stats.Health != 100 {
		t.Errorf("Expected health 100, got %d", p.Stats.Health)
	}
	
	if p.Stats.MaxMana != 50 {
		t.Errorf("Expected max mana 50, got %d", p.Stats.MaxMana)
	}
	
	if p.Stats.Level != 1 {
		t.Errorf("Expected level 1, got %d", p.Stats.Level)
	}
}

func TestConsumeMana(t *testing.T) {
	p := New("player1")
	
	// Should be able to consume available mana
	if !p.ConsumeMana(20) {
		t.Error("Should be able to consume 20 mana")
	}
	
	if p.Stats.Mana != 30 {
		t.Errorf("Expected 30 mana remaining, got %d", p.Stats.Mana)
	}
	
	// Should not be able to consume more than available
	if p.ConsumeMana(50) {
		t.Error("Should not be able to consume 50 mana when only 30 available")
	}
	
	// Mana should be unchanged
	if p.Stats.Mana != 30 {
		t.Errorf("Mana should still be 30, got %d", p.Stats.Mana)
	}
}

func TestTakeDamage(t *testing.T) {
	p := New("player1")
	
	p.TakeDamage(30, "physical")
	
	if p.Stats.Health != 70 {
		t.Errorf("Expected health 70, got %d", p.Stats.Health)
	}
	
	// Test damage with resistance
	p.Skills.Resistances["physical"] = 50 // 50% resistance
	p.TakeDamage(20, "physical")
	
	// Expected: 20 - (20 * 50 / 100) = 10 damage
	if p.Stats.Health != 60 {
		t.Errorf("Expected health 60 with resistance, got %d", p.Stats.Health)
	}
}

func TestHeal(t *testing.T) {
	p := New("player1")
	
	p.TakeDamage(50, "physical")
	p.Heal(20)
	
	if p.Stats.Health != 70 {
		t.Errorf("Expected health 70, got %d", p.Stats.Health)
	}
	
	// Should not exceed max health
	p.Heal(100)
	if p.Stats.Health != 100 {
		t.Errorf("Health should not exceed max, got %d", p.Stats.Health)
	}
}

func TestRestoreMana(t *testing.T) {
	p := New("player1")
	
	p.ConsumeMana(30)
	p.RestoreMana(10)
	
	if p.Stats.Mana != 30 {
		t.Errorf("Expected mana 30, got %d", p.Stats.Mana)
	}
	
	// Should not exceed max mana
	p.RestoreMana(100)
	if p.Stats.Mana != 50 {
		t.Errorf("Mana should not exceed max, got %d", p.Stats.Mana)
	}
}

func TestIsAlive(t *testing.T) {
	p := New("player1")
	
	if !p.IsAlive() {
		t.Error("New player should be alive")
	}
	
	p.TakeDamage(200, "physical")
	
	if p.IsAlive() {
		t.Error("Player with 0 health should not be alive")
	}
}

func TestGainExperience(t *testing.T) {
	p := New("player1")
	
	p.GainExperience(50)
	
	if p.Stats.Experience != 50 {
		t.Errorf("Expected 50 XP, got %d", p.Stats.Experience)
	}
	
	if p.Stats.Level != 1 {
		t.Error("Should still be level 1")
	}
	
	// Level up at 100 XP
	p.GainExperience(50)
	
	if p.Stats.Level != 2 {
		t.Errorf("Should be level 2, got %d", p.Stats.Level)
	}
	
	if p.Stats.Experience != 0 {
		t.Errorf("XP should reset after level up, got %d", p.Stats.Experience)
	}
}

func TestLevelUp(t *testing.T) {
	p := New("player1")
	
	oldMaxHealth := p.Stats.MaxHealth
	oldMaxMana := p.Stats.MaxMana
	
	p.LevelUp()
	
	if p.Stats.MaxHealth != oldMaxHealth+10 {
		t.Errorf("Max health should increase by 10")
	}
	
	if p.Stats.MaxMana != oldMaxMana+5 {
		t.Errorf("Max mana should increase by 5")
	}
	
	// Health and mana should be restored
	if p.Stats.Health != p.Stats.MaxHealth {
		t.Error("Health should be restored to max")
	}
}

func TestUnlockAssociation(t *testing.T) {
	p := New("player1")
	
	// Player starts with "identical"
	if !p.CanAssociate("identical") {
		t.Error("Player should start with 'identical' association")
	}
	
	if p.CanAssociate("elemental") {
		t.Error("Player should not start with 'elemental' association")
	}
	
	p.UnlockAssociation("elemental")
	
	if !p.CanAssociate("elemental") {
		t.Error("Player should now have 'elemental' association")
	}
	
	// Duplicate unlock should be safe
	p.UnlockAssociation("elemental")
	
	count := 0
	for _, a := range p.Skills.UnlockedAssociations {
		if a == "elemental" {
			count++
		}
	}
	if count != 1 {
		t.Error("Should not duplicate association types")
	}
}

func TestInventory(t *testing.T) {
	inv := NewInventory(10)
	
	// Add resources
	err := inv.AddResource("wood", 3)
	if err != nil {
		t.Errorf("Failed to add resource: %v", err)
	}
	
	err = inv.AddResource("stone", 2)
	if err != nil {
		t.Errorf("Failed to add resource: %v", err)
	}
	
	if inv.GetResourceCount("wood") != 3 {
		t.Errorf("Expected 3 wood, got %d", inv.GetResourceCount("wood"))
	}
	
	if !inv.HasResource("wood") {
		t.Error("Should have wood resource")
	}
	
	if inv.HasResource("gold") {
		t.Error("Should not have gold resource")
	}
	
	// Remove resources
	err = inv.RemoveResource("wood", 1)
	if err != nil {
		t.Errorf("Failed to remove resource: %v", err)
	}
	
	if inv.GetResourceCount("wood") != 2 {
		t.Errorf("Expected 2 wood, got %d", inv.GetResourceCount("wood"))
	}
	
	// Remove all - should delete entry
	inv.RemoveResource("wood", 2)
	if inv.HasResource("wood") {
		t.Error("Should not have wood after removing all")
	}
	
	// Can't remove more than available
	err = inv.RemoveResource("stone", 10)
	if err == nil {
		t.Error("Should not be able to remove more than available")
	}
}

func TestInventoryFull(t *testing.T) {
	inv := NewInventory(5)
	
	err := inv.AddResource("item1", 5)
	if err != nil {
		t.Errorf("Failed to fill inventory: %v", err)
	}
	
	err = inv.AddResource("item2", 1)
	if err == nil {
		t.Error("Should not be able to add when inventory is full")
	}
}
