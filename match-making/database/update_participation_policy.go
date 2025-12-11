package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// UpdateParticipationPolicy updates the participation policy and the policy argument of a gathering
func UpdateParticipationPolicy(manager *common_globals.MatchmakingManager, gatheringID uint32, participationPolicy uint32, policyArgument uint32) *nex.Error {
	result, err := manager.Database.Exec(`UPDATE matchmaking.gatherings SET participation_policy=$1, policy_argument=$2 WHERE id=$3`, participationPolicy, policyArgument, gatheringID)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	if rowsAffected == 0 {
		return nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
	}

	return nil
}
