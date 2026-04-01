package stores

import (
	"context"
	"errors"
	"time"

	"github.com/spf13/cast"

	"github.com/cupogo/andvari/models/oid"
	"github.com/liut/morign/pkg/models/convo"
	"github.com/liut/morign/pkg/models/corpus"
	"github.com/liut/morign/pkg/models/mcps"
)

// ConvoStoreX is the conversation storage extension interface
type ConvoStoreX interface {
	SaveSession(ctx context.Context, sess *convo.Session) error
	SaveUser(ctx context.Context, user *ConvoUser) error
	SyncUserFromOAuth(ctx context.Context, user IUser) error

	GetMyMemoryWithKey(ctx context.Context, key string) (*convo.Memory, error)
	ListMyMomory(ctx context.Context, spec *ConvoMemorySpec) (convo.Memories, error)
	MatchMemories(ctx context.Context, ms MatchSpec) (data convo.Memories, err error)
	SyncEmbeddingMemories(ctx context.Context, spec *ConvoMemorySpec) error

	InvokerForMemoryList() mcps.Invoker
	InvokerForMemoryRecall() mcps.Invoker
	InvokerForMemoryStore() mcps.Invoker
	InvokerForMemoryForget() mcps.Invoker
}

const defaultMemoryLimit = 5
const maxMemoryLimit = 100

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
			Email:      &user.Email,
			Phone:      &user.Phone,
		})
		existing.MergeMeta(user.Meta)
		dbMetaUp(ctx, s.w.db, existing)
		return dbUpdate(ctx, s.w.db, existing, "meta")
	}

	if errors.Is(err, ErrNoRows) || errors.Is(err, ErrNotFound) {
		// 用户不存在，创建
		dbMetaUp(ctx, s.w.db, user)
		return dbInsert(ctx, s.w.db, user)
	}

	logger().Infow("save user fail", "err", err, "user", user)

	return err
}

func (s *convoStore) SyncUserFromOAuth(ctx context.Context, user IUser) error {
	cub := convo.UserBasic{
		Username:   user.GetUID(),
		Nickname:   user.GetName(),
		AvatarPath: user.GetAvatar(),
		Email:      user.GetEmail(),
		Phone:      user.GetPhone(),
	}
	if wuid := WecomUIDFromContext(ctx); len(wuid) > 0 {
		logger().Infow("got wecomUID", "uid", wuid)
		cub.MetaAddKVs("wecomUID", wuid)
	}
	cuser := convo.NewUserWithBasic(cub)
	id := user.GetOID()
	_ = cuser.SetID(id)
	if err := s.SaveUser(ctx, cuser); err != nil {
		logger().Infow("save user failed", "err", err,
			"oid", id, "uid", user.GetUID())
		return err
	}
	return nil
}

func (s *convoStore) GetMyMemoryWithKey(ctx context.Context, key string) (*convo.Memory, error) {
	user, uok := UserFromContext(ctx)
	if !uok {
		return nil, errors.New("need login")
	}
	ownerID := oid.Cast(user.OID)
	obj := new(convo.Memory)
	err := dbGet(ctx, s.w.db, obj, "owner_id = ? AND key iLIKE ?", ownerID, key)

	if err != nil {
		return nil, err
	}

	return obj, nil
}

// afterCreatedMemory generates vector after memory creation
// Uses the same DocVector table as CobDocument
func (s *convoStore) afterCreatedMemory(ctx context.Context, obj *convo.Memory) error {
	subject := obj.GetSubject()
	dvb := corpus.DocVectorBasic{
		DocID:   obj.ID,
		Subject: subject,
	}
	vec, err := GetEmbedding(ctx, dvb.Subject)
	if err != nil {
		return err
	}
	if len(vec) > 0 {
		dvb.Vector = vec
	}

	_, err = s.w.Corpus().CreateDocVector(ctx, dvb)
	if err != nil {
		logger().Infow("create memory vector fail", "dvb", &dvb, "err", err)
		return err
	}
	return nil
}

func (spec *ConvoMemorySpec) SiftX(ctx context.Context, q *ormQuery) *ormQuery {
	if !spec.IsFull {
		q.ExcludeColumn("content")
	}
	if spec.IsOwner {
		user, uok := UserFromContext(ctx)
		if !uok {
			logger().Infow("need login when query owner memories")
			return q.Where("FALSE")
		}
		spec.OwnerID = user.OID
		if spec.Limit == 0 {
			spec.Limit = defaultMemoryLimit
		}
	}
	if len(spec.Sort) == 0 {
		spec.Sort = "created DESC"
	}
	return q
}

func (s *convoStore) ListMyMomory(ctx context.Context, spec *ConvoMemorySpec) (convo.Memories, error) {
	user, uok := UserFromContext(ctx)
	if !uok {
		return nil, errors.New("need login")
	}
	spec.OwnerID = user.OID
	data, _, err := s.ListMemory(ctx, spec)
	return data, err
}

// MatchMemories matches memories using vector similarity
func (s *convoStore) MatchMemories(ctx context.Context, ms MatchSpec) (data convo.Memories, err error) {
	ms.setDefaults()

	// Get embedding for the query
	vec, err := GetEmbedding(ctx, ms.Query)
	if err != nil {
		logger().Infow("GetEmbedding fail", "err", err)
		return
	}

	// Match vectors
	var ps corpus.DocMatches
	ps, err = s.w.Corpus().MatchVectorWith(ctx, vec, ms.Threshold, ms.Limit)
	if err != nil || len(ps) == 0 {
		logger().Infow("no match memories", "query", ms.Query)
		return
	}

	logger().Infow("matched memories", "count", len(ps))

	// Fetch memories by IDs
	spec := &ConvoMemorySpec{IsFull: true}
	spec.IDs = ps.DocumentIDs()
	err = queryList(ctx, s.w.db, spec, &data).Scan(ctx)
	if err != nil {
		logger().Infow("list memories fail", "spec", spec, "err", err)
	}
	return
}

// SyncEmbeddingMemories generates vectors for all memories
func (s *convoStore) SyncEmbeddingMemories(ctx context.Context, spec *ConvoMemorySpec) error {
	spec.IsFull = true
	data, _, err := s.ListMemory(ctx, spec)
	if err != nil {
		return err
	}

	for _, mem := range data {
		subject := mem.GetSubject()
		vec, err := GetEmbedding(ctx, subject)
		if err != nil {
			return err
		}

		exist := new(corpus.DocVector)
		err = dbGetWithUnique(ctx, s.w.db, exist, "doc_id", mem.ID)
		if err == nil {
			exist.SetWith(corpus.DocVectorSet{
				Subject: &subject,
				Vector:  &vec,
			})
			if err = dbUpdate(ctx, s.w.db, exist); err != nil {
				return err
			}
		} else {
			dv := corpus.NewDocVectorWithBasic(corpus.DocVectorBasic{
				DocID:   mem.ID,
				Subject: subject,
				Vector:  vec,
			})
			if err = dbInsert(ctx, s.w.db, dv); err != nil {
				return err
			}
		}
	}
	return nil
}

// InvokerForMemoryList returns an invoker for listing memories
func (s *convoStore) InvokerForMemoryList() mcps.Invoker {
	return func(ctx context.Context, args map[string]any) (map[string]any, error) {

		limit := defaultMemoryLimit
		if l := cast.ToInt(args["limit"]); l > 0 {
			limit = l
			if limit > maxMemoryLimit {
				limit = maxMemoryLimit
			}
		}

		spec := &ConvoMemorySpec{
			Cate: cast.ToString(args["category"]),
		}
		spec.Limit = limit
		spec.Sort = "id desc"

		includeContent := true
		if v, ok := args["include_content"]; ok {
			if ic, err := cast.ToBoolE(v); err == nil {
				includeContent = ic
			}
		}
		spec.IsFull = includeContent

		data, err := s.ListMyMomory(ctx, spec)
		if err != nil {
			return mcps.BuildToolErrorResult(err.Error()), nil
		}

		logger().Debugw("invoke memory list", "args", args, "ic", includeContent)

		var results []map[string]any
		for _, m := range data {
			item := map[string]any{
				"key":      m.Key,
				"category": m.Cate,
			}
			if includeContent {
				item["content"] = m.Content
			}
			results = append(results, item)
		}

		return mcps.BuildToolSuccessResult(results), nil
	}
}

// InvokerForMemoryRecall returns an invoker for searching memories
func (s *convoStore) InvokerForMemoryRecall() mcps.Invoker {
	return func(ctx context.Context, args map[string]any) (map[string]any, error) {
		query := cast.ToString(args["query"])
		if query == "" {
			return mcps.BuildToolErrorResult("missing required argument: query"), nil
		}

		limit := defaultMemoryLimit
		if l := cast.ToInt(args["limit"]); l > 0 {
			limit = l
			if limit > maxMemoryLimit {
				limit = maxMemoryLimit
			}
		}

		// Use vector-based matching
		data, err := s.MatchMemories(ctx, MatchSpec{
			Query: query,
			Limit: limit,
		})
		if err != nil {
			return mcps.BuildToolErrorResult(err.Error()), nil
		}

		if len(data) == 0 {
			return mcps.BuildToolSuccessResult("No matching memories found"), nil
		}

		var results []map[string]any
		for _, m := range data {
			results = append(results, map[string]any{
				"date":     m.GetUpdated().Format(time.DateOnly),
				"key":      m.Key,
				"category": m.Cate,
				"content":  m.Content,
			})
		}

		return mcps.BuildToolSuccessResult(results), nil
	}
}

// InvokerForMemoryStore returns an invoker for storing a memory
func (s *convoStore) InvokerForMemoryStore() mcps.Invoker {
	return func(ctx context.Context, args map[string]any) (map[string]any, error) {
		user, uok := UserFromContext(ctx)
		if !uok {
			return nil, errors.New("need login")
		}
		key := cast.ToString(args["key"])
		if key == "" {
			return mcps.BuildToolErrorResult("missing required argument: key"), nil
		}

		content := cast.ToString(args["content"])
		if content == "" {
			return mcps.BuildToolErrorResult("missing required argument: content"), nil
		}

		category := cast.ToString(args["category"])
		if category == "" {
			category = "custom"
		}

		// Try to get existing memory by key
		existing, err := s.GetMyMemoryWithKey(ctx, key)
		if err == nil {
			// Update existing
			existing.SetIsUpdate(true)
			existing.SetWith(convo.MemorySet{
				Cate:    &category,
				Content: &content,
			})
			dbMetaUp(ctx, s.w.db, existing)
			if err := dbUpdate(ctx, s.w.db, existing); err != nil {
				return mcps.BuildToolErrorResult(err.Error()), nil
			}
			return mcps.BuildToolSuccessResult(map[string]any{
				"action":   "updated",
				"key":      key,
				"category": category,
			}), nil
		}

		// Create new
		mb := convo.MemoryBasic{
			Key:     key,
			Cate:    category,
			Content: content,
		}
		mb.SetOwnerID(user.OID)
		obj, err := s.CreateMemory(ctx, mb)
		if err != nil {
			return mcps.BuildToolErrorResult(err.Error()), nil
		}

		return mcps.BuildToolSuccessResult(map[string]any{
			"action":    "created",
			"key":       key,
			"category":  category,
			"memory_id": obj.StringID(),
		}), nil
	}
}

// InvokerForMemoryForget returns an invoker for deleting a memory
func (s *convoStore) InvokerForMemoryForget() mcps.Invoker {
	return func(ctx context.Context, args map[string]any) (map[string]any, error) {
		key := cast.ToString(args["key"])
		if key == "" {
			return mcps.BuildToolErrorResult("missing required argument: key"), nil
		}

		// Try to get memory by key
		existing, err := s.GetMyMemoryWithKey(ctx, key)
		if err != nil {
			return mcps.BuildToolSuccessResult(map[string]any{
				"action": "not_found",
				"key":    key,
			}), nil
		}

		if err := s.DeleteMemory(ctx, existing.StringID()); err != nil {
			return mcps.BuildToolErrorResult(err.Error()), nil
		}

		return mcps.BuildToolSuccessResult(map[string]any{
			"action": "deleted",
			"key":    key,
		}), nil
	}
}
