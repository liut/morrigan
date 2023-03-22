package stores

import (
	"context"
	"os"

	"github.com/cupogo/andvari/models/oid"
	"gopkg.in/yaml.v3"

	"github.com/liut/morrigan/pkg/models/conversatio"
	"github.com/liut/morrigan/pkg/settings"
)

type Conversation interface {
	GetID() string
	ListHistory(ctx context.Context) (conversatio.HistoryItems, error)
}

func NewConversation(id any) Conversation {
	cid := oid.Cast(id)
	if cid.IsZero() {
		cid = oid.NewID(oid.OtEvent)
	}
	return &conversation{cid, SgtRC()}
}

type conversation struct {
	id oid.OID
	rc RedisClient
}

func (s *conversation) GetID() string {
	return s.id.String()
}

func (s *conversation) ListHistory(ctx context.Context) (data conversatio.HistoryItems, err error) {
	key := s.getKey()
	s.rc.LRange(ctx, key, 0, -1)
	return
}

func (s *conversation) getKey() string {
	return s.GetID()
}

func LoadPreset() (doc *conversatio.Preset, err error) {
	if len(settings.Current.PresetFile) > 0 {
		logger().Infow("load preset", "file", settings.Current.PresetFile)
		yf, err := os.Open(settings.Current.PresetFile)
		if err != nil {
			return nil, err
		}
		defer yf.Close()
		doc = new(conversatio.Preset)
		err = yaml.NewDecoder(yf).Decode(doc)
		if err != nil {
			return nil, err
		}
	}

	return
}
