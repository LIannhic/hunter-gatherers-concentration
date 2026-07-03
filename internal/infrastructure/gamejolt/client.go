package gamejolt

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GameID et PrivateKey sont injectables via -ldflags au moment du build.
// Exemple: -ldflags="-X 'github.com/LIannhic/hunter-gatherers-concentration/internal/infrastructure/gamejolt.GameID=123'"
var (
	GameID     string
	PrivateKey string
)

const BaseURL = "https://api.gamejolt.com/api/game/v1/"

// Client gère les interactions avec l'API Game Jolt.
type Client struct {
	gameID     string
	privateKey string
	username   string
	userToken  string
	active     bool
	httpClient *http.Client
}

// NewClient crée une instance du client. Si username ou token sont vides,
// le client sera marqué comme inactif.
func NewClient(username, token string) *Client {
	c := &Client{
		gameID:     GameID,
		privateKey: PrivateKey,
		username:   username,
		userToken:  token,
		httpClient: &http.Client{},
	}

	if c.gameID != "" && c.privateKey != "" && c.username != "" && c.userToken != "" {
		c.active = true
	}

	return c
}

// IsActive retourne vrai si le client dispose de tous les identifiants nécessaires.
func (c *Client) IsActive() bool {
	return c.active
}

// signURL génère la signature MD5 requise par Game Jolt et l'ajoute à l'URL.
func (c *Client) signURL(rawURL string) string {
	u := rawURL + c.privateKey
	hash := md5.Sum([]byte(u))
	return rawURL + "&signature=" + hex.EncodeToString(hash[:])
}

// get effectue une requête GET signée vers l'API.
func (c *Client) get(endpoint string, params url.Values) ([]byte, error) {
	if !c.active {
		return nil, fmt.Errorf("GameJolt client is not active")
	}

	params.Set("game_id", c.gameID)
	params.Set("format", "json")

	fullURL := BaseURL + endpoint + "/?" + params.Encode()
	signedURL := c.signURL(fullURL)

	resp, err := c.httpClient.Get(signedURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Log simple pour le debug
	if strings.Contains(string(body), `"success":"false"`) {
		return body, fmt.Errorf("GameJolt API error: %s", string(body))
	}

	return body, nil
}

// --- Sessions ---

// SessionOpen ouvre une session pour l'utilisateur.
func (c *Client) SessionOpen() error {
	params := url.Values{}
	params.Set("username", c.username)
	params.Set("user_token", c.userToken)
	_, err := c.get("sessions/open", params)
	return err
}

// SessionPing maintient la session ouverte (doit être appelé toutes les 30s env).
func (c *Client) SessionPing(active bool) error {
	params := url.Values{}
	params.Set("username", c.username)
	params.Set("user_token", c.userToken)
	if active {
		params.Set("status", "active")
	} else {
		params.Set("status", "idle")
	}
	_, err := c.get("sessions/ping", params)
	return err
}

// SessionCheck vérifie si une session est déjà ouverte.
func (c *Client) SessionCheck() (bool, error) {
	params := url.Values{}
	params.Set("username", c.username)
	params.Set("user_token", c.userToken)
	_, err := c.get("sessions/check", params)
	if err != nil {
		return false, err
	}
	return true, nil
}

// SessionClose ferme la session actuelle.
func (c *Client) SessionClose() error {
	params := url.Values{}
	params.Set("username", c.username)
	params.Set("user_token", c.userToken)
	_, err := c.get("sessions/close", params)
	return err
}

// --- Scores ---

// ScoreAdd ajoute un score pour l'utilisateur.
func (c *Client) ScoreAdd(scoreStr string, sortValue int, tableID string) error {
	params := url.Values{}
	params.Set("username", c.username)
	params.Set("user_token", c.userToken)
	params.Set("score", scoreStr)
	params.Set("sort", fmt.Sprintf("%d", sortValue))
	if tableID != "" {
		params.Set("table_id", tableID)
	}
	_, err := c.get("scores/add", params)
	return err
}

// ScoreFetch récupère les scores d'une table.
func (c *Client) ScoreFetch(tableID string, limit int) ([]byte, error) {
	params := url.Values{}
	if tableID != "" {
		params.Set("table_id", tableID)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	return c.get("scores", params)
}

// ScoreTables liste les tables de score du jeu.
func (c *Client) ScoreTables() ([]byte, error) {
	return c.get("scores/tables", url.Values{})
}

// ScoreGetRank retourne le rang d'un score spécifique.
func (c *Client) ScoreGetRank(sortValue int, tableID string) (string, error) {
	params := url.Values{}
	params.Set("sort", fmt.Sprintf("%d", sortValue))
	if tableID != "" {
		params.Set("table_id", tableID)
	}
	data, err := c.get("scores/get-rank", params)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
