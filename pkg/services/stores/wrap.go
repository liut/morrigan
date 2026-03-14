package stores

import (
	"context"
	"sync"
	"time"

	"github.com/cupogo/andvari/database/embeds"
	"github.com/cupogo/andvari/models/comm"
	"github.com/cupogo/andvari/stores/pgx"
	"github.com/cupogo/andvari/utils"

	"github.com/liut/morign/data/schemas"
	"github.com/liut/morign/pkg/settings"
)

type ormDB = pgx.IDB //nolint
type ormQuery = pgx.SelectQuery
type pgDB = pgx.IDB      //nolint
type pgTx = pgx.Tx       //nolint
type pgIdent = pgx.Ident //nolint
type pgSafe = pgx.Safe   //nolint

type Model = pgx.Model
type PageSpec = comm.PageSpec
type ModelSpec = pgx.ModelSpec
type TextSearchSpec = pgx.TextSearchSpec
type StringsDiff = pgx.StringsDiff

// vars
// nolint
var (
	pgIn = pgx.In

	ErrNoRows   = pgx.ErrNoRows
	ErrNotFound = pgx.ErrNotFound
	ErrEmptyKey = pgx.ErrEmptyKey

	dbGet           = pgx.Get
	dbFirst         = pgx.First
	dbLast          = pgx.Last
	dbEnsureID      = pgx.EnsureID
	dbExists        = pgx.Exists
	queryOne        = pgx.QueryOne
	queryList       = pgx.QueryList
	queryPager      = pgx.QueryPager
	dbGetWithPK     = pgx.ModelWithPK
	dbGetWithPKID   = pgx.ModelWithPKID
	dbGetWithUnique = pgx.ModelWithUnique
	dbGetWith       = pgx.ModelWith
	dbInsert        = pgx.DoInsert
	dbUpdate        = pgx.DoUpdate
	dbDelete        = pgx.DoDelete
	dbDeleteM       = pgx.DoDeleteM
	dbDeleteT       = pgx.DoDeleteT
	dbStoreSimple   = pgx.StoreSimple
	dbMetaUp        = pgx.DoMetaUp

	dbBatchDeleteWithKeyID = pgx.BatchDeleteWithKey

	sift      = pgx.Sift
	siftEqual = pgx.SiftEqual
	siftICE   = pgx.SiftICE
	siftMatch = pgx.SiftMatch
	siftOID   = pgx.SiftOID
	siftOIDs  = pgx.SiftOIDs
	siftDate  = pgx.SiftDate
	isZero    = utils.IsZero

	ContextWithColumns  = pgx.ContextWithColumns
	ColumnsFromContext  = pgx.ColumnsFromContext
	ContextWithRelation = pgx.ContextWithRelation
	RelationFromContext = pgx.RelationFromContext

	RegisterModel = pgx.RegisterModel
)

func init() {
	pgx.RegisterDbFs(embeds.DBFS(), schemas.SchemaFS())
	pgx.RegisterMigrationFs(schemas.UpgradesFS())
	pgx.RegisterMetaUp(dbModelMetaUps)
}

// vars ...
var (
	_ Storage = (*Wrap)(nil)

	dbOnce sync.Once
	dbX    *pgx.DB

	stoOnce sync.Once
	stoW    *Wrap
)

// Wrap implements Storages
type Wrap struct {
	db *pgx.DB

	corpuStore *corpuStore // gened
	mcpStore   *mcpStore   // gened
	convoStore *convoStore // gened
}

// NewWithDB return new instance of Wrap
func NewWithDB(db *pgx.DB) *Wrap {
	w := &Wrap{
		db: db,
	}

	w.corpuStore = &corpuStore{w: w} // gened
	w.mcpStore = &mcpStore{w: w}     // gened
	w.convoStore = &convoStore{w: w} // gened

	// more member stores
	return w
}

// SgtDB start and return a singleton instance of DB
// **Attention**: args only used with fist call
func SgtDB(args ...string) *pgx.DB {
	dbOnce.Do(func() {
		dsn := settings.Current.PgStoreDSN
		tscfg := settings.Current.PgTSConfig
		if len(args) > 0 && len(args[0]) > 0 {
			dsn = args[0]
			if len(args) > 1 {
				tscfg = args[1]
			}
		}
		var err error
		dbX, err = pgx.Open(dsn, tscfg, settings.Current.PgQueryDebug)
		if err != nil {
			logger().Panicw("connect to database fail", "err", err)
		}
	})
	return dbX
}

// Sgt start and return a singleton instance of Storage
func Sgt() *Wrap {
	stoOnce.Do(func() {
		stoW = NewWithDB(SgtDB())
	})
	return stoW
}

// Close closes the database connection
func (w *Wrap) Close() {
	_ = w.db.Close()
}

// InitDB initializes database schema and runs migrations
func InitDB(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	db := SgtDB()
	if err := db.InitSchemas(ctx, false); err != nil {
		logger().Errorw("InitSchemas fail", "err", err)
		return err
	}

	if err := db.RunMigrations(ctx); err != nil {
		logger().Errorw("RunMigrations fail", "err", err)
		return err
	}

	return nil
}

func (w *Wrap) Cob() CobStore     { return w.corpuStore } // Cob gened
func (w *Wrap) KB() CobStore      { return w.corpuStore } // Deprecated: by Cob
func (w *Wrap) MCP() MCPStore     { return w.mcpStore }   // MCP gened
func (w *Wrap) Convo() ConvoStore { return w.convoStore } // Convo gened
