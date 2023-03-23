package stores

import (
	"context"
	"os"
	"time"

	"github.com/cupogo/andvari/models/oid"
	"gopkg.in/yaml.v3"

	"github.com/liut/morrigan/pkg/models/conversatio"
	"github.com/liut/morrigan/pkg/settings"
)

const (
	historyLifetimeS = time.Second * 86400
	historyMaxLength = 25
)

type Conversation interface {
	GetID() string
	AddHistory(ctx context.Context, item *conversatio.HistoryItem) error
	ListHistory(ctx context.Context) (conversatio.HistoryItems, error)
	ClearHistory(ctx context.Context) error
}

func NewConversation(id any) Conversation {
	cid := oid.Cast(id)
	if cid.IsZero() {
		cid = oid.NewID(oid.OtEvent)
	}
	return &conversation{id: cid, rc: SgtRC()}
}

type conversation struct {
	id oid.OID
	rc RedisClient
}

func (s *conversation) GetID() string {
	return s.id.String()
}

func (s *conversation) AddHistory(ctx context.Context, item *conversatio.HistoryItem) error {
	key := s.getKey()
	b, err := item.MarshalBinary()
	if err != nil {
		return err
	}
	res := s.rc.RPush(ctx, key, b)
	logger().Infow("add history", "item", item, "res", res)
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
		logger().Infow("add history fail", "err", err)
	}
	return err
}

func (s *conversation) ListHistory(ctx context.Context) (data conversatio.HistoryItems, err error) {
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

func LoadPreset() (doc *conversatio.Preset, err error) {
	doc = new(conversatio.Preset)
	if len(settings.Current.PresetFile) > 0 {
		logger().Infow("load preset", "file", settings.Current.PresetFile)
		yf, err := os.Open(settings.Current.PresetFile)
		if err != nil {
			return nil, err
		}
		defer yf.Close()
		err = yaml.NewDecoder(yf).Decode(doc)
		if err != nil {
			logger().Infow("decode preset fail", "err", err)
			return nil, err
		}
	}

	return
}
