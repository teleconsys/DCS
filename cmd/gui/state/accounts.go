package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// AccountFile is the on-disk representation written by `account new`.
// Newly-created GUI accounts include the optional `Role` field; CLI
// accounts (no role) still load fine because Role is "".
type AccountFile struct {
	Alias      string `json:"alias"`
	Address    string `json:"address"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
	Role       string `json:"role,omitempty"`
}

// ListAccounts scans ./accounts/*.json and returns one entry per file.
// Files that fail to parse are skipped silently.
func ListAccounts() []AccountFile {
	wd, err := os.Getwd()
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(filepath.Join(wd, "accounts"))
	if err != nil {
		return nil
	}
	out := make([]AccountFile, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		path := filepath.Join(wd, "accounts", e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var a AccountFile
		if err := json.Unmarshal(raw, &a); err != nil {
			continue
		}
		if strings.TrimSpace(a.Alias) == "" {
			a.Alias = strings.TrimSuffix(e.Name(), ".json")
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Alias < out[j].Alias })
	return out
}

// AccountsFor returns accounts tagged for the given actor first, followed
// by any untagged ("legacy") accounts. Useful for actor-aware pickers.
func AccountsFor(a Actor) []AccountFile {
	all := ListAccounts()
	role := a.RoleTag()
	matching := make([]AccountFile, 0, len(all))
	untagged := make([]AccountFile, 0, len(all))
	for _, acc := range all {
		switch strings.TrimSpace(strings.ToLower(acc.Role)) {
		case role:
			matching = append(matching, acc)
		case "":
			untagged = append(untagged, acc)
		}
	}
	return append(matching, untagged...)
}
