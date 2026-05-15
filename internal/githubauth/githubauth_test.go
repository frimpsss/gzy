package githubauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDeviceFlowSuccessAndKeyUpload(t *testing.T) {
	var uploadedKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/device/code":
			_ = json.NewEncoder(w).Encode(DeviceCode{
				DeviceCode:      "device",
				UserCode:        "ABCD-1234",
				VerificationURI: "https://github.com/login/device",
				ExpiresIn:       900,
				Interval:        1,
			})
		case "/login/oauth/access_token":
			_ = json.NewEncoder(w).Encode(TokenResponse{AccessToken: "token", TokenType: "bearer", Scope: "write:public_key"})
		case "/user/keys":
			if got := r.Header.Get("Authorization"); got != "Bearer token" {
				t.Fatalf("Authorization = %q", got)
			}
			var body CreateKeyRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			uploadedKey = body.Key
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(KeyResponse{ID: 42, Key: body.Key, Title: body.Title})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := Client{
		ClientID: "client",
		BaseURL:  server.URL,
		HTTP:     server.Client(),
		Sleep:    func(time.Duration) {},
	}
	code, err := client.StartDeviceFlow()
	if err != nil {
		t.Fatalf("StartDeviceFlow() error = %v", err)
	}
	token, err := client.WaitForToken(code, time.Now().Add(time.Second))
	if err != nil {
		t.Fatalf("WaitForToken() error = %v", err)
	}
	key, err := client.UploadKey(token.AccessToken, "gzy-p-host", "ssh-ed25519 AAAATEST")
	if err != nil {
		t.Fatalf("UploadKey() error = %v", err)
	}
	if key.ID != 42 || uploadedKey != "ssh-ed25519 AAAATEST" {
		t.Fatalf("key=%#v uploaded=%q", key, uploadedKey)
	}
}

func TestWaitForTokenDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login/device/code" {
			_ = json.NewEncoder(w).Encode(DeviceCode{DeviceCode: "device", UserCode: "CODE", VerificationURI: "https://github.com/login/device", ExpiresIn: 900, Interval: 1})
			return
		}
		_ = json.NewEncoder(w).Encode(TokenResponse{Error: "access_denied"})
	}))
	defer server.Close()
	client := Client{ClientID: "client", BaseURL: server.URL, HTTP: server.Client(), Sleep: func(time.Duration) {}}
	code, err := client.StartDeviceFlow()
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.WaitForToken(code, time.Now().Add(time.Second))
	if err == nil {
		t.Fatal("WaitForToken denied returned nil")
	}
}

func TestDeleteKeySuccessAndIdempotent(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/user/keys/42":
			w.WriteHeader(http.StatusNoContent)
		case "/user/keys/99":
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := Client{BaseURL: server.URL, HTTP: server.Client()}
	if err := client.DeleteKey("token", 42); err != nil {
		t.Fatalf("DeleteKey(42) error = %v", err)
	}
	if err := client.DeleteKey("token", 99); err != nil {
		t.Fatalf("DeleteKey(99) on missing key should be nil, got %v", err)
	}
	if len(calls) != 2 || calls[0] != "DELETE /user/keys/42" || calls[1] != "DELETE /user/keys/99" {
		t.Fatalf("calls = %v", calls)
	}
}
