package stores

import (
	"context"
	"slices"

	"github.com/cupogo/andvari/models/comm"
	"github.com/cupogo/andvari/models/field"
	"github.com/cupogo/andvari/models/oid"
	"github.com/liut/morign/pkg/settings"
	auth "github.com/liut/simpauth"
)

type User = auth.User

// UserFromContext gets user information from context
func UserFromContext(ctx context.Context) (*User, bool) {
	return auth.UserFromContext(ctx)
}

// IsKeeper checks if the user in the current context has keeper role or UID
func IsKeeper(ctx context.Context) bool {
	user, ok := UserFromContext(ctx)
	if !ok {
		return false
	}
	// 检查 UID 是否在白名单中
	if len(settings.Current.KeeperUIDs) > 0 && slices.Contains(settings.Current.KeeperUIDs, user.UID) {
		return true
	}
	return slices.Contains(user.Roles, settings.Current.KeeperRole)
}

type ModelMeta = comm.ModelMeta

// ModelMetaCreator is the model metadata creator interface
type ModelMetaCreator interface {
	ModelMeta
	comm.ModelCreator
}

type ModelMetaOwner interface {
	ModelMeta
	comm.ModelCreator
	comm.ModelOwner
}

// OpModelCreator gets model creator information from context
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

// DbOpModelMetaCreator sets model creator information in database operations
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

func DbOpModelMetaOwner(ctx context.Context, db ormDB, obj ModelMetaOwner) (err error) {
	ownID := obj.GetOwnerID()
	if !ownID.Valid() {
		return
	}
	if _, cat, _ := ownID.Split(); cat != oid.OtAccount {
		logger().Infow("not account", "ownID", ownID)
		return
	}
	return opModelMetaSet(ctx, obj, field.MetaOwner, obj.GetOwnerID(), func(ctx context.Context, id oid.OID) (any, error) {
		user := new(ConvoUser)
		err := dbGetWithPKID(ctx, db, user, id)
		if err != nil {
			return "", err
		}
		return user.Nickname, nil
	})
}

// dbModelMetaUps updates model metadata before and after database operations
func dbModelMetaUps(ctx context.Context, db ormDB, obj Model) {
	if v, ok := obj.(ModelMetaCreator); ok {
		_ = DbOpModelMetaCreator(ctx, db, v)
	} else if v, ok := obj.(comm.ModelCreator); ok {
		_, _ = OpModelCreator(ctx, v)
	}

	if v, ok := obj.(ModelMetaOwner); ok {
		_ = DbOpModelMetaOwner(ctx, db, v)
	}

}
