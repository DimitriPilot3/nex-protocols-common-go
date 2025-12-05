package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/lib/pq"
)

// UpdateGameAttribute updates an attribute on a matchmake session
//
// attribIndex is 1-based, such that the first attribute is at index 1.
// This function may have undefined behavior if attributeIndex is zero or greater than the number of attributes currently stored.
func UpdateGameAttribute(manager *common_globals.MatchmakingManager, gatheringID uint32, attributeIndex uint32, newValue uint32) *nex.Error {
	if attributeIndex == 0 {
		return nex.NewError(nex.ResultCodes.Core.InvalidIndex, "change_error")
	}

	result, err := manager.Database.Exec(`UPDATE matchmaking.matchmake_sessions SET attribs[$1]=$2 WHERE id=$3`, attributeIndex, newValue, gatheringID)
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

// UpdateGameAttributes updates all the attributes on a matchmake session
func UpdateGameAttributes(manager *common_globals.MatchmakingManager, gatheringID uint32, attributes types.List[types.UInt32]) *nex.Error {
	result, err := manager.Database.Exec(`UPDATE matchmaking.matchmake_sessions SET attribs=$1 WHERE id=$2`, pq.Array(attributes), gatheringID)
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
