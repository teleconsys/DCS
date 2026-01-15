package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/teleconsys/DCS/internal/config"
	"github.com/teleconsys/DCS/internal/whitelist"
)

type whitelistReq struct {
	Member      string `json:"member"`
	WhitelistID string `json:"whitelistId,omitempty"`
}

func main() {
	// load env
	config.LoadEnv()

	r := gin.Default()

	// health
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// POST /v1/whitelist/add  (GC signs+submits)
	r.POST("/v1/whitelist/add", func(c *gin.Context) {
		if !requireAuth(c) {
			return
		}

		var req whitelistReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}

		member := strings.ToLower(strings.TrimSpace(req.Member))
		if member == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "member is required"})
			return
		}

		p, err := buildWhitelistAddParams(req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
		defer cancel()

		out, already, err := whitelist.AddToWhitelist(ctx, p)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"member":  member,
			"already": already,
			"tx":      jsonOrString(out),
		})
	})

	// POST /v1/whitelist/remove (GC signs+submits)
	r.POST("/v1/whitelist/remove", func(c *gin.Context) {
		if !requireAuth(c) {
			return
		}

		var req whitelistReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}

		member := strings.ToLower(strings.TrimSpace(req.Member))
		if member == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "member is required"})
			return
		}

		p, err := buildWhitelistRemoveParams(req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
		defer cancel()

		out, notPresent, err := whitelist.RemoveFromWhitelist(ctx, p)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"member":     member,
			"notPresent": notPresent,
			"tx":         jsonOrString(out),
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	_ = r.Run(":" + port)
}

// ---- helpers ----

func requireAuth(c *gin.Context) bool {
	want := strings.TrimSpace(os.Getenv("GC_API_TOKEN"))
	if want == "" {
		return true
	}

	got := strings.TrimSpace(c.GetHeader("Authorization"))
	got = strings.TrimPrefix(got, "Bearer ")
	got = strings.TrimSpace(got)

	if got != want {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return false
	}
	return true
}

func buildWhitelistAddParams(req whitelistReq) (whitelist.AddParams, error) {
	var p whitelist.AddParams
	member := strings.ToLower(strings.TrimSpace(req.Member))

	rpcURL := firstNonEmpty(os.Getenv("REBASE_RPC"), os.Getenv("DCS_RPC"))
	pkgID := strings.TrimSpace(os.Getenv("DCS_PACKAGE_ID"))
	wlID := firstNonEmpty(strings.TrimSpace(req.WhitelistID), os.Getenv("DCS_WHITELIST_ID"))

	gcPriv := strings.TrimSpace(os.Getenv("GC_PRIVATE_KEY"))
	gcAddr := strings.ToLower(strings.TrimSpace(os.Getenv("GC_ADDRESS")))
	gcGas := strings.TrimSpace(os.Getenv("GC_WALLET_GAS_ID"))

	p.WhitelistID = wlID
	p.Member = member
	p.PackageID = pkgID
	p.GasID = gcGas
	p.GasBudget = parseUint64Env("WALLET_GAS_BUDGET", 10_000_000)
	p.RPCURL = rpcURL
	p.SignerAddress = gcAddr
	p.PrivateKey = gcPriv

	return p, validateAddParams(p)
}

func buildWhitelistRemoveParams(req whitelistReq) (whitelist.RemoveParams, error) {
	var p whitelist.RemoveParams
	member := strings.ToLower(strings.TrimSpace(req.Member))

	rpcURL := firstNonEmpty(os.Getenv("REBASE_RPC"), os.Getenv("DCS_RPC"))
	pkgID := strings.TrimSpace(os.Getenv("DCS_PACKAGE_ID"))
	wlID := firstNonEmpty(strings.TrimSpace(req.WhitelistID), os.Getenv("DCS_WHITELIST_ID"))

	gcPriv := strings.TrimSpace(os.Getenv("GC_PRIVATE_KEY"))
	gcAddr := strings.ToLower(strings.TrimSpace(os.Getenv("GC_ADDRESS")))
	gcGas := strings.TrimSpace(os.Getenv("GC_WALLET_GAS_ID"))

	p.WhitelistID = wlID
	p.Member = member
	p.PackageID = pkgID
	p.GasID = gcGas
	p.GasBudget = parseUint64Env("WALLET_GAS_BUDGET", 10_000_000)
	p.RPCURL = rpcURL
	p.SignerAddress = gcAddr
	p.PrivateKey = gcPriv

	return p, validateRemoveParams(p)
}

func validateAddParams(p whitelist.AddParams) error {
	var missing []string
	if strings.TrimSpace(p.RPCURL) == "" {
		missing = append(missing, "REBASE_RPC/DCS_RPC")
	}
	if strings.TrimSpace(p.PackageID) == "" {
		missing = append(missing, "DCS_PACKAGE_ID")
	}
	if strings.TrimSpace(p.WhitelistID) == "" {
		missing = append(missing, "DCS_WHITELIST_ID")
	}
	if strings.TrimSpace(p.PrivateKey) == "" {
		missing = append(missing, "GC_PRIVATE_KEY")
	}
	if strings.TrimSpace(p.SignerAddress) == "" {
		missing = append(missing, "GC_ADDRESS")
	}
	if strings.TrimSpace(p.GasID) == "" {
		missing = append(missing, "GC_WALLET_GAS_ID")
	}
	if len(missing) > 0 {
		return errorf("missing env: %s", strings.Join(missing, ", "))
	}
	return nil
}

func validateRemoveParams(p whitelist.RemoveParams) error {
	var missing []string
	if strings.TrimSpace(p.RPCURL) == "" {
		missing = append(missing, "REBASE_RPC/DCS_RPC")
	}
	if strings.TrimSpace(p.PackageID) == "" {
		missing = append(missing, "DCS_PACKAGE_ID")
	}
	if strings.TrimSpace(p.WhitelistID) == "" {
		missing = append(missing, "DCS_WHITELIST_ID")
	}
	if strings.TrimSpace(p.PrivateKey) == "" {
		missing = append(missing, "GC_PRIVATE_KEY")
	}
	if strings.TrimSpace(p.SignerAddress) == "" {
		missing = append(missing, "GC_ADDRESS")
	}
	if strings.TrimSpace(p.GasID) == "" {
		missing = append(missing, "GC_WALLET_GAS_ID")
	}
	if len(missing) > 0 {
		return errorf("missing env: %s", strings.Join(missing, ", "))
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parseUint64Env(key string, def uint64) uint64 {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return def
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return def
	}
	return v
}

func jsonOrString(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	if json.Valid(b) {
		return json.RawMessage(b)
	}
	return string(b)
}

type simpleErr string

func (e simpleErr) Error() string { return string(e) }

func errorf(format string, args ...any) error {
	return simpleErr(sprintf(format, args...))
}

func sprintf(format string, args ...any) string {
	out := format
	for _, a := range args {
		out = strings.Replace(out, "%s", toString(a), 1)
	}
	return out
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}
