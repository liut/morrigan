package stores

import (
	"context"
	"os"
	"time"

	"github.com/cupogo/andvari/models/oid"
	"gopkg.in/yaml.v3"

	"github.com/liut/morrigan/pkg/models/aigc"
	"github.com/liut/morrigan/pkg/models/convo"
	"github.com/liut/morrigan/pkg/settings"
)

const (
	historyLifetimeS = time.Second * 86400
	historyMaxLength = 25
)

type Conversation interface {
	GetID() string
	GetOID() oid.OID
	AddHistory(ctx context.Context, item *aigc.HistoryItem) error
	ListHistory(ctx context.Context) (aigc.HistoryItems, error)
	ClearHistory(ctx context.Context) error
}

// NewConversation 创建会话，使用默认 Redis 客户端
func NewConversation(ctx context.Context, id any) Conversation {
	return newConversation(ctx, id, SgtRC())
}

// newConversation 内部构造函数，支持注入 Redis 客户端（用于测试）
func newConversation(ctx context.Context, id any, rc RedisClient) Conversation {
	sto := Sgt()
	cid := oid.Cast(id)
	var sess *convo.Session
	var err error
	if cid.Valid() {
		sess, err = sto.Convo().GetSession(ctx, cid.String())
		if err != nil {
			sess = convo.NewSessionWithID(cid)
		}
	} else {
		sess = new(convo.Session)
		sess.Creating() //nolint
	}

	return &conversation{
		id:   sess.ID,
		rc:   rc,
		sess: sess,
		sto:  sto,
	}
}

type conversation struct {
	id oid.OID
	rc RedisClient

	sess *convo.Session
	sto  Storage
}

func (s *conversation) GetID() string {
	return s.id.String()
}

func (s *conversation) GetOID() oid.OID {
	return s.id
}

// TODO: AddMessages(), Summary()

func (s *conversation) AddHistory(ctx context.Context, item *aigc.HistoryItem) error {
	key := s.getKey()

	// 去重检查：获取最后一条消息，如果内容相同则跳过
	lastMsg, err := s.getLastUserMessage(ctx)
	if err == nil && lastMsg != nil {
		if s.isDuplicate(lastMsg, item) {
			logger().Debugw("duplicate message skipped", "key", key)
			return nil
		}
	}

	b, err := item.MarshalBinary()
	if err != nil {
		return err
	}
	res := s.rc.RPush(ctx, key, b)
	err = res.Err()
	if err == nil {
		count, _ := res.Result()
		if err = s.rc.Expire(ctx, key, historyLifetimeS).Err(); err != nil {
			return err
		}
		if count > historyMaxLength {
			logger().Infow("history length overflow", "count", count)
			err = s.rc.LPop(ctx, key).Err()
		}
	}
	if err != nil {
		logger().Infow("add history fail", "key", key, "err", err)
	}
	return err
}

// getLastUserMessage 获取列表中最后一条消息
func (s *conversation) getLastUserMessage(ctx context.Context) (*aigc.HistoryItem, error) {
	key := s.getKey()
	// LLINDEX key -1 获取最后一条
	b, err := s.rc.LIndex(ctx, key, -1).Bytes()
	if err != nil {
		return nil, err
	}
	var item aigc.HistoryItem
	if err := item.UnmarshalBinary(b); err != nil {
		return nil, err
	}
	return &item, nil
}

// isDuplicate 检查新消息是否与最后一条消息重复
func (s *conversation) isDuplicate(last, new *aigc.HistoryItem) bool {
	// 优先比较 ChatItem.User
	if last.ChatItem != nil && new.ChatItem != nil {
		return last.ChatItem.User == new.ChatItem.User
	}
	// 备用比较 Text
	return last.Text == new.Text
}

func (s *conversation) ListHistory(ctx context.Context) (data aigc.HistoryItems, err error) {
	key := s.getKey()
	ss := s.rc.LRange(ctx, key, 0, -1)
	err = ss.ScanSlice(&data)
	return
}

func (s *conversation) ClearHistory(ctx context.Context) error {
	return s.rc.Del(ctx, s.getKey()).Err()
}

func (s *conversation) getKey() string {
	return "convs-" + s.GetID()
}

func LoadPreset() (doc aigc.Preset, err error) {
	if len(settings.Current.PresetFile) > 0 {
		var yf *os.File
		yf, err = os.Open(settings.Current.PresetFile)
		if err != nil {
			logger().Infow("load preset fail", "file", settings.Current.PresetFile, "err", err)
			return
		}
		defer yf.Close()
		err = yaml.NewDecoder(yf).Decode(&doc)
		if err != nil {
			logger().Infow("decode preset fail", "err", err)
			return
		}
	}

	return
}
