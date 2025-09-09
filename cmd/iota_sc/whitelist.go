package iota_sc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newWhitelistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "whitelist",
		Short:        "Whitelist utilities",
		SilenceUsage: true,
	}
	// Flag persistente: --id (puoi anche usare l'env DCS_WHITELIST_ID)
	cmd.PersistentFlags().String("id", "", "Whitelist object ID (0x...)")
	_ = viper.BindPFlag("dcs.whitelist_id", cmd.PersistentFlags().Lookup("id"))

	cmd.AddCommand(newWhitelistHasCmd())
	return cmd
}

func newWhitelistHasCmd() *cobra.Command {
	var member string
	var printAddr bool

	c := &cobra.Command{
		Use:   "has [ADDRESS]",
		Short: "Return true if ADDRESS/ID is in the whitelist",
		Args:  cobra.ArbitraryArgs, // gestiamo noi l'arg posizionale
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1) Risolvi l'ID whitelist: flag -> viper -> env
			id := viper.GetString("dcs.whitelist_id")
			if id == "" {
				id = os.Getenv("DCS_WHITELIST_ID")
			}
			if id == "" {
				return fmt.Errorf("set DCS_WHITELIST_ID env var or pass --id 0x...")
			}

			// 2) Risolvi l'indirizzo: flag --member prioritario, altrimenti argomento
			if mFlag, _ := cmd.Flags().GetString("member"); mFlag != "" {
				member = mFlag
			} else if len(args) > 0 {
				member = args[0]
			}
			if member == "" {
				return fmt.Errorf("provide address as positional arg or --member 0x")
			}
			member = strings.ToLower(member)

			// 3) Chiama l'IOTA CLI e prendi SOLO stdout in JSON
			out, err := exec.Command("iota", "client", "object", id, "--json").Output()
			// Se il comando fallisce ma ha comunque prodotto stdout, proviamo a parse-arlo.
			if err != nil && len(out) == 0 {
				return fmt.Errorf("iota client object failed: %w", err)
			}

			// 4) Se c'è qualsiasi prefisso non-JSON, tieni solo da '{'/'[' in poi
			if i := bytes.IndexAny(out, "{["); i >= 0 {
				out = out[i:]
			}

			// 5) Estrai fields.whitelist dal JSON (forma corrente o legacy)
			fields, err := extractFieldsMap(out)
			if err != nil {
				return err
			}
			rawWL, ok := fields["whitelist"]
			if !ok {
				// Nessuna whitelist presente → false (o niente se --print-addr)
				if printAddr {
					return nil
				}
				fmt.Println("false")
				return nil
			}

			// 6) Verifica presenza e stampa output minimale
			found := containsWhitelist(rawWL, member)
			if printAddr {
				if found {
					fmt.Println(member)
				}
				return nil
			}
			if found {
				fmt.Println("true")
			} else {
				fmt.Println("false")
			}
			return nil
		},
	}

	c.Flags().StringVarP(&member, "member", "m", "", "Address/ID to check (0x...)")
	c.Flags().BoolVar(&printAddr, "print-addr", false, "Print the address if present (instead of true/false)")
	return c
}

// Estrae la mappa fields dalle due possibili forme del JSON della CLI
// 1) root.content.fields (forma attuale)
// 2) root.data.content.fields (forma legacy)
func extractFieldsMap(jsonBytes []byte) (map[string]any, error) {
	var root map[string]any
	if err := json.Unmarshal(jsonBytes, &root); err != nil {
		return nil, fmt.Errorf("decode object json: %w", err)
	}
	if content, ok := root["content"].(map[string]any); ok {
		if fields, ok := content["fields"].(map[string]any); ok {
			return fields, nil
		}
	}
	if data, ok := root["data"].(map[string]any); ok {
		if content, ok := data["content"].(map[string]any); ok {
			if fields, ok := content["fields"].(map[string]any); ok {
				return fields, nil
			}
		}
	}
	return nil, fmt.Errorf("fields not found in object JSON")
}

// Cerca needle dentro al vettore Move fields.whitelist (stringhe o mappe {"id":..., "bytes":...})
func containsWhitelist(raw any, needle string) bool {
	switch arr := raw.(type) {
	case []any:
		for _, item := range arr {
			switch t := item.(type) {
			case string:
				if strings.EqualFold(t, needle) {
					return true
				}
			case map[string]any:
				if s, _ := t["id"].(string); s != "" && strings.EqualFold(s, needle) {
					return true
				}
				if s, _ := t["bytes"].(string); s != "" && strings.EqualFold(s, needle) {
					return true
				}
			}
		}
	}
	return false
}
