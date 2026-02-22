package resource

import (
	"math/rand"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// Resource est une entité récoltable
type Resource struct {
	entity.BaseEntity
	ResourceType string // "plant", "mineral", "organic", "ethereal"
	Lifecycle    component.Lifecycle
	Value        component.Value
	Visual       component.Visual
	Matchable    component.Matchable
}

func New(rtype string, pos entity.Position) *Resource {
	r := &Resource{
		BaseEntity:   entity.NewBaseEntity(entity.TypeResource),
		ResourceType: rtype,
	}
	r.SetPosition(pos)
	r.AddTag("resource")
	r.AddTag(rtype)
	return r
}

func (r *Resource) SetLifecycle(l component.Lifecycle) {
	r.Lifecycle = l
}

func (r *Resource) SetValue(v component.Value) {
	r.Value = v
}

func (r *Resource) SetVisual(v component.Visual) {
	r.Visual = v
}

func (r *Resource) SetMatchable(m component.Matchable) {
	r.Matchable = m
}

func (r *Resource) GetMatchID() string       { return r.Matchable.MatchID }
func (r *Resource) GetLogicKey() string      { return r.Matchable.LogicKey }
func (r *Resource) GetElement() string       { return r.Matchable.Element }
func (r *Resource) GetNarrativeTag() string  { return r.Matchable.NarrativeTag }
func (r *Resource) GetMatchTypes() []string { return r.Matchable.MatchTypes }

// Update appelé à chaque tour
func (r *Resource) Update() {
	if r.Lifecycle.Progress() {
		// Le stade a changé, met à jour la valeur
		r.updateValueByStage()
	}

	// Dégradation naturelle
	if r.Value.DegradeRate > 0 {
		r.Value.CurrentValue -= r.Value.DegradeRate
		if r.Value.CurrentValue < 0 {
			r.Value.CurrentValue = 0
		}
	}
}

func (r *Resource) updateValueByStage() {
	stage := r.Lifecycle.GetCurrentStageName()
	switch stage {
	case "bourgeon":
		r.Value.CurrentValue = r.Value.BaseValue / 4
	case "jeune":
		r.Value.CurrentValue = r.Value.BaseValue / 2
	case "fruit", "mûr":
		r.Value.CurrentValue = r.Value.BaseValue
	case "gâté", "pourri":
		r.Value.CurrentValue = r.Value.BaseValue / 10
	}
}

func (r *Resource) CanPropagate() bool {
	return r.Lifecycle.CanPropagate && r.Lifecycle.CurrentStage >= 1
}

func (r *Resource) IsHarvestable() bool {
	return r.Value.CurrentValue > 0
}

func (r *Resource) GetHarvestValue() int {
	return r.Value.CurrentValue
}

// Factory pour créer des ressources
type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) Create(rtype string, pos entity.Position) *Resource {
	r := New(rtype, pos)

	switch rtype {
	case "dreamberry":
		r.SetLifecycle(component.Lifecycle{
			CurrentStage: 0,
			MaxStages:    4,
			StageNames:   []string{"bourgeon", "fleur", "fruit", "gâté"},
			TurnsToNext:  3,
			CanPropagate: true,
		})
		r.SetValue(component.Value{
			BaseValue:    100,
			CurrentValue: 25,
			DegradeRate:  5,
		})
		r.SetMatchable(component.Matchable{
			MatchID:    "dreamberry",
			MatchTypes: []string{"identical", "elemental"},
			Element:    "ethereal",
		})

	case "moonstone":
		r.SetLifecycle(component.Lifecycle{
			CurrentStage: 1,
			MaxStages:    3,
			StageNames:   []string{"brute", "taillée", "polie"},
			TurnsToNext:  -1, // Ne change pas seul
			CanPropagate: false,
		})
		r.SetValue(component.Value{
			BaseValue:    200,
			CurrentValue: 200,
			DegradeRate:  0,
		})
		r.SetMatchable(component.Matchable{
			MatchID:    "moonstone",
			LogicKey:   "mineral",
			MatchTypes: []string{"identical", "logical"},
		})

	case "whispering_herb":
		r.SetLifecycle(component.Lifecycle{
			CurrentStage: 0,
			MaxStages:    3,
			StageNames:   []string{"graine", "pousse", "mature"},
			TurnsToNext:  2,
			CanPropagate: false,
		})
		r.SetValue(component.Value{
			BaseValue:    50,
			CurrentValue: 10,
			DegradeRate:  2,
		})
		r.SetMatchable(component.Matchable{
			NarrativeTag: "healing",
			MatchTypes:   []string{"narrative", "elemental"},
			Element:      "life",
		})

	case "crystal_shard":
		r.SetLifecycle(component.Lifecycle{
			CurrentStage: 0,
			MaxStages:    2,
			StageNames:   []string{"brut", "purifié"},
			TurnsToNext:  -1,
			CanPropagate: false,
		})
		r.SetValue(component.Value{
			BaseValue:    150,
			CurrentValue: 150,
			DegradeRate:  0,
		})
		r.SetMatchable(component.Matchable{
			MatchID:    "crystal_shard",
			MatchTypes: []string{"identical"},
			Element:    "ethereal",
		})
	}

	return r
}

func (f *Factory) CreateRandom(pos entity.Position) *Resource {
	types := []string{"dreamberry", "moonstone", "whispering_herb", "crystal_shard"}
	return f.Create(types[rand.Intn(len(types))], pos)
}
