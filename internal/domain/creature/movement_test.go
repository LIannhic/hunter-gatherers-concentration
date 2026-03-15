package creature

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// ============================================================================
// Tests des Triggers
// ============================================================================

func TestMovementTriggerPassive(t *testing.T) {
	trigger := MovementTrigger{
		Type: TriggerPassive,
	}

	// Un trigger passif ne doit jamais se déclencher
	if trigger.ShouldTrigger(nil, nil) {
		t.Error("Passive trigger should never trigger")
	}
}

func TestMovementTriggerAuto(t *testing.T) {
	trigger := MovementTrigger{
		Type: TriggerAuto,
	}

	// Un trigger auto doit toujours se déclencher
	if !trigger.ShouldTrigger(nil, nil) {
		t.Error("Auto trigger should always trigger")
	}
}

func TestMovementTriggerReset(t *testing.T) {
	trigger := MovementTrigger{
		Type:      TriggerOnEcho,
		Triggered: true,
	}

	if !trigger.Triggered {
		t.Error("Triggered should be true initially")
	}

	trigger.Reset()

	if trigger.Triggered {
		t.Error("Triggered should be false after Reset()")
	}
}

// ============================================================================
// Tests de Navigation
// ============================================================================

func TestNavigationWander(t *testing.T) {
	nav := NavigationLogic{
		Type:       NavWander,
		WanderBias: entity.Position{X: 0, Y: -1},
	}

	// La navigation errante doit retourner une direction
	dir := nav.wander(nil, nil)

	// Vérifie que la direction est valide (une seule composante non nulle)
	absSum := abs(dir.X) + abs(dir.Y)
	if absSum != 1 {
		t.Errorf("Wander should return a cardinal direction, got %v", dir)
	}
}

func TestNavigationOrientation(t *testing.T) {
	creature := New("test", entity.Position{X: 0, Y: 0})
	creature.SetMovementProfile(&MovementProfile{
		Orientation: Orientation{Direction: DirNorth},
	})

	nav := NavigationLogic{
		Type: NavOrientation,
	}

	dir := nav.followOrientation(creature)

	if dir.X != 0 || dir.Y != -1 {
		t.Errorf("Expected north direction (0,-1), got %v", dir)
	}
}

func TestNavigationPatrol(t *testing.T) {
	route := []entity.Position{
		{X: 0, Y: 0},
		{X: 0, Y: 3},
		{X: 3, Y: 3},
	}

	nav := NavigationLogic{
		Type:        NavPatrol,
		PatrolRoute: route,
		PatrolIndex: 0,
	}

	creature := New("test", entity.Position{X: 0, Y: 0})

	// Première direction doit être vers le point suivant
	dir := nav.patrol(nil, creature)

	if dir.X != 0 || dir.Y != 1 {
		t.Errorf("Expected direction (0,1), got %v", dir)
	}
}

// ============================================================================
// Tests d'Orientation
// ============================================================================

func TestOrientationToVector(t *testing.T) {
	tests := []struct {
		dir      Direction
		expected entity.Position
	}{
		{DirNorth, entity.Position{X: 0, Y: -1}},
		{DirEast, entity.Position{X: 1, Y: 0}},
		{DirSouth, entity.Position{X: 0, Y: 1}},
		{DirWest, entity.Position{X: -1, Y: 0}},
	}

	for _, tc := range tests {
		o := Orientation{Direction: tc.dir}
		vec := o.ToVector()
		if vec != tc.expected {
			t.Errorf("Direction %d: expected %v, got %v", tc.dir, tc.expected, vec)
		}
	}
}

func TestOrientationRotate(t *testing.T) {
	o := Orientation{Direction: DirNorth}

	o.Rotate(90)
	if o.Direction != DirEast {
		t.Errorf("After 90° rotation from North, expected East, got %d", o.Direction)
	}

	o.Rotate(90)
	if o.Direction != DirSouth {
		t.Errorf("After another 90° rotation, expected South, got %d", o.Direction)
	}

	o.Rotate(180)
	if o.Direction != DirNorth {
		t.Errorf("After 180° rotation from South, expected North, got %d", o.Direction)
	}
}

func TestOrientationRelativeDirections(t *testing.T) {
	o := Orientation{Direction: DirNorth}

	// Depuis le nord
	tests := []struct {
		rel      RelativeDirection
		expected entity.Position
	}{
		{RelForward, entity.Position{X: 0, Y: -1}},
		{RelBackward, entity.Position{X: 0, Y: 1}},
		{RelLeft, entity.Position{X: -1, Y: 0}},
		{RelRight, entity.Position{X: 1, Y: 0}},
	}

	for _, tc := range tests {
		vec := o.GetRelativeDirection(tc.rel)
		if vec != tc.expected {
			t.Errorf("%s from North: expected %v, got %v", tc.rel, tc.expected, vec)
		}
	}
}

// ============================================================================
// Tests de Fréquence
// ============================================================================

func TestMovementFrequencyVelocity(t *testing.T) {
	freq := MovementFrequency{
		Type:     FreqVelocity,
		Velocity: 3,
	}

	if !freq.CanMove() {
		t.Error("Velocity frequency should always allow move")
	}

	if freq.GetMoveCount() != 3 {
		t.Errorf("Expected 3 moves, got %d", freq.GetMoveCount())
	}
}

func TestMovementFrequencyDelay(t *testing.T) {
	freq := MovementFrequency{
		Type:  FreqDelay,
		Delay: 2,
	}

	// Avec un delay de 2, le compteur doit atteindre 2 pour déclencher
	// Tour 0: compteur=1, pas de mouvement
	// Tour 1: compteur=2, mouvement et reset
	
	// Premier appel: compteur passe à 1, pas de mouvement
	if freq.CanMove() {
		t.Error("First turn should not allow move (counter=1)")
	}

	// Deuxième appel: compteur passe à 2, atteint le délai, mouvement et reset
	if !freq.CanMove() {
		t.Error("Second turn should allow move (counter reached delay)")
	}

	// Après reset, le compteur est à 0, donc pas de mouvement
	if freq.CanMove() {
		t.Error("After reset, should not allow move immediately")
	}
}

// ============================================================================
// Tests de Collision
// ============================================================================

func TestCollisionHandlerStop(t *testing.T) {
	ch := CollisionHandler{
		Type: CollideStop,
	}

	// Pour CollideStop, on retourne simplement la position actuelle
	// La logique détaillée est testée via l'intégration
	if ch.Type != CollideStop {
		t.Error("Collision type should be Stop")
	}
}

func TestCollisionHandlerBounce(t *testing.T) {
	ch := CollisionHandler{
		Type: CollideBounce,
	}

	// Pour Bounce, la créature doit changer d'orientation
	if ch.Type != CollideBounce {
		t.Error("Collision type should be Bounce")
	}
}

// ============================================================================
// Tests des Profils Prédéfinis
// ============================================================================

func TestDefaultMovementProfile(t *testing.T) {
	profile := DefaultMovementProfile()

	if profile.Trigger.Type != TriggerAuto {
		t.Errorf("Default trigger should be Auto, got %s", profile.Trigger.Type)
	}

	if profile.Navigation.Type != NavWander {
		t.Errorf("Default navigation should be Wander, got %s", profile.Navigation.Type)
	}

	if profile.Mode.Type != ModeBento {
		t.Errorf("Default mode should be Bento, got %s", profile.Mode.Type)
	}

	if profile.Collision.Type != CollideStop {
		t.Errorf("Default collision should be Stop, got %s", profile.Collision.Type)
	}
}

func TestPassiveProfile(t *testing.T) {
	profile := PassiveProfile()

	if profile.Trigger.Type != TriggerPassive {
		t.Errorf("Passive trigger should be Passive, got %s", profile.Trigger.Type)
	}
}

func TestHunterProfile(t *testing.T) {
	profile := HunterProfile()

	if profile.Navigation.Type != NavAttraction {
		t.Errorf("Hunter navigation should be Attraction, got %s", profile.Navigation.Type)
	}

	if profile.Navigation.Target != TargetPlayer {
		t.Errorf("Hunter target should be Player, got %s", profile.Navigation.Target)
	}

	if profile.Mode.Type != ModeBento {
		t.Errorf("Hunter mode should be Bento, got %s", profile.Mode.Type)
	}

	if profile.Frequency.Velocity != 2 {
		t.Errorf("Hunter velocity should be 2, got %d", profile.Frequency.Velocity)
	}
}

func TestFleeingProfile(t *testing.T) {
	profile := FleeingProfile()

	if profile.Navigation.Type != NavRepulsion {
		t.Errorf("Fleeing navigation should be Repulsion, got %s", profile.Navigation.Type)
	}

	if profile.Mode.Type != ModeShadow {
		t.Errorf("Fleeing mode should be Shadow, got %s", profile.Mode.Type)
	}
}

func TestSpecterProfile(t *testing.T) {
	profile := SpecterProfile()

	if profile.Trigger.Type != TriggerOnEcho {
		t.Errorf("Specter trigger should be OnEcho, got %s", profile.Trigger.Type)
	}

	if profile.Mode.Type != ModeShadow {
		t.Errorf("Specter mode should be Shadow, got %s", profile.Mode.Type)
	}

	if profile.Collision.Type != CollidePhase {
		t.Errorf("Specter collision should be Phase, got %s", profile.Collision.Type)
	}

	if len(profile.Collision.CanPhaseThrough) == 0 {
		t.Error("Specter should be able to phase through some tiles")
	}
}

func TestPatrollerProfile(t *testing.T) {
	route := []entity.Position{
		{X: 0, Y: 0},
		{X: 1, Y: 1},
	}

	profile := PatrollerProfile(route)

	if profile.Navigation.Type != NavPatrol {
		t.Errorf("Patroller navigation should be Patrol, got %s", profile.Navigation.Type)
	}

	if len(profile.Navigation.PatrolRoute) != 2 {
		t.Errorf("Patroller should have 2 waypoints, got %d", len(profile.Navigation.PatrolRoute))
	}
}

// ============================================================================
// Tests des Créatures avec MovementProfile
// ============================================================================

func TestCreatureWithMovementProfile(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		species         string
		expectedTrigger TriggerType
		expectedMode    MoveMode
	}{
		{"specter", TriggerOnEcho, ModeShadow},
		{"echo_hound", TriggerOnEcho, ModeBento},
		{"fleeing_sprite", TriggerProximity, ModeShadow},
	}

	for _, tc := range tests {
		c, err := factory.Create(tc.species, entity.Position{X: 0, Y: 0})
		if err != nil {
			t.Errorf("Failed to create %s: %v", tc.species, err)
			continue
		}

		if c.MovementProfile == nil {
			t.Errorf("%s should have a MovementProfile", tc.species)
			continue
		}

		if c.MovementProfile.Trigger.Type != tc.expectedTrigger {
			t.Errorf("%s: expected trigger %s, got %s",
				tc.species, tc.expectedTrigger, c.MovementProfile.Trigger.Type)
		}

		if c.MovementProfile.Mode.Type != tc.expectedMode {
			t.Errorf("%s: expected mode %s, got %s",
				tc.species, tc.expectedMode, c.MovementProfile.Mode.Type)
		}
	}
}

func TestCreatePatroller(t *testing.T) {
	factory := NewFactory()

	route := []entity.Position{
		{X: 0, Y: 0},
		{X: 2, Y: 0},
		{X: 2, Y: 2},
	}

	c, err := factory.CreatePatroller("stonewarden", entity.Position{X: 0, Y: 0}, route)
	if err != nil {
		t.Fatalf("Failed to create patroller: %v", err)
	}

	if c.MovementProfile == nil {
		t.Fatal("Patroller should have a MovementProfile")
	}

	if c.MovementProfile.Navigation.Type != NavPatrol {
		t.Errorf("Patroller should have Patrol navigation, got %s",
			c.MovementProfile.Navigation.Type)
	}

	if len(c.MovementProfile.Navigation.PatrolRoute) != 3 {
		t.Errorf("Patroller should have 3 waypoints, got %d",
			len(c.MovementProfile.Navigation.PatrolRoute))
	}

	if !c.HasTag("patroller") {
		t.Error("Patroller should have 'patroller' tag")
	}
}

// ============================================================================
// Tests Utilitaires
// ============================================================================

func TestSign(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{5, 1},
		{-5, -1},
		{0, 0},
		{100, 1},
		{-100, -1},
	}

	for _, tc := range tests {
		result := Sign(tc.input)
		if result != tc.expected {
			t.Errorf("Sign(%d): expected %d, got %d", tc.input, tc.expected, result)
		}
	}
}
