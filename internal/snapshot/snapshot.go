package snapshot

import (
	"encoding/json"
	"os"

	"github.com/Saba101/GoMetaSync/internal/models"
)

func SaveSnapshot(path string, snap *models.Snapshot) error {
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

func LoadSnapshot(path string) (*models.Snapshot, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snap models.Snapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}
