package stores

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	openai "github.com/sashabaranov/go-openai"

	"github.com/liut/morrigan/pkg/models/qas"
)

var (
	ErrEmptyParam = errors.New("empty param")

	qaHeads = []string{"title", "heading", "content"}
)

func validHead(rec []string) bool {
	return len(rec) >= len(qaHeads) && rec[0] == qaHeads[0] && rec[1] == qaHeads[1] && rec[2] == qaHeads[2]
}

type qaStoreX interface {
	ImportFromCSV(ctx context.Context, r io.Reader) error
	ConstructPrompt(ctx context.Context, question string) (prompt string, err error)
	MatchDocments(ctx context.Context, question string) (data qas.Documents, err error)
	MatchDocmentsWithVector(ctx context.Context, vec qas.Vector) (data qas.Documents, err error)
}

func (s *qaStore) ImportFromCSV(ctx context.Context, r io.Reader) error {
	rd := csv.NewReader(r)
	rec, err := rd.Read()
	if err != nil {
		return err
	}
	if !validHead(rec) {
		return fmt.Errorf("invalid csv head: %+v", rec)
	}
	for {
		row, err := rd.Read()
		if err != nil {
			return err
		}
		if len(row) < 3 || len(row[0]) == 0 || len(row[1]) == 0 {
			return fmt.Errorf("invalid csv row: %+v", row)
		}
		doc := new(qas.Document)
		err = dbGet(ctx, s.w.db, doc, "title = ? AND heading = ?", row[0], row[1])
		if err != nil {
			doc, err = s.CreateDocument(ctx, qas.DocumentBasic{
				Title:   row[0],
				Heading: row[1],
				Content: row[2],
			})
		} else {
			err = s.UpdateDocument(ctx, doc.StringID(), qas.DocumentSet{
				Content: &row[2],
			})
		}
		if err != nil {
			return err
		}
	}
}
func dbBeforeSaveDocument(ctx context.Context, db ormDB, obj *qas.Document) error {
	if len(obj.Content) == 0 {
		return fmt.Errorf("empty content: %s,%s", obj.Title, obj.Heading)
	}
	if !obj.IsUpdate() || obj.HasChange("content") {
		text := fmt.Sprintf("%s %s: %s", obj.Title, obj.Heading, obj.Content)
		vec, err := GetEmbedding(ctx, text)
		if err != nil {
			return err
		}
		if len(vec) > 0 {
			obj.Embedding = vec
			if obj.IsUpdate() {
				obj.SetChange("embedding")
			}
		}
	}
	return nil
}

func GetEmbedding(ctx context.Context, text string) (vec qas.Vector, err error) {
	if len(text) == 0 {
		err = ErrEmptyParam
		return
	}
	oc := NewOpenAIClient()
	res, err := oc.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.AdaEmbeddingV2,
	})
	if err != nil {
		logger().Infow("embedding fail", "text", text, "err", err)
		return
	}
	logger().Infow("embedding res", "text", text, "usage", &res.Usage)
	if len(res.Data) > 0 {
		vec = qas.Vector(res.Data[0].Embedding)
	}
	return
}

const (
	Separator  = "\n* "
	threshold  = 0.52
	matchCount = 4
)

var (
	replPrompt = strings.NewReplacer("\n", " ")
)

func (s *qaStore) ConstructPrompt(ctx context.Context, question string) (prompt string, err error) {
	var docs qas.Documents
	docs, err = s.MatchDocments(ctx, question)
	if err != nil {
		return
	}
	var sections []string
	for _, doc := range docs {
		logger().Infow("hit", "id", doc.ID, "title", doc.Title, "heading", doc.Heading, "sim", doc.Similarity)
		text := fmt.Sprintf("%s%s: %s", Separator, doc.Heading, replPrompt.Replace(doc.Content))
		sections = append(sections, text)
	}

	prompt = strings.Join(sections, "") + "\n\n Q: " + question + "\n A:"

	return
}
func (s *qaStore) MatchDocments(ctx context.Context, question string) (data qas.Documents, err error) {
	var vec qas.Vector
	vec, err = GetEmbedding(ctx, question)
	if err != nil {
		return
	}
	data, err = s.MatchDocmentsWithVector(ctx, vec)
	return
}
func (s *qaStore) MatchDocmentsWithVector(ctx context.Context, vec qas.Vector) (data qas.Documents, err error) {
	if len(vec) != qas.VectorLen {
		return
	}
	logger().Debugw("embedding", "vec", vec[0:5])
	err = s.w.db.NewRaw("SELECT * FROM qa_match_documents(?, ?, ?)", vec, threshold, matchCount).Scan(ctx, &data)
	if err != nil {
		logger().Infow("query fail", "err", err)
	}
	return
}
