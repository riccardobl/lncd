package main

import (
	"fmt"

	"regexp"

	"google.golang.org/protobuf/proto"

	"github.com/lightninglabs/lightning-terminal/perms"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	"gopkg.in/macaroon-bakery.v2/bakery"
	"gopkg.in/macaroon.v2"
)

var permUriREGEX = regexp.MustCompile("(\\w+)\\.(\\w+)\\.(\\w+)")

type PermissionManager struct {
	manager *perms.Manager
	conn *Connection
}

func NewPermissionManager(conn* Connection) (*PermissionManager, error) {
	permsMgr, err := perms.NewManager(true)
	if err != nil {
		return nil, err
	}

	return &PermissionManager{
		manager: permsMgr,
		conn: conn,
	}, nil
}

func (mng *PermissionManager) check(permission string) (bool, error) {
	permission = permUriREGEX.ReplaceAllString(permission, "/$1.$2/$3")

	permsMgr := mng.manager
	ops, ok := permsMgr.URIPermissions(permission)
	if !ok {
		log.Debugf("uri %s not found in known permissions list", permission)
		return false, nil
	}

	macaroon := mng.conn.connInfo.macaroon
	if UNSAFE_LOGS {
		log.Debugf("checking permission %s for macaroon %x", permission, macaroon.Id())
	}

	macOps, err := extractMacaroonOps(macaroon)
	if err != nil {
		log.Debugf("could not extract macaroon ops: %v", err)
		return false, nil
	}

	// Check that the macaroon contains each of the required permissions
	// for the given URI.
	return hasPermissions(permission, macOps, ops), nil
}

// extractMacaroonOps is a helper function that extracts operations from the
// ID of a macaroon.
func extractMacaroonOps(mac *macaroon.Macaroon) ([]*lnrpc.Op, error) {
	rawID := mac.Id()

    if len(rawID) == 0 {
        return nil, fmt.Errorf("macaroon ID is empty")
    }

	if rawID[0] != byte(bakery.LatestVersion) {
		return nil, fmt.Errorf("invalid macaroon version: %x", rawID)
	}
    
	if len(rawID) < 2 {
        return nil, fmt.Errorf("invalid macaroon ID length")
    }
	
	decodedID := &lnrpc.MacaroonId{}
	idProto := rawID[1:]
	err := proto.Unmarshal(idProto, decodedID)
	if err != nil {
		return nil, fmt.Errorf("unable to decode macaroon: %v", err)
	}

	return decodedID.Ops, nil
}

func hasPermissions(uri string, macOps []*lnrpc.Op,
	requiredOps []bakery.Op) bool {

	// Create a lookup map of the macaroon operations.
	macOpsMap := make(map[string]map[string]bool)
	for _, op := range macOps {
		macOpsMap[op.Entity] = make(map[string]bool)

		for _, action := range op.Actions {
			macOpsMap[op.Entity][action] = true

			// We account here for the special case where the
			// operation gives access to an entire URI. This is the
			// case when the Entity is equal to the "uri" keyword
			// and when the Action is equal to the URI that access
			// is being granted to.
			if op.Entity == macaroons.PermissionEntityCustomURI &&
				action == uri {

				return true
			}
		}
	}

	// For each of the required operations, we ensure that the macaroon also
	// contains the operation.
	for _, op := range requiredOps {
		macEntity, ok := macOpsMap[op.Entity]
		if !ok {
			return false
		}

		if !macEntity[op.Action] {
			return false
		}
	}

	return true
}

