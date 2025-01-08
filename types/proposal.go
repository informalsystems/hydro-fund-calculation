package types

type Proposal struct {
	ProposalID                  uint64 `json:"proposal_id"`
	Title                       string `json:"title"`
	Description                 string `json:"description"`
	RoundID                     uint64 `json:"round_id"`
	TrancheID                   uint64 `json:"tranche_id"`
	DeploymentDuration          uint64 `json:"deployment_duration"`
	MinimumAtomLiquidityRequest string `json:"minimum_atom_liquidity_request"`
	Percentage                  string `json:"percentage"`
	Power                       string `json:"power"`
	AllocatedAtoms              float64

	// if this proposal was also in the previous round, this will be the id the proposal had before.
	PreviousProposalID *uint64 `json:"previous_proposal_id"`

	DeploymentVenues []DeploymentVenue `json:"deployment_locations"`

	PreviousFunds uint64 `json:"previous_funds"`
}

type DeploymentVenue struct {
	// Fields that are read from the CSV
	ContractAddress   string  `json:"contract_address"`
	Denom             string  `json:"denom"`
	DeploymentType    string  `json:"deployment_type"` // "lending" or "DEX"
	ExistingTVL       float64 `json:"existing_tvl"`    // total value locked
	BootstrapEligible bool    `json:"bootstrap_eligible"`
	Percentage        float64 `json:"percentage"`

	// Computed fields
	VenueLimit          float64 `json:"venue_limit"` // max ATOM we can deposit
	VenueAllocatedAtoms float64 `json:"venue_allocated_atoms"`
}
