package stores

import (
	"context"

	"github.com/cupogo/andvari/models/comm"
	"github.com/cupogo/andvari/models/field"
	"github.com/cupogo/andvari/models/oid"
	auth "github.com/liut/simpauth"
)

type User = auth.User

func UserFromContext(ctx context.Context) (*User, bool) {
	return auth.UserFromContext(ctx)
}

type ModelMeta = comm.ModelMeta

type ModelMetaCreator interface {
	ModelMeta
	comm.ModelCreator
}

func OpModelCreator(ctx context.Context, obj comm.ModelCreator) (id oid.OID, name string) {
	if user, ok := UserFromContext(ctx); ok {
		if obj.GetCreatorID().IsZero() {
			id := oid.Cast(user.OID)
			obj.SetCreatorID(id)
			return id, user.Name
		}
	}

	return obj.GetCreatorID(), ""
}

func DbOpModelMetaCreator(ctx context.Context, db ormDB, obj ModelMetaCreator) (err error) {
	id, name := OpModelCreator(ctx, obj)
	if !id.Valid() {
		return
	}

	if len(name) > 0 {
		obj.MetaSet(field.MetaCreator, name)
		if obj.CountChange() > 0 {
			obj.SetChange(field.Meta)
		}
	}

	return
}

func dbModelMetaUps(ctx context.Context, db ormDB, obj Model) {
	if v, ok := obj.(ModelMetaCreator); ok {
		_ = DbOpModelMetaCreator(ctx, db, v)
	} else if v, ok := obj.(comm.ModelCreator); ok {
		_, _ = OpModelCreator(ctx, v)
	}
}
