package query

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"fund_calculation/config"
	"fund_calculation/types"
)

// OuterResponse holds the "data" field returned by the LCD
type OuterResponse struct {
	Data InnerData `json:"data"`
}

// InnerData holds the "proposals" array
type InnerData struct {
	Proposals []types.Proposal `json:"proposals"`
}

// queryLCD is a helper function that encodes a query message in base64, sends it to the LCD,
// and returns the raw array of proposals (missing the merging logic).
func queryLCD(queryMsg []byte) ([]types.Proposal, error) {
	queryB64 := base64.StdEncoding.EncodeToString(queryMsg)
	url := fmt.Sprintf("%s/cosmwasm/wasm/v1/contract/%s/smart/%s",
		config.GlobalConfig.LCDURL, config.GlobalConfig.ContractAddress, queryB64)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query LCD: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyErr, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("LCD query failed (HTTP %d): %s", resp.StatusCode, string(bodyErr))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read LCD response: %w", err)
	}

	var outer OuterResponse
	err = json.Unmarshal(body, &outer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract response: %w", err)
	}

	return outer.Data.Proposals, nil
}

// QueryRoundProposals fetches the "round_proposals" from the contract
// with the specified limit, round_id, start_from, and tranche_id.
func QueryRoundProposals(roundID, trancheID, startFrom, limit uint64) ([]types.Proposal, error) {
	queryMsg := fmt.Sprintf(
		`{"round_proposals":{"limit":%d,"round_id":%d,"start_from":%d,"tranche_id":%d}}`,
		limit, roundID, startFrom, trancheID,
	)

	return queryLCD([]byte(queryMsg))
}

// QueryTopNProposals fetches the "top_n_proposals" from the contract
// which gives us the updated percentages for each proposal.
func QueryTopNProposals(numberOfProposals, roundID, trancheID uint64) ([]types.Proposal, error) {
	queryMsg := fmt.Sprintf(
		`{"top_n_proposals":{"number_of_proposals":%d,"round_id":%d,"tranche_id":%d}}`,
		numberOfProposals, roundID, trancheID,
	)

	return queryLCD([]byte(queryMsg))
}

// QueryContract is the main entry point to fetch and merge data
// from both round_proposals and top_n_proposals.
func QueryContract() ([]types.Proposal, error) {
	// 1. Query the "round_proposals"
	roundProposals, err := QueryRoundProposals(config.GlobalConfig.RoundID, 1, 0, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to query round proposals: %w", err)
	}

	// 2. Query the "top_n_proposals" to get the updated percentages
	topNProposals, err := QueryTopNProposals(100, 1, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to query top_n_proposals: %w", err)
	}

	// 3. Merge the percentages from topNProposals into roundProposals
	//
	//    We'll do this by building a map for O(1) lookup
	//    using proposal_id as the key.
	topMap := make(map[uint64]string)
	for _, tp := range topNProposals {
		topMap[tp.ProposalID] = tp.Percentage
	}

	// 4. Update roundProposals with the percentage found in topNProposals
	for i := range roundProposals {
		if newPct, ok := topMap[roundProposals[i].ProposalID]; ok {
			roundProposals[i].Percentage = newPct
		}
	}

	return roundProposals, nil
}

type deployedFund struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

type liquidityDeployment struct {
	RoundID         uint64         `json:"round_id"`
	TrancheID       uint64         `json:"tranche_id"`
	ProposalID      uint64         `json:"proposal_id"`
	Destinations    []string       `json:"destinations"`
	DeployedFunds   []deployedFund `json:"deployed_funds"`
	FundsBefore     []deployedFund `json:"funds_before_deployment"`
	TotalRounds     uint64         `json:"total_rounds"`
	RemainingRounds uint64         `json:"remaining_rounds"`
}

type liquidityDeploymentResponse struct {
	Data struct {
		LiquidityDeployment liquidityDeployment `json:"liquidity_deployment"`
	} `json:"data"`
}

func buildLiquidityDeploymentQuery(proposalID, roundID, trancheID uint64) []byte {
	queryMsg := fmt.Sprintf(`{"liquidity_deployment":{"proposal_id":%d,"round_id":%d,"tranche_id":%d}}`,
		proposalID, roundID, trancheID)
	return []byte(queryMsg)
}

func QueryLiquidityDeploymentTotal(proposalID, roundID, trancheID uint64) (uint64, error) {
	msg := buildLiquidityDeploymentQuery(proposalID, roundID, trancheID)
	queryB64 := base64.StdEncoding.EncodeToString(msg)

	// Build the LCD endpoint (or adapt if using another style)
	url := fmt.Sprintf("%s/cosmwasm/wasm/v1/contract/%s/smart/%s",
		config.GlobalConfig.LCDURL,
		config.GlobalConfig.ContractAddress,
		queryB64,
	)

	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to query liquidity_deployment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyErr, _ := ioutil.ReadAll(resp.Body)
		return 0, fmt.Errorf("liquidity_deployment query failed: %d %s", resp.StatusCode, string(bodyErr))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read liquidity_deployment response: %w", err)
	}

	var out liquidityDeploymentResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return 0, fmt.Errorf("failed to parse liquidity_deployment response: %w", err)
	}

	deployedFunds := out.Data.LiquidityDeployment.DeployedFunds
	if len(deployedFunds) == 0 {
		// If no funds are deployed, we consider it zero
		return 0, nil
	}

	// Sum up all the deployed funds where denom is "uatom"
	var total uint64
	for _, fund := range deployedFunds {
		if fund.Denom == "uatom" {
			amount, err := strconv.ParseUint(fund.Amount, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse deployed fund amount: %w", err)
			}
			total += amount
		}
	}

	// normalize the total by converting to ATOM (divide by 1_000_000)
	total /= 1_000_000

	return total, nil
}
