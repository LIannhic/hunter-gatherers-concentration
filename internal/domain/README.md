# Domain Layer - Architecture et Design Patterns

Ce document décrit l'architecture et les patterns de conception utilisés dans la couche domain du jeu "Hunter Gatherers Concentration".

## Vue d'ensemble

Le domaine est organisé selon une architecture **Clean Architecture** avec une séparation claire des responsabilités. Le code est réparti dans des sous-packages thématiques pour faciliter la maintenance et les tests.

```
┌─────────────────────────────────────────────────────────────┐
│                        Domain Layer                          │
├─────────────┬─────────────┬─────────────┬───────────────────┤
│   Entity    │   Board     │ Component   │    Systems        │
│  (Identity) │  (Grid)     │   (Data)    │  (Logic/Update)   │
├─────────────┼─────────────┼─────────────┼───────────────────┤
│  Creature   │  Resource   │   Player    │      Meta         │
│  (Behavior) │  (Items)    │  (Stats)    │ (Progression)     │
├─────────────┴─────────────┴─────────────┴───────────────────┤
│                      Event Bus                               │
│                 (Communication)                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 1. Entity-Component-System (ECS)

### Pourquoi ECS ?

Le jeu nécessite une grande flexibilité : des entités (ressources, créatures) avec des comportements variés qui évoluent au fil du temps. L'ECS permet :
- **Composition over inheritance** : pas de hiérarchie de classes complexe
- **Flexibilité** : ajouter/retirer des comportements dynamiquement
- **Performance** : les systèmes traitent les données de manière cache-friendly

### Structure ECS

```go
// Entity - Identité unique + position
type Entity interface {
    GetID() ID
    GetType() Type
    GetPosition() Position
    // ...
}

// Component - Données pures, pas de logique
type Lifecycle struct {
    CurrentStage int
    MaxStages    int
    StageNames   []string
}

// System - Logique métier qui opère sur les composants
type LifecycleSystem struct{}
func (s *LifecycleSystem) Update(world *World) {
    // Met à jour tous les composants Lifecycle
}
```

### Implémentation

- **`entity/`** : Gestion des identités (`ID`, `Manager`)
- **`component/`** : Stockage et définition des composants (`Store`)
- **`system.go`** : Systèmes qui traitent les données (`LifecycleSystem`, `CreatureAISystem`)

---

## 2. Factory Pattern

### Objectif

Créer des entités préconfigurées avec des valeurs par défaut cohérentes sans polluer le code métier avec des constructeurs complexes.

### Exemple : Création de créatures

```go
type Factory struct{}

func (f *Factory) Create(species string, pos entity.Position) (*Creature, error) {
    c := New(species, pos)
    
    switch species {
    case "lumifly":
        c.SetBehavior(component.Behavior{State: "pollinating"})
        c.SetMobility(component.Mobility{CanMove: true})
        c.AddTag("flying")
        
    case "shadowstalker":
        c.SetBehavior(component.Behavior{State: "hunting", Aggression: 80})
        c.AddTag("dangerous")
    }
    
    return c, nil
}
```

### Avantages

- **Centralisation** : La logique de création est au même endroit
- **Extensibilité** : Ajouter une nouvelle espèce = ajouter un case
- **Testabilité** : Facile de mocker les factories

---

## 3. Strategy Pattern (Associations)

### Objectif

Le cœur du jeu est le mécanisme d'association de tuiles (Memory). Différents types d'associations existent :
- **Identical** : Paire identique
- **Logical** : Clé/Serrure, Marteau/Enclume
- **Elemental** : Feu + Bois, Eau + Plante
- **Narrative** : Fragments d'histoire

Le pattern Strategy permet de traiter ces différents types uniformément.

### Implémentation

```go
type Strategy interface {
    Type() Type
    CanAssociate(a, b Matchable) bool
    Resolve(a, b Matchable) (Result, error)
}

// Implémentations concrètes
type IdenticalStrategy struct{}
type LogicalStrategy struct{}
type ElementalStrategy struct{}

// Engine qui orchestre
func (e *Engine) TryAssociate(a, b Matchable) (Result, error) {
    for _, strategy := range e.strategies {
        if strategy.CanAssociate(a, b) {
            return strategy.Resolve(a, b)
        }
    }
    return Result{Success: false}, errors.New("aucune association")
}
```

### Extensibilité

Ajouter un nouveau type d'association :
```go
type TemporalStrategy struct{} // Associations basées sur le temps
engine.RegisterStrategy(&TemporalStrategy{})
```

---

## 4. Observer Pattern (Event Bus)

### Objectif

Découpler les systèmes qui produisent des événements de ceux qui les consomment. Éviter les dépendances cycliques.

### Cas d'usage

- Une créature se déplace → Le système de déplacement publie un événement
- Le système de score peut écouter et attribuer des points
- Le système d'affichage peut mettre à jour l'UI

### Implémentation

```go
// Publication
eventBus.Publish(event.NewCreatureMovedEvent(creatureID, from, to))

// Souscription
eventBus.SubscribeFunc(CreatureMoved, func(e Event) {
    // Réagir au mouvement
})

// Traitement batch (éviter les effets de bord en cascade)
eventBus.ProcessQueue()
```

### Types d'événements

```go
CreatureMoved      // Déplacement
ResourceMatured    // Changement de stade
AssociationMade    // Paire trouvée
PlayerDamaged      // Dégâts subis
TurnEnded          // Fin de tour
```

---

## 5. Adapter Pattern

### Objectif

Adapter l'interface `World` pour l'IA des créatures sans exposer tout l'état du monde.

### Implémentation

```go
// Interface minimale pour l'IA
type WorldState interface {
    GetPlayerPosition() entity.Position
    GetNearbyCreatures(pos entity.Position, radius int) []*Creature
    IsValidMove(pos entity.Position) bool
}

// Adaptateur
type worldAdapter struct {
    world *World
}

func (wa *worldAdapter) GetPlayerPosition() entity.Position {
    return wa.world.playerPosition
}
// ... implémentations limitées
```

### Avantages

- **Principe de moindre privilège** : L'IA n'a accès qu'à ce dont elle a besoin
- **Testabilité** : Facile de créer un mock WorldState pour tester l'IA

---

## 6. Repository Pattern (Entity Manager)

### Objectif

Abstraire le stockage et la récupération des entités.

### Implémentation

```go
type Manager struct {
    entities map[ID]Entity      // Accès par ID
    byType   map[Type]map[ID]Entity  // Index par type
    byPos    map[Position]ID         // Index spatial
}

func (m *Manager) Get(id ID) (Entity, bool)
func (m *Manager) GetByPosition(pos Position) (Entity, bool)
func (m *Manager) GetByType(t Type) []Entity
func (m *Manager) QueryByTag(tag string) []Entity
```

### Indexation spatiale

Le `byPos` permet des requêtes rapides : "Quelle entité est à la position (3,4) ?"

---

## 7. State Pattern (Créatures)

### Objectif

Les créatures changent de comportement selon leur état (chasse, fuite, pollinisation).

### Implémentation

```go
// L'état est stocké dans le composant Behavior
type Behavior struct {
    State string // "idle", "hunting", "fleeing", "pollinating"
}

// L'AI utilise l'état pour décider
func (ai *SimpleAI) Decide(c *Creature, world WorldState) Action {
    switch c.Behavior.State {
    case "fleeing":
        return ai.flee(c, world)
    case "hunting":
        return ai.hunt(c, world)
    default:
        return ai.idle(c, world)
    }
}
```

---

## Flux de données

```
1. Joueur révèle une tuile
   ↓
2. World.RevealTile() → Événement TileRevealed
   ↓
3. Systèmes écoutent et réagissent
   ↓
4. Engine.Update() progresse d'un tour
   ↓
5. LifecycleSystem : les ressources mûrissent
   CreatureAISystem : les créatures se déplacent
   TriggerSystem : vérifie les conditions de déclenchement
   ↓
6. EventBus.ProcessQueue() traite les événements
   ↓
7. Mise à jour de l'affichage
```

---

## Testing

L'architecture facilite les tests unitaires :

```go
// Test d'une stratégie d'association
func TestIdenticalStrategy(t *testing.T) {
    strategy := &IdenticalStrategy{}
    a := &MockMatchable{matchID: "card1"}
    b := &MockMatchable{matchID: "card1"}
    
    if !strategy.CanAssociate(a, b) {
        t.Error("Should associate identical items")
    }
}

// Test de l'AI avec un WorldState mocké
func TestCreatureAI(t *testing.T) {
    mockWorld := &MockWorldState{playerPos: Position{X: 5, Y: 5}}
    creature := NewCreature("test", Position{X: 0, Y: 0})
    
    action := ai.Decide(creature, mockWorld)
    // Vérifier l'action
}
```

---

## Bonnes pratiques

1. **Données séparées de la logique** : Les composants sont des structs simples, la logique est dans les systèmes
2. **Immutabilité préférée** : Les événements sont immutables, les composants sont modifiés par les systèmes
3. **Pas de dépendances circulaires** : Les packages dépendent de `entity` et `component`, jamais l'inverse
4. **Interfaces minimales** : `WorldState` pour l'IA, `Matchable` pour les associations

---

## Ajouter une fonctionnalité

### Exemple : Ajouter un nouveau système

```go
// 1. Créer le système
 type WeatherSystem struct{}
 
 func (s *WeatherSystem) Priority() int { return 5 }
 
 func (s *WeatherSystem) Update(world *World) {
     // Modifier les ressources selon la météo
 }
 
// 2. L'enregistrer dans l'Engine
 engine := NewEngine(world)
 engine.AddSystem(&WeatherSystem{})
```

### Exemple : Ajouter un nouveau composant

```go
// 1. Définir le composant
 type WeatherSensitive struct {
     PreferredWeather string
 }
 func (w WeatherSensitive) Type() string { return "weather_sensitive" }

// 2. L'ajouter aux entités concernées
 resource.Components.Add(entityID, &WeatherSensitive{PreferredWeather: "rain"})

// 3. Le système Weather peut le lire
 if comp, ok := components.Get(id, "weather_sensitive"); ok {
     ws := comp.(*WeatherSensitive)
     // Appliquer les effets
 }
```
