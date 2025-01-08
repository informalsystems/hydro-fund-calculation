package process

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"fund_calculation/types"
)

// MergeDeploymentVenues reads CSV data from filename
// and merges deployment venues into the proposals array.
func MergeDeploymentVenues(filename string, proposals []types.Proposal) error {
	// 1. Open the CSV file
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)

	// 2. Read the header row
	header, err := csvReader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Build a map: column name -> column index
	headerMap := make(map[string]int)
	for i, colName := range header {
		headerMap[strings.ToLower(colName)] = i
	}

	// Verify we have the columns we need
	requiredCols := []string{
		"proposal_id",
		"contract_address",
		"percentage",
		"denom",
		"deployment_type",
	}
	for _, col := range requiredCols {
		if _, ok := headerMap[col]; !ok {
			return fmt.Errorf("missing required column '%s' in CSV", col)
		}
	}

	// 3. Build a map: proposal_id -> slice of DeploymentVenue
	mapVenues := make(map[uint64][]types.DeploymentVenue)

	// 4. Read each data row
	for {
		record, err := csvReader.Read()
		if err != nil {
			// break on EOF
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("failed to read CSV record: %w", err)
		}

		// Extract columns by name
		proposalIDStr := record[headerMap["proposal_id"]]
		contractAddr := record[headerMap["contract_address"]]
		deplPercentageStr := record[headerMap["percentage"]]
		denom := record[headerMap["denom"]]
		deploymentType := record[headerMap["deployment_type"]]
		existingTvlStr := record[headerMap["existing_tvl"]]
		bootstrapStr := record[headerMap["bootstrap_eligible"]]

		// Convert proposal_id to uint64
		proposalID, err := strconv.ParseUint(proposalIDStr, 10, 64)
		if err != nil {
			fmt.Printf("Skipping invalid proposal_id: %s\n", proposalIDStr)
			continue
		}

		// Convert existing_tvl from string to float64
		existingTvl, err := strconv.ParseFloat(existingTvlStr, 64)
		if err != nil {
			existingTvl = 0.0 // or handle error
		}

		// Convert existing_tvl from string to float64
		deplPercentage, err := strconv.ParseFloat(deplPercentageStr, 64)
		if err != nil {
			existingTvl = 0.0 // or handle error
		}

		// Convert bootstrap_eligible from string to bool
		// (assuming CSV uses "true"/"false" or "1"/"0")
		bootstrapEligible, err := strconv.ParseBool(bootstrapStr)
		if err != nil {
			bootstrapEligible = false // or handle error
		}

		venue := types.DeploymentVenue{
			ContractAddress:   contractAddr,
			Percentage:        deplPercentage,
			Denom:             denom,
			DeploymentType:    deploymentType,
			ExistingTVL:       existingTvl,
			BootstrapEligible: bootstrapEligible,
		}

		mapVenues[proposalID] = append(mapVenues[proposalID], venue)
	}

	// 5. Merge data into proposals
	for i, p := range proposals {
		if vSlice, ok := mapVenues[p.ProposalID]; ok {
			proposals[i].DeploymentVenues = vSlice
		}
	}

	return nil
}
