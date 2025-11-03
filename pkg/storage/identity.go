package storage

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/crypto"
)

type Identity struct {
	PrivKey []byte `json:"priv_key"`
}

func LoadOrCreateIdentity(dataDir string) (crypto.PrivKey, error) {
	keyPath := filepath.Join(dataDir, "identity.json")

	if _, err := os.Stat(keyPath); err == nil {
		// load existing
		data, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, err
		}
		var id Identity
		if err := json.Unmarshal(data, &id); err != nil {
			return nil, err
		}
		return crypto.UnmarshalPrivateKey(id.PrivKey)
	}

	// generate new one
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}

	// store it
	b, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	raw, _ := json.MarshalIndent(Identity{PrivKey: b}, "", "  ")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyPath, raw, 0600); err != nil {
		return nil, err
	}
	fmt.Printf("🔑 Generated new peer identity at %s\n", keyPath)
	return priv, nil
}
