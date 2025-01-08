package process

import (
	"math"
	"strconv"

	"fund_calculation/config"
	"fund_calculation/types"
)

// AllocateToVenues first allocates ATOM among proposals by their Percentage,
// then allocates each proposal's slice among that proposal’s venues, capped by venue limits.
func AllocateToVenues(proposals []types.Proposal) {
	//-----------------------------------------------------
	// 1) PROPOSAL-LEVEL DISTRIBUTION
	//-----------------------------------------------------
	// Allocate each proposal's share of the total 300,000
	for i := range proposals {
		perc, err := strconv.ParseFloat(proposals[i].Percentage, 64)
		if err != nil {
			proposals[i].AllocatedAtoms = 0
			continue
		}
		share := perc / 100 * config.GlobalConfig.TotalATOM
		proposals[i].AllocatedAtoms = share
	}

	//-----------------------------------------------------
	// 2) VENUE-LEVEL DISTRIBUTION (per proposal)
	//-----------------------------------------------------
	for i, p := range proposals {
		leftoverWanted := p.AllocatedAtoms
		if leftoverWanted <= 0 {
			continue
		}

		// (A) Compute each venue's fraction-based limit, store in VenueLimit
		for j, v := range p.DeploymentVenues {
			f := venueFraction(v.DeploymentType)
			limit := 0.0
			if f > 0 {
				// formula: limit = f/(1-f)*existingTVL
				limit = (f / (1 - f)) * v.ExistingTVL
			}
			proposals[i].DeploymentVenues[j].VenueLimit = limit
			proposals[i].DeploymentVenues[j].VenueAllocatedAtoms = 0

			// if the limit would be less than 10,000 and the venue is bootstrap eligible, set it to 10,000
			if limit < 10000 && v.BootstrapEligible {
				proposals[i].DeploymentVenues[j].VenueLimit = 10000
			}
		}

		// (B) Convert each venue’s "percentage" field into a numeric weight
		//     for distributing leftoverWanted
		var sumVenueWeights float64
		venueWeights := make([]float64, len(p.DeploymentVenues))
		for j, v := range p.DeploymentVenues {
			venPerc := v.Percentage
			venueWeights[j] = venPerc
			sumVenueWeights += venPerc
		}

		// Build a list of "active" venues
		activeVenues := make([]int, 0)
		for j := range p.DeploymentVenues {
			if venueWeights[j] > 0 && p.DeploymentVenues[j].VenueLimit > 0 {
				activeVenues = append(activeVenues, j)
			}
		}

		// (C) Iteratively distribute leftoverWanted among active venues
		for leftoverWanted > 0 && len(activeVenues) > 0 {
			// sum weights among active
			var sumActiveWeights float64
			for _, idx := range activeVenues {
				sumActiveWeights += venueWeights[idx]
			}
			if sumActiveWeights <= 0 {
				break
			}

			newActive := make([]int, 0)
			removedAny := false

			for _, idx := range activeVenues {
				venue := &proposals[i].DeploymentVenues[idx]

				// fraction of leftoverWanted for this venue
				fWeight := venueWeights[idx] / sumActiveWeights
				desired := fWeight * leftoverWanted

				space := venue.VenueLimit - venue.VenueAllocatedAtoms
				allocate := math.Min(desired, space)

				venue.VenueAllocatedAtoms += allocate
				leftoverWanted -= allocate

				// check if venue still has capacity
				if venue.VenueAllocatedAtoms < venue.VenueLimit-1e-9 {
					// keep active
					newActive = append(newActive, idx)
				} else {
					removedAny = true
				}
			}

			activeVenues = newActive

			if !removedAny {
				// means no one got capped, so we allocated everything proportionally
				// leftoverWanted might be 0 or > 0 if sum weights are fully satisfied
				break
			}
		}

		// leftoverWanted > 0 means we can't place any more ATOM
		// (all venues are maxed out or have no capacity).
	}
}
