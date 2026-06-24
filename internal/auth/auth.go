// Package auth stores and resolves provider credentials. Resolution order (contract §7):
//
//	env var  →  OS keyring  →  0600 XDG file fallback
//
// Secrets are read from stdin/env, never argv. The keyring is restricted to OS-NATIVE
// backends (Keychain / Secret Service / WinCred) — never the passphrase-prompting file
// backend, which would deadlock a headless agent. When no OS backend exists we fall back
// to our own 0600 JSON file (and warn on loose perms), not an interactive keyring.
package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/99designs/keyring"
)

// Credential field names.
const (
	FieldAPIKey       = "api_key"       // third-party providers (SerpApi, Rainforest)
	FieldClientID     = "client_id"     // official Creators OAuth
	FieldClientSecret = "client_secret" // official Creators OAuth
)

const serviceName = "rivr"

// EnvVar returns the environment variable that overrides a stored credential, e.g.
// provider "serpapi" + FieldAPIKey → "RIVR_SERPAPI_API_KEY".
func EnvVar(provider, field string) string {
	up := func(s string) string { return strings.ToUpper(strings.ReplaceAll(s, "-", "_")) }
	return "RIVR_" + up(provider) + "_" + up(field)
}

func ringKey(provider, field string) string { return provider + "/" + field }

// openRing opens the OS-native keyring, or returns ok=false to trigger the file fallback.
func openRing() (keyring.Keyring, bool) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName: serviceName,
		AllowedBackends: []keyring.BackendType{
			keyring.KeychainBackend,
			keyring.SecretServiceBackend,
			keyring.WinCredBackend,
			keyring.KWalletBackend,
		},
	})
	if err != nil {
		return nil, false
	}
	return ring, true
}

// Get resolves a credential: env → keyring → file. Returns "" (no error) when unset.
func Get(provider, field string) (string, error) {
	if v := os.Getenv(EnvVar(provider, field)); v != "" {
		return strings.TrimSpace(v), nil
	}
	if ring, ok := openRing(); ok {
		if item, err := ring.Get(ringKey(provider, field)); err == nil {
			return string(item.Data), nil
		}
		// fall through to file fallback on miss / backend error
	}
	return fileGet(provider, field)
}

// Set persists a credential to the keyring, or the 0600 file fallback.
func Set(provider, field, value string) error {
	if ring, ok := openRing(); ok {
		err := ring.Set(keyring.Item{Key: ringKey(provider, field), Data: []byte(value)})
		if err == nil {
			return nil
		}
	}
	return fileSet(provider, field, value)
}

// Delete removes ALL local credentials for a provider (keyring + file). Local only.
func Delete(provider string) error {
	if ring, ok := openRing(); ok {
		for _, f := range []string{FieldAPIKey, FieldClientID, FieldClientSecret} {
			_ = ring.Remove(ringKey(provider, f))
		}
	}
	return fileDelete(provider)
}

// Backend reports which store is active, for doctor/status display.
func Backend() string {
	if _, ok := openRing(); ok {
		return "os-keyring"
	}
	return "file (0600 fallback)"
}

// --- 0600 file fallback ------------------------------------------------------

func fileDir() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "rivr")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "rivr")
}

func filePath() string { return filepath.Join(fileDir(), "credentials.json") }

func fileLoad() (map[string]string, error) {
	m := map[string]string{}
	b, err := os.ReadFile(filePath())
	if os.IsNotExist(err) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(b, &m)
	return m, nil
}

func fileGet(provider, field string) (string, error) {
	m, err := fileLoad()
	if err != nil {
		return "", err
	}
	return m[ringKey(provider, field)], nil
}

func fileSet(provider, field, value string) error {
	m, err := fileLoad()
	if err != nil {
		return err
	}
	m[ringKey(provider, field)] = value
	if err := os.MkdirAll(fileDir(), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath(), b, 0o600)
}

func fileDelete(provider string) error {
	m, err := fileLoad()
	if err != nil {
		return err
	}
	changed := false
	for k := range m {
		if strings.HasPrefix(k, provider+"/") {
			delete(m, k)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(filePath(), b, 0o600)
}

// InsecureFilePerms reports whether the fallback credentials file has perms looser than
// 0600 (group/other bits set), for doctor to warn on.
func InsecureFilePerms() (bool, string) {
	fi, err := os.Stat(filePath())
	if err != nil {
		return false, ""
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return true, fmt.Sprintf("%s is %o; tighten with: chmod 600 %s", filePath(), fi.Mode().Perm(), filePath())
	}
	return false, ""
}
