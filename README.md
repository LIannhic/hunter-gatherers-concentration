# Hunter Gatherers Concentration

> Ce puzzle-aventure réinvente le "Memory" en outil de survie stratégique. Associez des tuiles pour capturer des entités, récolter des ressources et créer des artefacts. Un défi cognitif au tour par tour où chaque erreur de mémoire fragilise votre progression.

---

## Concept

Le projet propose une expérience de puzzle-aventure en tour par tour, où la mécanique du "Memory" est réinventée pour devenir un outil d'exploration et de survie. Dans ce jeu, l'association de paires identiques permet d'interagir avec un environnement complexe. Le joueur doit faire preuve de mémoire pour capturer des entités, récolter des composants et manufacturer des artefacts. En transformant un exercice cognitif en une boucle de gameplay profonde, le jeu invite à une réflexion calme mais tendue, où chaque erreur peut altérer la stabilité de l'expérience.

## Contexte

L’univers du jeu prend racine dans un monde marqué par la coexistence de la réalité matérielle et de plans de réalités alternatifs, des dimensions oniriques et ésotériques où les souvenirs et les symboles acquièrent une substance physique. Dans ces contrées instables, les ressources ne sont pas inertes : elles naissent, mûrissent et réagissent selon des cycles mystérieux que seuls les initiés savent décrypter. Le récit suit le parcours d'un jeune protagoniste soudainement propulsé au rang de "récolteur" suite à une rupture brutale de son apprentissage. Sans avoir achevé sa formation, il hérite de la lourde responsabilité de subvenir aux besoins d'un foyer fragilisé par l'absence et les dettes.
L’enjeu narratif réside dans la tension entre la grisaille du monde réel et l'étrangeté fascinante des plans parcourus. Chaque incursion est une nécessité vitale : le héros doit apprendre à déchiffrer les indices d'un environnement qui cherche autant à se dissimuler qu'à l'entraver. La survie ne dépend pas seulement de sa force, mais de sa capacité à comprendre la faune éthérée qui marque son territoire et transforme le décor. Le contexte impose ainsi un poids émotionnel à chaque tour de jeu : le joueur n'explore pas par simple curiosité, mais pour allouer les gains de ses découvertes à une famille dont l'avenir dépend de sa maîtrise croissante des lois invisibles de ces dimensions.

## Fonctionnalités principales et Contenus

Le projet se définit par un système de jeu hybride alliant le puzzle mental, la survie et la gestion. Le contenu principal repose sur une mécanique d'association de paires identiques : le joueur mémorise des visuels pour récolter des ressources ou capturer des entités. Ce système est soutenu par un écosystème dynamique où les ressources ne sont pas de simples objets inertes, mais des entités vivantes capables de maturer, de se propager ou de réagir aux interactions environnementales. Le plateau de jeu devient ainsi un monde en constante mutation, influencé par des structures interactives telles que des terriers ou des zones de dissimulation qui modifient la visibilité et la sécurité du plan.
L'expérience propose une profondeur stratégique s'appuyant sur une double boucle de progression. En mission, le joueur doit composer avec des créatures aux comportements riches, capables de transformer le terrain ou de laisser des traces persistantes de leur passage, offrant un défi de lecture environnementale permanent. Hors mission, le contenu se déplace vers une gestion de méta-progression exigeante : le foyer familial devient un enjeu central où chaque ressource collectée doit être judicieusement allouée pour assurer la survie du groupe et l'évolution des capacités du protagoniste. Cette richesse de contenu vise à récompenser l'expertise du joueur, transformant chaque incursion en une opportunité de déchiffrer les secrets d'un univers où l'indice visuel et le cycle de vie des éléments sont aussi cruciaux que la mémoire pure.

## Mécanisme

Le gameplay repose sur une gestion rigoureuse des ressources et du temps, dictée par un système en tour par tour. Chaque interaction, qu’il s’agisse de retourner une tuile ou d’utiliser un objet, consomme une unité de ressource (temps ou mana), forçant le joueur à planifier ses mouvements au sein d’une limite de tours impartis. La survie dépend de la gestion d'une barre de santé, physique ou mentale, qui s’érode au fil des erreurs ou des confrontations.
L’aspect central de l’association de paires identiques enrichit la mécanique de mémoire classique. La difficulté est modulable via des variables structurelles, comme l'éparpillement des paires ou la visibilité de l'inventaire. Enfin, l'environnement est rendu vivant et menaçant par la présence de créatures aux comportements déterminés : celles-ci occupent des placements précis et effectuent des déplacements prévisibles mais contraignants, obligeant le joueur à adapter sa stratégie de mémorisation en fonction de leurs mouvements sur le plateau.

### Équilibre de la Population

Chaque zone (grille) respecte désormais une répartition équilibrée et déterministe de sa population :
- **40% de Ressources** : Objectifs principaux de récolte.
- **40% de Créatures** : Menaces et opportunités (incluant l'espèce exclusive du biome).
- **20% de Pièges** : Obstacles à la progression.

Cette répartition est calculée individuellement pour chaque grille lors de sa génération, garantissant un défi constant quelle que soit la zone explorée.

### Minimap et Exploration (Fog of War)

Le plan onirique est un réseau de zones interconnectées. Pour s'orienter sans briser le mystère, la minimap utilise un système de **brouillard de guerre dynamique** :

- **Rendu Centré** : La zone actuelle est toujours fixée au centre de la minimap (grille 9x9). Le reste du monde glisse autour lors des transitions.
- **Persistance de l'Entrée** : En entrant dans une nouvelle zone, la sortie par laquelle le joueur arrive est **automatiquement révélée et appairée**, symbolisant le chemin connu.
- **États de Découverte** :
  - **Inconnu** : Zone cachée.
  - **Adjacent** : Silhouette grise (route possible identifiée). Les sorties découvertes restent affichées en permanence.
  - **Visité** : Affiche l'icône du biome (Forest, Cave, Swamp, etc.).
  - **Actuel** : Indicateur doré pulsant.

### Matrice de Dégâts (MATCH vs SKIP)

Le jeu punit l'inattention et la précipitation via une matrice de décision stricte lors de l'interaction avec des paires de créatures ou de ressources :

| État de la paire | Action: **MATCH** | Action: **SKIP** |
| :--- | :--- | :--- |
| **VALIDE** | 0 Dégât (Succès) | Pénalité (Erreur) |
| **INVALIDE** | Pénalité (Échec) | 0 Dégât (Prudence) |

- **Dégâts (Créatures)** : `nombre de créatures révélées * 10` points de vie.
- **Mana (Ressources)** : `nombre de ressources révélées * 5` points de mana.
- **Match Valide** : Les entités sont capturées/récoltées (Loot).
- **Match Invalide** : Les tuiles se referment violemment.
- **Skip** : Permet de refermer les tuiles sagement si aucune paire n'est identifiée.

### Confrontation et Animations d'Attaque

Chaque créature possède une **Zone de Menace** (directions qu'elle attaque).
- **Placement Périphérique** : Le joueur agit depuis le bord des cases. Sa position (`entity.Position`) et son ancrage (`BorderPosition`) sont déterminés par l'endroit où il clique pour révéler une tuile.
- **Animation de Lunge** : Lors du dévoilement (Hidden -> Revealed), la créature effectue une translation brusque de quelques pixels vers sa zone de menace, suivie d'un retour lent à sa position initiale.
- **Indicateurs de Menace** : Durant cette animation, des demi-cercles blancs (Intention de Menace) apparaissent entre la créature et les cases qu'elle menace. Ces indicateurs s'affichent même si la cible est en dehors de la grille (sur le tapis de jeu).
- **Calcul de Confrontation** : Si le joueur se trouve dans la zone de menace lors du dévoilement, il subit **10 points de dégâts physiques**.
- **Visualisation des Dégâts** : Une attaque réussie remplace l'indicateur blanc par un demi-cercle rouge intense, synchronisé avec le mouvement brusque de la créature.
- **Esquive** : En choisissant de "tirer" la tuile depuis un angle mort de la créature, le joueur peut éviter l'attaque lors de la révélation.

### Compte à rebours temps réel (Turn Timer)

Pour rompre le rythme classique du Memory et simuler l'urgence de la survie en plan onirique, chaque tour est soumis à une **pression temporelle dynamique** :

- **Durée** : dépend de la difficulté (10s en Easy, 8s en Normal, 5s en Hard, 4s en Insane).
- **Reset** : le timer se réinitialise à chaque action volontaire (retourner une tuile, Match, Skip, Fin de tour).
- **Auto-skip** : si le timer atteint 0, le système simule automatiquement un **Skip** via l'événement `turn_timer_expired` — les tuiles retournées se referment et le tour est consommé, entraînant la pénalité de Santé Mentale associée.
- **Feedback visuel** :
  - Le bouton **Skip** se remplit progressivement d'une couleur violette, puis passe au rouge brique en phase d'alerte.
  - La jauge de **Santé Mentale** tremble de plus en plus fort lorsque le timer descend sous les 3 secondes (phase de panique).

### Système de Portail Portable

Le portail portable permet au joueur de s'extraire rapidement du plan actif.
- **Déploiement** : Nécessite une zone 3x3.
- **Effet Séisme** : Lors de l'activation, toutes les entités présentes sur les 8 parcelles entourant le portail sont immédiatement supprimées pour libérer l'espace.
- **Feedback Visuel** : Un shader de type **Vortex** crée un tourbillon de distorsion centré sur le portail pendant toute la durée de l'extraction.
- **Extraction** : Une fois déployé, un compte à rebours de 5 secondes se lance avant la victoire.

### Boutons d'action réactifs du Playmat

Quatre boutons fixes occupent les coins du Playmat. Leur accessibilité est purement réactive :

| Bouton | Position | Comportement |
|--------|----------|--------------|
| **MATCH** | Haut-gauche | Capture la paire. Donne du butin (Loot). |
| **SKIP** | Haut-droite | Actif uniquement quand 2 tuiles sont retournées. Referme les tuiles et consomme le tour. |
| **TURN** | Bas-gauche | Toujours actif. Force la fin du tour. Consomme 1 Sanité par défaut. |
| **MERGE** | Bas-droite | Fusionne la paire en une version plus forte (Niveau +1). Se recache. |

### Fusion et Cumul (MERGE vs MATCH)

Le système de progression immédiate sur le plateau repose sur le choix du joueur :
- **MERGE** : Fusionne deux tuiles identiques de même niveau.
    - **Coût** : `3 * (Niveau + 1)` Mana.
    - **Résultat** : Une tuile est absorbée, l'autre devient une version **Cumulée** supérieure (Niveau max 2).
    - **Conséquence** : Termine le tour et referme toutes les tuiles.
- **MATCH** : Capture une paire identique de même niveau.
    - **Coût** : 1 Mana.
    - **Résultat** : Les deux tuiles sont supprimées et transformées en **Butin** (Loot).
    - **Scaling** : Un match de haut niveau produit un butin de plus grande valeur.
- **Révélation** : Retourner une tuile cumulée (Niv.1+) consomme du Mana égal à son niveau.

### Feedback Visuel et Prudence (Hover)

L'interface assiste le joueur dans sa gestion des ressources via un système de **feedback dynamique au survol** :
- **Jauges Clignotantes** : Survoler un bouton d'action (**Match**, **Merge**, **Skip**, **Turn**) fait clignoter un segment blanc sur les jauges (Mana, Santé, Sanité) correspondant au coût ou au risque maximum de l'action.
- **Anti-Triche** : Aucun feedback de coût n'est affiché au survol des tuiles cachées pour éviter de divulguer des informations sur leur nature (créature vs ressource).
- **Rendu Sélectif** : Les tuiles cumulées n'apparaissent plus grandes et dorées que lorsqu'elles sont **Révélées**. À l'état caché, elles sont identiques aux tuiles normales.

### Notifications Défilantes (HUD)

Pour maintenir l'immersion tout en informant le joueur, le HUD intègre deux zones de messages dynamiques :
- **Zone Gauche (Portrait/Inventaire)** : Confirme l'activation des capacités d'objets (ex: "Vous vous sentez évanescent").
- **Zone Droite (Jauges/Minimap)** : Signale les événements de combat et les erreurs (ex: "CONFRONTATION ! -10 HP", "MATCH INVALIDE").
- **Comportement** : Les messages utilisent une file d'attente pour éviter les chevauchements et défilent de droite à gauche deux fois avant de disparaître.

### Troubles cognitifs (Status Effects)

Le joueur peut subir des altérations mentales qui déforment l'interface :

- **Aphasia** – brouille les labels des boutons (ex: "MATCH" devient "???" ou est permuté).
- **Agnosia** – rend les boutons visuellement indifférenciés (couleurs altérées).
- **Ataxia** – scramble les positions des boutons (ex: le bouton Match peut basculer au coin bas-droit).
- **Amnesia** – désactive aléatoirement des boutons (30% de chance d'"oublier" un bouton).

Ces effets sont interceptés par le système de rendu **avant** l'affichage, forçant le joueur à lutter contre sa propre interface.

### État Évanescent (Bonus Shadowstalker)

L'utilisation d'un butin de Shadowstalker permet de passer entre les plans :
- **Protection** : Immunité totale aux dégâts pendant 1 tour.
- **Feedback** : Le cadre de survol et de sélection sur la grille devient **gris pierre**.
- **Message** : "Vous vous sentez évanescent."

### Vision des Intentions (Bonus Fleeing Sprite)

L'utilisation d'un butin de Fleeing Sprite illumine les dangers :
- **Effet** : Rend visible les zones de menace de toutes les créatures de la grille pour 1 tour.
- **Feedback** : Affiche des arcs blancs entre les créatures et les cases qu'elles ciblent.
- **Message** : "L'éclat du Fleeing Sprite révèle les intentions de vos prédateurs."

## Verbes

Pour le joueur :Pendant les parties (Actions directes)

* Dévoiler : Révéler le contenu d’une tuile face cachée (consomme du temps ou du mana).
* Prendre : Collecter des éléments ou capturer des créatures simples via l’identification d’une paire identique.
* Associer : Créer un lien entre deux tuiles identiques pour synthétiser des composants ou capturer des entités.
* Utiliser : Activer un outil, une ressource consommable ou une capacité de créature (ex: cri de l'Echo Hound) pour modifier le plateau ou protéger ses statistiques. Nécessite une confirmation par un second clic sur l'objet sélectionné.
* Déployer portail : Activer un portail portable pour créer une zone de dégagement 3x3 et initier l’extraction du plan.
* S’extraire : Initier la fin de l’incursion pour sécuriser le butin avant l’épuisement des ressources de survie.

Entre les parties (Méta-progression)

* Commencer : Choisir une destination onirique et lancer une nouvelle session d'exploration.
* Échanger : Troc ou vente des ressources collectées contre des devises ou des objets rares.
* Allouer : Distribuer les gains au foyer pour répondre aux besoins de la famille (nourriture, dettes, santé).
* Apprendre : Investir dans des compétences pour débloquer de nouveaux types d'associations ou améliorer les résistances du héros.
* Équiper : Préparer son inventaire et ses outils en fonction des dangers prévus pour la prochaine mission.

Savoir du joueur (Apprentissage et Déduction)

* Identifier : Reconnaître le stade de maturation d'une ressource (du bourgeon au fruit gâté) pour optimiser le moment de la récolte.
* Déchiffrer : Interpréter les indices visuels dissimulés sur le recto/verso des tuiles ou dans l'arrière-plan pour deviner le contenu caché.
* Prédire : Anticiper le comportement d'une créature ou l'évolution d'un cycle environnemental en fonction de l'expérience acquise.
* Distinguer : Différencier une opportunité d'un danger (piège) grâce à l'observation de détails subtils sur le plateau de jeu.

Pour les créatures :

* Se placer : se positionner au départ.
* Se déplacer : changer position au cours de la partie.
* Fuir: quitter la zone de jeu.
* Attaquer: infliger des dégâts au joueur.
* Altérer : modifie le statut du joueur, transforme l'environnement.
* Transformer : Agir sur les ressources pour les faire évoluer, les dégrader ou modifier leur accessibilité (ex: polliniser, briser, fertiliser).
* Marquer : Laisser des traces persistantes de son passage sur le plateau (empreintes, griffures), offrant au joueur des indices sur sa position ou son trajet.

### Système de Déplacement Avancé

Les créatures disposent d'un système de mouvement configurable avec les paramètres suivants :

#### Déclencheurs (quand se déplacer)
- **Passif** : Aucun mouvement
- **Auto** : À la fin de chaque tour
- **Vue** : Dès que révélée
- **Echo** : Quand une autre tuile est révélée
- **Proximité** : Si action dans un rayon de N cases

#### Navigation (où aller)
- **Errance** : Direction aléatoire
- **Patrouille** : Suit un itinéraire défini
- **Orientation** : Selon la direction du regard
- **Attraction** : Vise une cible spécifique
- **Répulsion** : S'éloigne d'une cible

#### Modes de déplacement
- **Bento** : Déplacement visible (le joueur voit le mouvement)
- **Shadow** : Déplacement invisible (face cachée, le joueur doit deviner)
- **Swap** : Échange de place avec la cible
- **Over** : Calque supérieur (vol, effets de surface)
- **Under** : Calque inférieur (fouissage, traces profondes)

### Traces et Indices Environnementaux

Les créatures laissent derrière elles des traces qui respectent des règles de placement précises sur les calques de rendu :

- **Boue (Mud)** : Rendu sur le calque **Under**, positionné exactement entre deux cases (interstice) et orienté selon la direction du mouvement.
- **Herbe Brisée (Broken Grass)** : Rendu sur le calque **Under**, marquant la case d'origine du déplacement.
- **Griffures (Claws)** : Rendu sur le calque **Over**, marquant la case de destination (impact).
- **Empreintes (Footprints)** : Rendu sur le calque **Normal**, apparaissant sous les tuiles physiques pour simuler le passage au sol.
- **Rayons d'Attaque (Intent Beam)** : Rendu sur le calque **Over**, positionné entre la créature et sa cible. Rouge pour une attaque réussie, Blanc pour une simple zone de menace.

Toutes les traces s'adaptent dynamiquement à la distance entre les cases, qu'il s'agisse d'un plateau 3x3 ou 6x6.

### Bestiaire

| Créature | Biome | Déclencheur | Navigation | Mode | Collision | Description |
|----------|-------|-------------|------------|------|-----------|-------------|
| **Lumifly** | Global | Auto | Errance (nord) | Over | Glisse | Insecte lumineux qui vole au-dessus du plateau |
| **Shadowstalker** | Global | Proximité (4) | Attraction joueur | Shadow | Rebond | Prédateur qui chasse discrètement le joueur |
| **Burrower** | Désert | Vue | Errance | Under | Phase (terre) | Créature fouisseuse qui se cache sous terre (Exclusif) |
| **Specter** | Grotte | Echo | Errance | Shadow | Phase (murs) | Fantôme qui traverse les murs (Exclusif) |
| **Stonewarden** | Global | Passif | Patrouille | Bento | Stop | Gardien immobile qui patrouille si révélé |
| **Echo Hound** | Marais | Echo | Attraction curseur | Bento | Glisse | Chien rapide qui réagit aux révélations (Exclusif). |
| **Moss Monkey** | Forêt | Proximité (4) | Target Empty | Bento | Glisse | Saboteur qui rebouche les cases vides avec des leurres (Exclusif). Fuit si saturé. |
| **Flutterwing** | Global | Proximité (2) | Répulsion joueur | Over | Glisse | Créature timide dont l'essence apaise l'esprit. |
| **Fleeing Sprite** | Global | Proximité (2) | Répulsion joueur | Over | Glisse | Étincelle d'énergie vive révélant les dangers. |

### Botanique et Minéralogie

| Ressource | Biome | Effet du Butin | Description |
|-----------|-------|----------------|-------------|
| **Dreamberry** | Global | +5 Mana | Baie onirique violette, base de l'alchimie. |
| **Moonstone** | Global | +5 Sanité | Pierre de lune bleutée, stabilise l'esprit. |
| **Whispering Herb** | Global | Lore | Herbe murmurante, révèle des secrets. |
| **Crystal Shard** | Global | +5 Santé | Éclat de cristal, régénère les tissus. |
| **Moss Truffle** | Forêt | +15 Santé | Truffe rare poussant sous la mousse (Exclusif). |
| **Void Bloom** | Grotte | +15 Sanité | Fleur indigo défiant les lois de la physique (Exclusif). |
| **Echo Crystal** | Marais | +15 Mana | Cristal résonnant avec les courants éthérés (Exclusif). |
| **Sand Core** | Désert | +5 All Stats | Noyau d'énergie pure extrait du sable (Exclusif). |

Pour les ressources :

* Mûrir / Maturer : Évoluer d'un stade initial (ex: bourgeon) vers un stade optimal (ex: fruit mûr) puis vers un stade dégradé (ex: fruit gâté), changeant ainsi sa valeur et ses propriétés d'association.
* Se propager : S'étendre aux tuiles adjacentes (uniquement directions cardinales) si certaines conditions sont remplies. Une faible probabilité (5%) de stérilité peut freiner cette expansion.
* Réagir : Modifier son état au contact d'un autre élément ou d'une créature (ex: une ressource qui "éclot" si une créature pollinisatrice passe dessus).
* Se dégrader : Perdre en qualité ou disparaître si elle n'est pas récoltée à temps ou si le plan devient instable.
* Rayonner : Laisser filtrer des indices visuels sur l'arrière-plan ou le verso de la tuile (ex: une ressource luminescente qui "brille" légèrement à travers la carte face cachée).
* S'altérer : Changer de nature sous l'influence d'un événement climatique ou psychique (ex: un minerai qui devient instable ou explosif).


Verbes de l'Environnement (Décors et Structures)

* Dissimuler : Masquer la présence d'une créature ou d'une ressource (ex: de hautes herbes ou un brouillard psychique sur la tuile).
* Abriter / Révéler : Servir de refuge à une entité (ex: le terrier). Si le refuge est dévoilé en même temps que l'entité, celle-ci est débusquée.
* Déclencher (Trigger) : Activer un événement si une condition est remplie (ex: si le terrier ET la créature sont visibles, la créature fuit).
* Entraver : Restreindre les mouvements des créatures ou l'accès du joueur à certaines tuiles (ex: ronces, rochers).

### Note aux collaborateurs

Ce document pose des piliers mécaniques et narratifs. Tout le reste, l'esthétique précise, le bestiaire, les collectables, les ressources, l’écosystème, l’environnement, la structure des plans parallèles, l’histoire, les personnages, les lieux, etc...,  est un espace ouvert à vos idées et à votre expertise.

---

## Design et travaux préparatoires

[Consulter le tableau blanc sur Figma.](https://www.figma.com/design/Pzh46PCN6sdWgwqBhf8bvn/Jeux-Vid%C3%A9o?node-id=325-2&t=69lWglExoEjbC8O1-1)

---

## Installion & lancement

### Prérequis

* go

### Récupérer le projet

* cloner le dépôt
* Installer les dépendances

```bash
go mod download
```

### Lancer le projet

```bash
# Mode normal
go run ./cmd/game

# Mode debug (avec logs détaillés)
go run ./cmd/game -debug

# Build production
go build -o game ./cmd/game
./game
```

### Contrôles

#### Jeu de base (Actions directes)

| Action | Touche |
|--------|--------|
| Révéler tuile | Click gauche (Plateau) |
| Sélectionner tuile révélée | Click gauche (Tuile révélée) |
| Désélectionner / Annuler | Click droit (Plateau) ou Échap |
| Matcher (valider paire) | M ou Bouton MATCH |
| Skip (si 2 tuiles révélées) | Espace ou Bouton SKIP |
| Fin de tour forcée | Espace (sans match en cours) ou Bouton TURN |
| Naviguer entre les zones | Flèches ou ZQSD / WASD |
| Rotation plateau (Visuel) | + (Horaire) / - (Anti-horaire) |
| Reset rotation | R |

#### Gestion et Inventaire

| Action | Touche |
|--------|--------|
| Sélection Butin (Usage) | Click gauche (Inventaire) |
| Utiliser Butin sélectionné | Re-click gauche (Inventaire) |
| Sélection Suppression | Click droit (Inventaire) |
| Désélectionner Butin | Click droit (Hors inventaire) |
| Portail Portatif (Raccourci) | P |
| Statistiques des zones | I |
| Détails Inventaire | L |
| Atlas des Assets (Debug) | T |

#### Paramètres et Debug

| Action | Touche |
|--------|--------|
| Difficulté (E, N, H, I) | F1 à F4 |
| Fenêtre de Debug (Console) | F12 |
| Spawn entités (Debug ouvert) | S |
| Spawn toutes créatures | Shift + S |
| Spawn créature aléatoire | F9 |
| Nettoyer plateau (Cheat) | C |
| Révéler tout (Cheat) | F5 |
| Cacher tout (Cheat) | F6 |
| Débloquer Navigation (Cheat) | F7 |
| Retirer état Bloqué (Cheat) | F8 |
| Changer de grille active | 1 à 9 |
| Retour menu / Abandon | \ ou Échap |
| Remplir Inventaire (Debug) | B |
| Toggle Mouvement Auto | F10 |
Annulation de la PR #39
