package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_db "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

// * EXPERIMENTAL!

func (commonProtocol *CommonProtocol) updateMatchmakeSessionPart(err error, packet nex.PacketInterface, callID uint32, updateMatchmakeSessionParam match_making_types.UpdateMatchmakeSessionParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	commonProtocol.manager.Mutex.Lock()

	gid := updateMatchmakeSessionParam.GID
	session, _, nexError := database.GetMatchmakeSessionByID(commonProtocol.manager, endpoint, uint32(gid))
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	if !session.Gathering.OwnerPID.Equals(connection.PID()) {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.RendezVous.PermissionDenied, "change_error")
	}

	if (updateMatchmakeSessionParam.ModificationFlag & 0x60) == 0x60 {
		// * MatchmakeParam can be updated (0x20) or overridden (0x40), but not both simultaneously
		commonProtocol.manager.Mutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.RendezVous.InvalidOperation, "change_error")
	}

	//tx, err := commonProtocol.manager.Database.Begin()
	//if err != nil {
	//common_globals.Logger.Error("Could not begin transaction")
	//commonProtocol.manager.Mutex.Unlock()
	//return nil, nex.NewError(nex.ResultCodes.Core.Unknown, "change_error")
	//}

	common_globals.Logger.Infof("+++ UpdateMatchmakeSessionPart (gid=%d, ModificationFlag=%d) +++", gid, updateMatchmakeSessionParam.ModificationFlag)

	nexError = match_making_db.UpdateParticipationPolicy(commonProtocol.manager, uint32(gid), uint32(updateMatchmakeSessionParam.ParticipationPolicy), uint32(updateMatchmakeSessionParam.PolicyArgument))
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}
	if session.ParticipationPolicy != updateMatchmakeSessionParam.ParticipationPolicy {
		common_globals.Logger.Infof("changed ParticipationPolicy: %d => %d", session.ParticipationPolicy, updateMatchmakeSessionParam.ParticipationPolicy)
	}
	if session.PolicyArgument != updateMatchmakeSessionParam.PolicyArgument {
		common_globals.Logger.Infof("changed PolicyArgument: %d => %d", session.PolicyArgument, updateMatchmakeSessionParam.PolicyArgument)
	}

	if (updateMatchmakeSessionParam.ModificationFlag & 0x1) != 0 {
		nexError := database.UpdateGameAttributes(commonProtocol.manager, uint32(gid), updateMatchmakeSessionParam.Attributes)
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}
		common_globals.Logger.Infof("updated Attributes: %d => %d", session.Attributes, updateMatchmakeSessionParam.Attributes)
	}

	if (updateMatchmakeSessionParam.ModificationFlag & 0x2) != 0 {
		nexError := database.UpdateParticipation(commonProtocol.manager, uint32(gid), bool(updateMatchmakeSessionParam.OpenParticipation))
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}
		common_globals.Logger.Infof("updated OpenParticipation: %t => %t", session.OpenParticipation, updateMatchmakeSessionParam.OpenParticipation)
	}

	if (updateMatchmakeSessionParam.ModificationFlag & 0x4) != 0 {
		nexError := database.UpdateApplicationBuffer(commonProtocol.manager, uint32(gid), updateMatchmakeSessionParam.ApplicationBuffer)
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}
		common_globals.Logger.Infof("updated ApplicationBuffer: %s => %s", session.ApplicationBuffer, updateMatchmakeSessionParam.ApplicationBuffer)
	}

	if (updateMatchmakeSessionParam.ModificationFlag & 0x8) != 0 {
		nexError := database.UpdateProgressScore(commonProtocol.manager, uint32(gid), uint8(updateMatchmakeSessionParam.ProgressScore))
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}
		common_globals.Logger.Infof("updated ProgressScore: %d => %d", session.ProgressScore, updateMatchmakeSessionParam.ProgressScore)
	}

	if (updateMatchmakeSessionParam.ModificationFlag & 0x10) != 0 {
		common_globals.Logger.Warning("not implemented: update Option0")
	}

	// * TODO: please review this
	switch updateMatchmakeSessionParam.ModificationFlag & (0x20 | 0x40) {
	case 0x20:
		// MatchmakeSessionModificationFlagMatchmakeParam:
		// This flag will entirely replace the old MatchmakeParam.
		matchmakeParamOut := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())
		updateMatchmakeSessionParam.MatchmakeParam.WriteTo(matchmakeParamOut)

		nexError = database.UpdateMatchmakeParam(commonProtocol.manager, uint32(gid), matchmakeParamOut.Bytes())
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}
		common_globals.Logger.Infof("updated MatchmakeParam: replace %s => %s", session.MatchmakeParam.Params, updateMatchmakeSessionParam.MatchmakeParam.Params)

	case 0x40:
		// MatchmakeSessionModificationFlagMatchmakeParamOverride:
		// This flag will only update existing parameters and add new ones, but does not delete any.
		common_globals.Logger.Infof("not implemented: update MatchmakeParam: override %s => %s", session.MatchmakeParam.Params, updateMatchmakeSessionParam.MatchmakeParam.Params)
		/*
			matchmakeParam := session.MatchmakeParam
			matchmakeParamOut := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())
			maps.Copy(matchmakeParam.Params, updateMatchmakeSessionParam.MatchmakeParam.Params)
			matchmakeParam.WriteTo(matchmakeParamOut)

			nexError = database.UpdateMatchmakeParam(commonProtocol.manager, uint32(gid), matchmakeParamOut.Bytes())
			if nexError != nil {
				commonProtocol.manager.Mutex.Unlock()
				return nil, nexError
			}
			common_globals.Logger.Infof("updated MatchmakeParam: override %s => %s", session.MatchmakeParam.Params, updateMatchmakeSessionParam.MatchmakeParam.Params)
		*/
	}

	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	if (updateMatchmakeSessionParam.ModificationFlag & 0x80) != 0 {
		common_globals.Logger.Warningf("not implemented: update StartedTime: %s => %s", session.StartedTime, updateMatchmakeSessionParam.StartedTime)
	}

	if (updateMatchmakeSessionParam.ModificationFlag & 0x100) != 0 {
		common_globals.Logger.Warning("not implemented: update UserPassword")
	}

	if (updateMatchmakeSessionParam.ModificationFlag & 0x200) != 0 {
		common_globals.Logger.Warningf("not implemented: update GameMode: %d => %d", session.GameMode, updateMatchmakeSessionParam.GameMode)
	}

	if (updateMatchmakeSessionParam.ModificationFlag & 0x400) != 0 {
		common_globals.Logger.Warningf("not implemented: update Description: `%s` => `%s`", session.Description, updateMatchmakeSessionParam.Description)
	}

	if (updateMatchmakeSessionParam.ModificationFlag & 0x800) != 0 {
		common_globals.Logger.Warningf("not implemented: update MinParticipants: %d => %d", session.MinimumParticipants, updateMatchmakeSessionParam.MinParticipants)
	}

	if (updateMatchmakeSessionParam.ModificationFlag & 0x1000) != 0 {
		common_globals.Logger.Warningf("not implemented: update MaxParticipants: %d => %d", session.MaximumParticipants, updateMatchmakeSessionParam.MaxParticipants)
	}

	if (updateMatchmakeSessionParam.ModificationFlag & 0x2000) != 0 {
		nexError := database.UpdateMatchmakeSystemType(commonProtocol.manager, uint32(gid), uint32(updateMatchmakeSessionParam.MatchmakeSystemType))
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}
		common_globals.Logger.Infof("updated MatchmakeSystemType: %d => %d", session.MatchmakeSystemType, updateMatchmakeSessionParam.MatchmakeSystemType)
	}

	if (updateMatchmakeSessionParam.ModificationFlag & 0x4000) != 0 {
		common_globals.Logger.Warning("not implemented: update Codeword")
	}

	//err = tx.Commit()
	//if err != nil {
	//tx.Rollback()
	//commonProtocol.manager.Mutex.Unlock()
	//return nil, nex.NewError(nex.ResultCodes.Core.Unknown, "change_error")
	//}
	commonProtocol.manager.Mutex.Unlock()

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodUpdateMatchmakeSessionPart
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
