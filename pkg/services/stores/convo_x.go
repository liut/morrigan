package stores

import (
	"context"
	"errors"

	"github.com/liut/morign/pkg/models/convo"
)

type ConvoStoreX interface {
	SaveUser(ctx context.Context, user *ConvoUser) error
}

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
