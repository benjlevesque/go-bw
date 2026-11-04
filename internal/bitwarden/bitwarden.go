package bitwarden

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/benjlevesque/go-bw/internal/login"
)

type bitwardenDB struct {
	Folders []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"folders"`
	Items []struct {
		ID       string `json:"id"`
		FolderId string `json:"folderId"`
		Name     string `json:"name"`
		Login    struct {
			Username string `json:"username"`
			Password string `json:"password"`
			URIs     []struct {
				URI string `json:"uri"`
			} `json:"uris"`
			TOTP string `json:"totp"`
		} `json:"login"`
	} `json:"items"`
}

func ParseDatabaseJson(reader io.Reader) ([]login.LoginEntry, error) {
	jsonData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("Unable to read database %v", err)
	}
	db := bitwardenDB{}
	err = json.Unmarshal(jsonData, &db)
	if err != nil {
		return nil, fmt.Errorf("Unable to unmarshal data: %v", err)
	}
	folders := make(map[string]string)
	for _, folder := range db.Folders {
		folders[folder.ID] = folder.Name
	}

	entries := []login.LoginEntry{}
	for _, entry := range db.Items {
		uris := []string{}
		for _, uri := range entry.Login.URIs {
			uris = append(uris, uri.URI)
		}
		entries = append(entries, login.LoginEntry{
			ID:         entry.ID,
			Name:       entry.Name,
			Login:      entry.Login.Username,
			Password:   entry.Login.Password,
			URIs:       uris,
			Folder:     folders[entry.FolderId],
			TOTPSecret: entry.Login.TOTP,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Fullname() < entries[j].Fullname()
	})
	return entries, nil
}
