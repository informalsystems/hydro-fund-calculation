package query

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

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
	roundProposals, err := QueryRoundProposals(1, 1, 0, 100)
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
