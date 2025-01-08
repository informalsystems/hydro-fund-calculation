This repository contains a Go program for computing the funds allocated to each proposal.

To run it, simply run `go run main.go` in the root directory of this repository.

The program reads the proposals from the contract at a specified address and computes the funds allocated to each proposal.
Supplementary information is given in the `venues.csv` file, where for each proposal, the deployment venues are listed.
For a short explanation of the columns of that file:
- `proposal_id`: the ID of the proposal
- `contract_address`: the address of the venue
- `percentage`: what the desired percentage of the funds allocated to the proposal should be spent at the venue
- `deployment_type`: can either be `dex` or `lending`. This influences the cap on the size of the deployed position relative to the existing tvl of the venue (dex = 33%, lending = 50%).
- `existing_tvl`: the existing tvl of the venue
- `bootstrap_eligible`: whether we should apply the bootstrap rule for this venue (at least 10k ATOM are deployed, even when the existing tvl would be too small to support this amount)