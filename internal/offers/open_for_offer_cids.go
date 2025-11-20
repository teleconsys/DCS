package offers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/machinebox/graphql"
)

type OpenOfferCID struct {
	ID              string `json:"id"`
	Owner           string `json:"owner"`
	CID             string `json:"cid"`
	ClosesInMinutes int64  `json:"closes_in_minutes"` // minutes until offer window closes
	StartsInMinutes int64  `json:"starts_in_minutes"` // <0 means epoch already started
	NextOffers      int    `json:"next_offers"`       // count of pending offers for next epoch?
}

// FindOpenOfferCIDs lists CIDs from cidListID whose offer window is open now:
// current_epoch_end ≤ nowMs < next_epoch_start + 600_000 (10 minutes)
func FindOpenOfferCIDs(ctx context.Context, endpoint, cidListID string, nowMs int64) ([]OpenOfferCID, error) {
	client := graphql.NewClient(endpoint, graphql.WithHTTPClient(&http.Client{Timeout: 15 * time.Second}))

	// 1) Read CIDLIST → []object IDs
	var lr struct {
		Object struct {
			AsMoveObject struct {
				Contents struct {
					JSON map[string]any `json:"json"`
				} `json:"contents"`
			} `json:"asMoveObject"`
		} `json:"object"`
	}
	qList := graphql.NewRequest(`query($id:IotaAddress!){
	  object(address:$id){ asMoveObject{ contents{ json } } }
	}`)
	qList.Var("id", cidListID)
	if err := client.Run(ctx, qList, &lr); err != nil {
		return nil, fmt.Errorf("fetch cid list: %w", err)
	}
	rawIDs, _ := lr.Object.AsMoveObject.Contents.JSON["cidlist"].([]any)
	ids := make([]string, 0, len(rawIDs))
	for _, v := range rawIDs {
		if s, ok := v.(string); ok && s != "" {
			ids = append(ids, s)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	// 2) Read those CIDs and filter windows
	var or struct {
		Objects struct {
			Nodes []struct {
				Address      string `json:"address"`
				AsMoveObject struct {
					Contents struct {
						JSON map[string]any `json:"json"`
					} `json:"contents"`
				} `json:"asMoveObject"`
			} `json:"nodes"`
		} `json:"objects"`
	}
	qObjs := graphql.NewRequest(`query($ids:[IotaAddress!]){
	  objects(filter:{objectIds:$ids}){
	    nodes{ address asMoveObject{ contents{ json } } }
	  }
	}`)
	qObjs.Var("ids", ids)
	if err := client.Run(ctx, qObjs, &or); err != nil {
		return nil, fmt.Errorf("fetch cids: %w", err)
	}

	const tenMinMs = int64(600_000)
	out := make([]OpenOfferCID, 0, len(or.Objects.Nodes))
	for _, n := range or.Objects.Nodes {
		js := n.AsMoveObject.Contents.JSON
		curEnd := asI64(js["current_epoch_end"])
		nextStart := asI64(js["next_epoch_start"])
		if nextStart == 0 {
			continue // no next epoch scheduled
		}
		if nowMs >= curEnd && nowMs < nextStart+tenMinMs {
			item := OpenOfferCID{
				ID:              n.Address,
				Owner:           asString(js["owner"]),
				CID:             decodeBytesString(js["cid_str"]),
				ClosesInMinutes: max0((nextStart + tenMinMs - nowMs) / 60_000),
				StartsInMinutes: (nextStart - nowMs) / 60_000,
				NextOffers:      lenArray(js["next_epoch_offers"]),
			}
			out = append(out, item)
		}
	}
	return out, nil
}
