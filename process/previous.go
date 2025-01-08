package process

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"fund_calculation/query"
	"fund_calculation/types"
)

// MergePreviousProposalIDs reads a CSV that has columns:
//
//	proposal_id, previous_proposal_id
//
// and attaches the previous ID to each existing proposal (optionally).
func MergePreviousProposalIDs(filename string, proposals []types.Proposal) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)

	// Read the header row
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Build a map: colName -> colIndex (case-insensitive)
	headerMap := make(map[string]int)
	for i, col := range header {
		headerMap[strings.ToLower(col)] = i
	}

	requiredCols := []string{"proposal_id", "previous_proposal_id"}
	for _, rc := range requiredCols {
		if _, ok := headerMap[rc]; !ok {
			return fmt.Errorf("CSV missing required column '%s'", rc)
		}
	}

	// Build a map: currentProposalID -> *uint64
	prevMap := make(map[uint64]*uint64)

	for {
		record, err := reader.Read()
		if err != nil {
			// break on EOF
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("failed to read CSV record: %w", err)
		}
		// Parse needed columns
		currentIDStr := record[headerMap["proposal_id"]]
		prevIDStr := record[headerMap["previous_proposal_id"]]

		currentID, err := strconv.ParseUint(currentIDStr, 10, 64)
		if err != nil {
			fmt.Printf("Skipping invalid current proposal_id: %s\n", currentIDStr)
			continue
		}

		// If previous_proposal_id is empty, we skip -> means "no previous proposal".
		if strings.TrimSpace(prevIDStr) == "" {
			prevMap[currentID] = nil
			continue
		}

		prevIDVal, err := strconv.ParseUint(prevIDStr, 10, 64)
		if err != nil {
			fmt.Printf("Skipping invalid previous_proposal_id: %s\n", prevIDStr)
			continue
		}

		prevMap[currentID] = &prevIDVal
	}

	// Merge into proposals array
	for i := range proposals {
		currID := proposals[i].ProposalID
		if pptr, ok := prevMap[currID]; ok {
			proposals[i].PreviousProposalID = pptr
		}
	}

	return nil
}

func MergePreviousFunds(proposals []types.Proposal) {
	for i, p := range proposals {
		// Decide how to pick previous proposal ID / round ID
		// In this example, we assume we use p.PreviousProposalID if valid
		if p.PreviousProposalID == nil {
			proposals[i].PreviousFunds = 0
			continue
		}
		prevID := *p.PreviousProposalID
		roundID := p.RoundID - 1
		trancheID := uint64(1)

		funds, err := query.QueryLiquidityDeploymentTotal(prevID, roundID, trancheID)
		if err != nil {
			fmt.Printf("Warning: could not get previous funds for proposal=%d -> %v\n", p.ProposalID, err)
			continue
		}

		// Assign the single integer
		proposals[i].PreviousFunds = funds
	}
}
