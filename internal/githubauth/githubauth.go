package githubauth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	ClientID string
	BaseURL  string
	HTTP     *http.Client
	Sleep    func(time.Duration)
	Scopes   []string
}

type DeviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
}

type CreateKeyRequest struct {
	Title string `json:"title"`
	Key   string `json:"key"`
}

type KeyResponse struct {
	ID    int64  `json:"id"`
	Key   string `json:"key"`
	Title string `json:"title"`
}

func (c Client) StartDeviceFlow() (DeviceCode, error) {
	if c.ClientID == "" {
		return DeviceCode{}, errors.New("GitHub OAuth client id is not configured")
	}
	scopes := c.Scopes
	if len(scopes) == 0 {
		scopes = []string{"write:public_key"}
	}
	form := "client_id=" + c.ClientID + "&scope=" + strings.Join(scopes, "+")
	req, err := http.NewRequest(http.MethodPost, c.endpoint("/login/device/code"), strings.NewReader(form))
	if err != nil {
		return DeviceCode{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var code DeviceCode
	if err := c.doJSON(req, &code); err != nil {
		return DeviceCode{}, err
	}
	return code, nil
}

func (c Client) WaitForToken(code DeviceCode, deadline time.Time) (TokenResponse, error) {
	interval := time.Duration(code.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	for time.Now().Before(deadline) {
		form := "client_id=" + c.ClientID + "&device_code=" + code.DeviceCode + "&grant_type=urn:ietf:params:oauth:grant-type:device_code"
		req, err := http.NewRequest(http.MethodPost, c.endpoint("/login/oauth/access_token"), strings.NewReader(form))
		if err != nil {
			return TokenResponse{}, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		var token TokenResponse
		if err := c.doJSON(req, &token); err != nil {
			return TokenResponse{}, err
		}
		switch token.Error {
		case "":
			if token.AccessToken == "" {
				return TokenResponse{}, errors.New("GitHub returned an empty access token")
			}
			return token, nil
		case "authorization_pending":
			c.sleep(interval)
		case "slow_down":
			interval += 5 * time.Second
			c.sleep(interval)
		case "expired_token", "access_denied":
			return TokenResponse{}, fmt.Errorf("GitHub authorization failed: %s", token.Error)
		default:
			return TokenResponse{}, fmt.Errorf("GitHub authorization failed: %s", token.Error)
		}
	}
	return TokenResponse{}, errors.New("GitHub authorization timed out")
}

func (c Client) UploadKey(token string, title string, publicKey string) (KeyResponse, error) {
	body, err := json.Marshal(CreateKeyRequest{Title: title, Key: publicKey})
	if err != nil {
		return KeyResponse{}, err
	}
	req, err := http.NewRequest(http.MethodPost, c.endpoint("/user/keys"), bytes.NewReader(body))
	if err != nil {
		return KeyResponse{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	var response KeyResponse
	uploadErr := c.doJSON(req, &response)
	if uploadErr == nil {
		return response, nil
	}
	var apiErr *APIError
	if errors.As(uploadErr, &apiErr) && apiErr.StatusCode == http.StatusUnprocessableEntity && strings.Contains(strings.ToLower(apiErr.Body), "already in use") {
		existing, lookupErr := c.findKeyByMaterial(token, publicKey)
		if lookupErr != nil {
			return KeyResponse{}, fmt.Errorf("GitHub reports the key is already in use, but listing your account keys failed: %w", lookupErr)
		}
		if existing.ID != 0 {
			return existing, nil
		}
		return KeyResponse{}, fmt.Errorf("GitHub reports the key is already in use but it is not on this account — it may be registered to a different GitHub user. Generate a new key (delete %s and re-run) or remove the duplicate from the other account", keyMaterial(publicKey))
	}
	return KeyResponse{}, uploadErr
}

func (c Client) findKeyByMaterial(token string, publicKey string) (KeyResponse, error) {
	req, err := http.NewRequest(http.MethodGet, c.endpoint("/user/keys"), nil)
	if err != nil {
		return KeyResponse{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	var keys []KeyResponse
	if err := c.doJSON(req, &keys); err != nil {
		return KeyResponse{}, err
	}
	target := keyMaterial(publicKey)
	for _, k := range keys {
		if keyMaterial(k.Key) == target {
			return k, nil
		}
	}
	return KeyResponse{}, nil
}

func keyMaterial(line string) string {
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		return parts[0] + " " + parts[1]
	}
	return strings.TrimSpace(line)
}

func (c Client) DeleteKey(token string, keyID int64) error {
	url := c.endpoint(fmt.Sprintf("/user/keys/%d", keyID))
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("GitHub key delete failed with status %s", resp.Status)
	}
	return nil
}

func (c Client) endpoint(path string) string {
	base := c.BaseURL
	if base == "" {
		base = "https://github.com"
	}
	if c.BaseURL == "" && strings.HasPrefix(path, "/user/keys") {
		return "https://api.github.com" + path
	}
	return strings.TrimRight(base, "/") + path
}

type APIError struct {
	Status     string
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	body := strings.TrimSpace(e.Body)
	if body == "" {
		return fmt.Sprintf("GitHub request failed with status %s", e.Status)
	}
	return fmt.Sprintf("GitHub request failed with status %s: %s", e.Status, body)
}

func (c Client) doJSON(req *http.Request, into any) error {
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{Status: resp.Status, StatusCode: resp.StatusCode, Body: string(body)}
	}
	if into == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(into)
}

func (c Client) sleep(duration time.Duration) {
	if c.Sleep != nil {
		c.Sleep(duration)
		return
	}
	time.Sleep(duration)
}
