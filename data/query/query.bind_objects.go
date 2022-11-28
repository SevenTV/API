package query

import (
	"context"

	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type QueryBinder struct {
	ctx context.Context
	q   *Query
}

func (q *Query) NewBinder(ctx context.Context) *QueryBinder {
	return &QueryBinder{ctx, q}
}

func (qb *QueryBinder) MapUsers(users []structures.User, roleEnts ...structures.Entitlement[bson.Raw]) (map[primitive.ObjectID]structures.User, error) {
	m := make(map[primitive.ObjectID]structures.User)
	entOW := len(roleEnts) > 0
	for _, v := range users {
		m[v.ID] = v

		if !entOW {
			roleEnts = append(roleEnts, v.Entitlements...)
		}
	}

	m2 := make(map[primitive.ObjectID][]primitive.ObjectID)
	for _, ent := range roleEnts {
		ent, err := structures.ConvertEntitlement[structures.EntitlementDataRole](ent)
		if err != nil {
			return nil, err
		}

		m2[ent.UserID] = append(m2[ent.UserID], ent.Data.ObjectReference)
	}

	roles, _ := qb.q.Roles(qb.ctx, bson.M{})
	if len(roles) > 0 {
		roleMap := make(map[primitive.ObjectID]structures.Role)
		var defaultRole structures.Role
		for _, r := range roles {
			if r.Default {
				defaultRole = r
			}
			roleMap[r.ID] = r
		}
		for key, u := range m {
			roleIDs := make([]primitive.ObjectID, len(m2[u.ID])+len(u.RoleIDs)+1)
			if defaultRole.ID.IsZero() {
				roleIDs[0] = structures.NilRole.ID
			} else {
				roleIDs[0] = defaultRole.ID
			}

			roleIDs[0] = defaultRole.ID
			copy(roleIDs[1:], u.RoleIDs)
			copy(roleIDs[len(u.RoleIDs)+1:], m2[u.ID])

			u.Roles = make([]structures.Role, len(roleIDs)) // allocate space on the user's roles slice
			for i, roleID := range roleIDs {
				if role, ok := roleMap[roleID]; ok { // add role if exists
					u.Roles[i] = role
				} else {
					u.Roles[i] = structures.NilRole // set nil role if role wasn't found
				}
			}

			m[key] = u
		}
	}

	return m, nil
}
