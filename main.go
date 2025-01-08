package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"fund_calculation/config"
	"fund_calculation/process"
	"fund_calculation/query"
)

type Config struct {
	TotalATOM       float64            `json:"totalATOM"`
	VenueFractions  map[string]float64 `json:"venueFractions"`
	ContractAddress string             `json:"contractAddress"`
	LCDURL          string             `json:"lcdURL"`
}

func loadConfig() {
	file, err := os.Open("config.json")
	if err != nil {
		log.Fatalf("Failed to open config file: %v", err)
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var cfg config.Config
	err = json.Unmarshal(bytes, &cfg)
	if err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	config.SetConfig(cfg)
}

func main() {
	loadConfig()

	// 1. Query contract to get proposals
	proposals, err := query.QueryContract()
	if err != nil {
		fmt.Printf("Error querying contract: %v\n", err)
		return
	}

	// 2. Enrich proposals with extra CSV data
	err = process.MergeDeploymentVenues("venues.csv", proposals)
	if err != nil {
		fmt.Printf("Error enriching proposals: %v\n", err)
		return
	}

	// 3. (Optional) Process or allocate ATOM
	process.AllocateToVenues(proposals)

	// 4. Export to CSV or do further analysis
	// process.ExportProposalsToCSV(proposals, "proposals.csv")

	for _, p := range proposals {
		fmt.Printf("Proposal %d: %s received %s percent of votes and receives %f ATOM\n", p.ProposalID, p.Title, p.Percentage, p.AllocatedAtoms)
		for _, v := range p.DeploymentVenues {
			fmt.Printf("  Venue %s: %f ATOM\n", v.ContractAddress, v.VenueAllocatedAtoms)
		}
		fmt.Println("\n")
	}
}
