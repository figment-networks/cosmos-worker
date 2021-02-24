package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/figment-networks/indexer-manager/structs"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
)

type responseWithHeight struct {
	Height string                                   `json:"height"`
	Result types.QueryDelegatorTotalRewardsResponse `json:"result"`
}

const maxRetries = 3

// GetReward fetches total rewards for delegator account
func (c *Client) GetReward(ctx context.Context, params structs.HeightAccount) (resp structs.GetRewardResponse, err error) {
	resp.Height = params.Height
	endpoint := fmt.Sprintf("%s/distribution/delegators/%v/rewards", c.tendermintLCDAddr, params.Account)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return resp, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", c.datahubKey)

	q := req.URL.Query()
	if params.Height > 0 {
		q.Add("height", strconv.FormatUint(params.Height, 10))
	}

	req.URL.RawQuery = q.Encode()

	err = c.rateLimiter.Wait(ctx)
	if err != nil {
		return resp, err
	}

	var cliResp *http.Response

	for i := 1; i <= maxRetries; i++ {
		n := time.Now()
		cliResp, err = c.httpClient.Do(req)
		if err, ok := err.(net.Error); ok && err.Timeout() && i != maxRetries {
			continue
		} else if err != nil {
			return resp, err
		}
		rawRequestDuration.WithLabels(endpoint, cliResp.Status).Observe(time.Since(n).Seconds())

		defer cliResp.Body.Close()

		if cliResp.StatusCode < 500 {
			break
		}
		time.Sleep(time.Duration(i*500) * time.Millisecond)
	}

	decoder := json.NewDecoder(cliResp.Body)

	if cliResp.StatusCode > 399 {
		var result rest.ErrorResponse
		if err = decoder.Decode(&result); err != nil {
			return resp, fmt.Errorf("[COSMOS-API] Error fetching rewards: %d", cliResp.StatusCode)
		}
		return resp, fmt.Errorf("[COSMOS-API] Error fetching rewards: %s ", result.Error)
	}
	var result responseWithHeight
	if err = decoder.Decode(&result); err != nil {
		return resp, err
	}

	for _, reward := range result.Result.Total {
		resp.Rewards = append(resp.Rewards,
			structs.TransactionAmount{
				Text:     reward.Amount.String(),
				Numeric:  reward.Amount.BigInt(),
				Currency: reward.Denom,
				Exp:      sdk.Precision,
			},
		)
	}

	return resp, err
}
