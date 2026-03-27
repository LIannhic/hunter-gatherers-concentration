# Assets - Hunter Gatherers Concentration

⚠️ **ATTENTION : ASSETS TEMPORAIRES / PLACEHOLDERS**

Tous les assets visuels actuels sont **générés procéduralement par le code** et servent de 
**placeholders temporaires** pour le développement. Ils seront remplacés par des assets
finaux (sprites, pixel art, illustrations) avant la release.

## Statut des Assets

| Type | Statut | Priorité | Notes |
|------|--------|----------|-------|
| Tuiles avec thèmes | 🟡 Temporaire | Moyenne | Thèmes : Default, Forest, Cave |
| Ressources | 🟡 Temporaire | Haute | Génération procédurale |
| Créatures | 🟡 Temporaire | Haute | Génération procédurale |
| Effets de flip | 🟡 Temporaire | Basse | États : Hidden, Revealed, Matched |
| Audio | 🟡 Temporaire | Moyenne | Non implémenté |

## Avantages des Assets Temporaires

- ✅ **Développement rapide** - Pas besoin d'attendre les assets finaux
- ✅ **Pas de dépendances** - Aucun fichier externe à gérer
- ✅ **Taille réduite** - Génération à la volée
- ✅ **Modifiables** - Ajustement facile via code
- ✅ **Libres de droit** - CC0 / Domaine public

## Remplacement par des Assets Finaux

Pour remplacer un asset temporaire :

1. Ajouter le fichier image dans `assets/images/`
2. Modifier `internal/infrastructure/assets/manager.go` pour charger le fichier
3. Conserver la génération procédurale comme fallback

### Exemple de chargement d'asset externe :

```go
// Charger une image externe
img, _, err := ebitenutil.NewImageFromFile("assets/images/dreamberry.png")
if err == nil {
    m.images["resource_dreamberry"] = img
} else {
    // Fallback sur la génération procédurale
    m.images["resource_dreamberry"] = generateDreamberry(size, DreamberryPalette)
}
```

## 🎨 Nature des Assets

Contrairement à de nombreux jeux qui utilisent des images externes, tous les graphiques de ce jeu sont créés dynamiquement via des algorithmes de génération procédurale. Cela signifie :

- ✅ **Aucune dépendance externe** - Pas de fichiers image à gérer
- ✅ **Taille réduite** - Le code génère les images à la volée
- ✅ **Modifiable à l'infini** - Changez les paramètres pour obtenir des variantes
- ✅ **Libre de droit garanti** - Vous êtes propriétaire du code générateur

## 🗂️ Structure des Assets

### 1. Tuiles de Jeu (`internal/infrastructure/assets/tiles.go`)

Les tuiles utilisent un système de thèmes pour s'adapter à différents environnements :

| Thème | Description | Utilisation |
|-------|-------------|-------------|
| `default` | Bleu-violet classique | Grille par défaut |
| `forest` | Vert naturel | Forêts, bois |
| `cave` | Sombre gris-violet | Cavernes, grottes |
| `swamp` | Vert mystique | Marais, zones humides |

Chaque tuile comprend :
- **Face cachée** : Motif géométrique décoratif avec bordure
- **Face révélée** : Fond avec grille subtile
- **Face appairée** : Rayons de succès verts

### 2. Ressources (`internal/infrastructure/assets/resources.go`)

| Ressource | Description | Palette |
|-----------|-------------|---------|
| `dreamberry` | Baie onirique violette | Violets |
| `moonstone` | Pierre de lune bleutée | Bleus |
| `whispering_herb` | Herbe murmurante | Verts |
| `shadow_essence` | Essence d'ombre | Mauves sombres |
| `crystal_shard` | Éclat de cristal | Cyans |

### 3. Créatures (`internal/infrastructure/assets/creatures.go`)

| Créature | Description | Palette |
|----------|-------------|---------|
| `lumifly` | Luciole lumineuse | Jaune doré |
| `shadowstalker` | Rôdeur des ombres | Violet sombre |
| `burrower` | Fouisseur terrestre | Brun terreux |
| `flutterwing` | Papillon aérien | Bleu ciel |

### 4. Effets Visuels

- **Overlays de flip** : Effets d'ombre pour chaque direction
- **Indicateurs de direction** : Flèches pour visualiser le sens de flip
- **Icônes de comportement** : Pastilles colorées pour les états des créatures

## 🎮 Utilisation dans le Code

```go
// Obtenir le manager d'assets
manager := assets.NewManager()

// Tuile avec thème spécifique (état de l'entité sur la tuile)
// Les états "hidden", "revealed", "matched" correspondent à TileState de l'entité
tileImg := manager.GetTileImage(entity.Hidden, "forest")

// Icône de ressource
berryImg := manager.GetResourceIcon("dreamberry")

// Icône de créature
flyImg := manager.GetCreatureIcon("lumifly")
```

Note : Les états visuels (Hidden, Revealed, Matched) à partir de la fusion #18 sont gérés par les entités, pas par les tuiles. Le manager d'assets utilise les états des entités pour afficher le bon visuel.

## 🎨 Personnalisation

Pour créer un nouveau thème de tuiles :

```go
var MonTheme = assets.TileTheme{
    HiddenBg:        color.RGBA{45, 40, 65, 255},
    HiddenPattern:   color.RGBA{65, 60, 90, 255},
    HiddenBorder:    color.RGBA{100, 95, 130, 255},
    RevealedBg:      color.RGBA{55, 55, 75, 255},
    RevealedPattern: color.RGBA{75, 75, 95, 255},
    MatchedBg:       color.RGBA{40, 90, 50, 255},
    MatchedPattern:  color.RGBA{60, 130, 70, 255},
}
```

## 📄 Licence

```
Tous les assets générés par ce code sont dans le domaine public.

Vous êtes libres de :
- ✅ Utiliser les assets à des fins commerciales ou personnelles
- ✅ Modifier les assets comme vous le souhaitez
- ✅ Distribuer les assets sans attribution
- ✅ Créer des œuvres dérivées

Aucune restriction ne s'applique.
```

## 🔧 Fichiers Source

Les assets sont définis dans :
- `internal/infrastructure/assets/manager.go` - Gestionnaire principal
- `internal/infrastructure/assets/tiles.go` - Génération des tuiles
- `internal/infrastructure/assets/resources.go` - Icônes de ressources
- `internal/infrastructure/assets/creatures.go` - Icônes de créatures

## 🎵 Assets Audio Existant

Le dossier `assets/` contient également des fichiers audio :
- `musics/music.ogg` - Musique d'ambiance
- `sfx/un_bruit.wav` - Effet sonore

Ces fichiers doivent être remplacés par des assets libres de droit (voir ressources ci-dessous).

### Ressources pour Assets Audio Libres

- [OpenGameArt.org](https://opengameart.org/) - Musique et SFX CC0
- [Freesound.org](https://freesound.org/) - Effets sonores (vérifier les licences)
- [Incompetech](https://incompetech.com/music/royalty-free/music.html) - Musique de Kevin MacLeod
- [FreePD](https://freepd.com/) - Musique domaine public
