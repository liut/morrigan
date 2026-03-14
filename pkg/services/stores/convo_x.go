package stores

import (
	"context"
	"errors"

	"github.com/liut/morign/pkg/models/convo"
)

// ConvoStoreX is the conversation storage extension interface
type ConvoStoreX interface {
	SaveSession(ctx context.Context, sess *convo.Session) error
	SaveUser(ctx context.Context, user *ConvoUser) error
}

// SaveSession saves the session to database
func (s convoStore) SaveSession(ctx context.Context, obj *convo.Session) error {
	if !obj.IsZeroID() {
		exist := new(convo.Session)
		if err := dbGetWithPKID(ctx, s.w.db, exist, obj.ID); err == nil {
			exist.SetIsUpdate(true)
			exist.SetWith(convo.SessionSet{
				MessageCount: &obj.MessageCount,
			})
			dbMetaUp(ctx, s.w.db, exist)
			return dbUpdate(ctx, s.w.db, obj)
		}
	}
	dbMetaUp(ctx, s.w.db, obj)
	return dbInsert(ctx, s.w.db, obj)
}

// SaveUser saves or updates user information
func (s *convoStore) SaveUser(ctx context.Context, user *convo.User) error {
	// 根据 username 查询用户是否存在
	existing := new(convo.User)
	err := dbGetWithUnique(ctx, s.w.db, existing, "username", user.Username)
	if err == nil {
		// 用户存在，更新
		existing.SetIsUpdate(true)
		existing.SetWith(convo.UserSet{
			Nickname:   &user.Nickname,
			AvatarPath: &user.AvatarPath,
		})
		dbMetaUp(ctx, s.w.db, existing)
		return dbUpdate(ctx, s.w.db, existing)
	}

	if errors.Is(err, ErrNoRows) || errors.Is(err, ErrNotFound) {
		// 用户不存在，创建
		dbMetaUp(ctx, s.w.db, user)
		return dbInsert(ctx, s.w.db, user)
	}

	logger().Infow("save user fail", "err", err, "user", user)

	return err
}
