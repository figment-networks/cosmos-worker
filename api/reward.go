package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/figment-networks/indexer-manager/structs"

	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
)

// GetReward fetches total rewards for delegator account
func (c *Client) GetReward(ctx context.Context, params structs.HeightAccount) (rresp structs.GetRewardResponse, err error) {
	rresp.Height = params.Height
	endpoint := fmt.Sprintf("/distribution/delegators/%v/rewards", params.Account)

	req, err := http.NewRequest(http.MethodGet, c.baseURL+endpoint, nil)
	if err != nil {
		return rresp, err
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

	q := req.URL.Query()
	if params.Height > 0 {
		q.Add("height", strconv.FormatUint(params.Height, 10))
	}

	req.URL.RawQuery = q.Encode()

	err = c.rateLimiter.Wait(ctx)
	if err != nil {
		return rresp, err
	}

	n := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return rresp, err
	}
	rawRequestDuration.WithLabels(endpoint, resp.Status).Observe(time.Since(n).Seconds())
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	fmt.Println("resp.StatusCode ", resp.StatusCode)

	if resp.StatusCode > 399 {
		var result rest.ErrorResponse
		if err = decoder.Decode(&result); err != nil {
			return rresp, fmt.Errorf("[COSMOS-API] Error fetching rewards: %d", resp.StatusCode)
		}
		return rresp, fmt.Errorf("[COSMOS-API] Error fetching rewards: %s ", result.Error)
	}

	type ResponseWithHeight struct {
		Height string                                   `json:"height"`
		Result types.QueryDelegatorTotalRewardsResponse `json:"result"`
	}

	var result ResponseWithHeight
	if err = decoder.Decode(&result); err != nil {
		return rresp, err
	}

	if len(result.Result.Total) < 1 {
		return rresp, nil
	}

	rresp.Rewards = structs.TransactionAmount{
		Text:     result.Result.Total[0].Amount.String(),
		Numeric:  result.Result.Total[0].Amount.BigInt(),
		Currency: result.Result.Total[0].Denom,
		Exp:      sdk.Precision,
	}

	// round rewards up?

	return rresp, err
}
