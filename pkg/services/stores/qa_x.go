package stores

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	openai "github.com/sashabaranov/go-openai"

	"github.com/liut/morrigan/pkg/models/qas"
	"github.com/liut/morrigan/pkg/settings"
)

const (
	Separator    = "\n* "
	AnswerStop   = " END"
	dftThreshold = 0.43
	dftLimit     = 4

	tplKeyword  = "总结下面的文字内容，提炼出关键字句，如果是疑问句，则忽略问话的形式，只罗列出重点关键词，去除疑问形式，不考虑疑问表达，也不要返回多余内容，只关注最重要的词语，例如如果文字内容是问“什么”“为什么”“有什么”“怎么样”等等类似的语句，这些问话形式一律忽略，只返回关键字句，如果关键字句不成语句，则以关键字列表的形式返回，且用空格分隔，仅占一行，不要多行:\n\n%s\n\n"
	tplQaCtx    = "根据以下文本编写尽可能多一些的问题及回答:  \n\n文本:\n%s\n\n"
	maxQaTokens = 1024
)

var (
	ErrEmptyParam = errors.New("empty param")

	qaHeads = []string{"title", "heading", "content"}

	replPrompt = strings.NewReplacer("\n", " ")
	replText   = strings.NewReplacer("\u2028", "\n")
)

type MatchSpec struct {
	Question     string
	Threshold    float32
	Limit        int
	SkipKeywords bool
}

func (ms *MatchSpec) setDefaults() {
	if ms.Threshold == 0 {
		ms.Threshold = dftThreshold
	}
	if ms.Limit == 0 {
		ms.Limit = dftLimit
	}
}

type ExportArg struct {
	Spec   *QaDocumentSpec
	Out    io.Writer
	Format string // csv,jsonl
}

func validHead(rec []string) bool {
	return len(rec) >= len(qaHeads) && rec[0] == qaHeads[0] && rec[1] == qaHeads[1] && rec[2] == qaHeads[2]
}

type qaStoreX interface {
	ImportDocs(ctx context.Context, r io.Reader, lw io.Writer) error
	ExportDocs(ctx context.Context, ea ExportArg) error
	EmbeddingDocVector(ctx context.Context, spec *QaDocumentSpec) error
	ConstructPrompt(ctx context.Context, ms MatchSpec) (prompt string, err error)
	MatchDocments(ctx context.Context, ms MatchSpec) (data qas.Documents, err error)
	MatchVectorWith(ctx context.Context, vec qas.Vector, threshold float32, limit int) (data qas.DocMatches, err error)
}

func (s *qaStore) ImportDocs(ctx context.Context, r io.Reader, lw io.Writer) error {
	rd := csv.NewReader(r)
	rec, err := rd.Read()
	if err != nil {
		return err
	}
	if !validHead(rec) {
		return fmt.Errorf("invalid csv head: %+v", rec)
	}

	var idx int
	var valid int
	for {
		row, err := rd.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				logger().Infow("import docs", "lines", idx, "valid", valid)
				return nil
			}
			return err
		}
		idx++
		if len(row) < 3 || len(row[0]) == 0 || len(row[1]) == 0 || len(row[2]) == 0 {
			logger().Infow("empty row, skip", "idx", idx, "row", row)
			// return fmt.Errorf("invalid csv row #%d: %+v", idx, row)
			continue
		}
		err = s.importLine(ctx, qas.DocumentBasic{
			Title:   row[0],
			Heading: row[1],
			Content: row[2],
		}, lw)
		if err != nil {
			return err
		}
		valid++
	}
}

func (s *qaStore) importLine(ctx context.Context, basic qas.DocumentBasic, lw io.Writer) error {
	doc := new(qas.Document)
	basic.Content = replText.Replace(basic.Content)
	err := dbGet(ctx, s.w.db, doc, "title = ? AND heading = ?", basic.Title, basic.Heading)
	if err != nil {
		doc, err = s.CreateDocument(ctx, basic)
	} else {
		if doc.Content != basic.Content {
			logger().Infow("updating", "id", doc.ID, "title", basic.Title, "heading", basic.Heading)
			dif := diff2(doc.Content, basic.Content)
			if lw != nil {
				if _, werr := fmt.Fprintf(lw, "Doc ID: %s, Title: %s, Heading: %s\nDiff:\n%s\n\n",
					doc.StringID(), basic.Title, basic.Heading, dif); werr != nil {
					logger().Infow("write diff fail", "lw", lw, "err", werr)
				}
			}

		}
		err = s.UpdateDocument(ctx, doc.StringID(), qas.DocumentSet{
			Content: &basic.Content,
		})
	}
	if err != nil {
		logger().Infow("save document fail", "id", doc.ID, "err", err)
		return err
	}
	return nil
}

func diff2(text1, text2 string) string {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(text1),
		B:        difflib.SplitLines(text2),
		FromFile: "Original",
		ToFile:   "Current",
		Context:  3,
	}
	text, _ := difflib.GetUnifiedDiffString(diff)
	return text
}

func (s *qaStore) afterCreatedQaDocument(ctx context.Context, obj *qas.Document) error {
	dvb := qas.DocVectorBasic{
		DocID:   obj.ID,
		Subject: obj.GetSubject(),
	}
	vec, err := GetEmbedding(ctx, dvb.Subject)
	if err != nil {
		return err
	}
	if len(vec) > 0 {
		dvb.Vector = vec
	}

	_, err = s.CreateDocVector(ctx, dvb)
	if err != nil {
		logger().Infow("create doc vector fail", "dvb", &dvb, "err", err)
		return err
	}
	return nil
}

func GetEmbedding(ctx context.Context, text string) (vec qas.Vector, err error) {
	if len(text) == 0 {
		err = ErrEmptyParam
		return
	}

	res, err := ocEm.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(settings.Current.Embedding.Model),
	})
	if err != nil {
		logger().Infow("embedding fail", "text", text, "err", err)
		return
	}
	if len(res.Data) > 0 {
		vec = qas.Vector(res.Data[0].Embedding)
		logger().Infow("embedding res", "text", text, "vec", len(vec), "usage", &res.Usage)
	} else {
		logger().Infow("embedding result is empty", "text", text)
	}
	return
}

func GetKeywords(ctx context.Context, text string) (kw string, err error) {
	if len(text) == 0 {
		err = ErrEmptyParam
		return
	}
	res, err := ocSu.CreateCompletion(ctx, openai.CompletionRequest{
		Model:  settings.Current.Summarize.Model,
		Prompt: fmt.Sprintf(tplKeyword, text),
	})
	if err != nil {
		logger().Infow("summarize fail", "text", text, "err", err)
		return
	}
	if len(res.Choices) > 0 {
		kw = res.Choices[0].Text
		logger().Infow("summarize ok", "text", text, "kw", kw, "usage", &res.Usage)
	} else {
		logger().Infow("summarize result is empty", "text", text, "res", res)
	}
	return
}

func (s *qaStore) ConstructPrompt(ctx context.Context, ms MatchSpec) (prompt string, err error) {
	var docs qas.Documents
	docs, err = s.MatchDocments(ctx, ms)
	if err != nil {
		return
	}
	var sections []string
	for _, doc := range docs {
		text := fmt.Sprintf("%s%s: %s", Separator, doc.Heading, replPrompt.Replace(doc.Content))
		sections = append(sections, text)
	}

	prompt = fmt.Sprintf("%s\n\n%s %s\n%s",
		strings.Join(sections, ""), qas.PrefixQ, ms.Question, qas.PrefixA)

	return
}
func (s *qaStore) MatchDocments(ctx context.Context, ms MatchSpec) (data qas.Documents, err error) {
	ms.setDefaults()
	var subject string
	if ms.SkipKeywords {
		subject = ms.Question
	} else {
		subject, err = GetKeywords(ctx, ms.Question)
		if err != nil {
			return
		}
	}

	if len(subject) == 0 {
		logger().Infow("empty subject", "spec", ms)
		return
	}
	var vec qas.Vector
	vec, err = GetEmbedding(ctx, subject)
	if err != nil {
		logger().Infow("GetEmbedding fail", "err", err)
		return
	}
	var ps qas.DocMatches
	ps, err = s.MatchVectorWith(ctx, vec, ms.Threshold, ms.Limit)
	logger().Infow("matching", "docs", len(ps), "err", err)
	if err != nil || len(ps) == 0 {
		logger().Infow("no match docs", "subj", subject)
		return
	}
	spec := &QaDocumentSpec{}
	spec.IDs = ps.DocumentIDs()
	err = queryList(ctx, s.w.db, spec, &data).Scan(ctx)
	if err != nil {
		logger().Infow("list docs fail", "spec", spec, "err", err)
	} else {
		logger().Infow("list docs", "matches", len(data))
	}
	return
}

func (s *qaStore) MatchVectorWith(ctx context.Context, vec qas.Vector, threshold float32, limit int) (data qas.DocMatches, err error) {
	if len(vec) != qas.VectorLen {
		logger().Infow("mismatch length of vector", "a", len(vec), "b", qas.VectorLen)
		return
	}
	logger().Debugw("match with", "vec", vec[0:5])
	err = s.w.db.NewRaw("SELECT * FROM qa_match_docs_4(?, ?, ?)", vec, threshold, limit).
		Scan(ctx, &data)
	// err = s.w.db.NewSelect().
	// 	Table(qas.DocVectorTable).
	// 	Column("doc_id", "subject").
	// 	ColumnExpr("(embedding <=> ?) as similarity", vec).
	// 	Where("(embedding <=> ?) < ?", vec, threshold).
	// 	OrderExpr("embedding <=> ?", vec).
	// 	Limit(limit).Scan(ctx, &data)
	if err != nil {
		logger().Infow("match vector fail", "threshold", threshold, "limit", limit, "err", err)
	} else {
		logger().Infow("match vector ok", "threshold", threshold, "limit", limit, "data", data)
	}
	return
}

func (s *qaStore) ExportDocs(ctx context.Context, ea ExportArg) error {
	data, _, err := s.ListDocument(ctx, ea.Spec)
	if err != nil {
		return err
	}

	if ea.Format == "csv" {
		return documentsToCSV(data, ea.Out)
	}

	// TODO: jsonl?
	return errors.New("invalid format: " + ea.Format)
}

func documentsToCSV(data qas.Documents, w io.Writer) error {

	head := []string{"doc_id", "title", "heading", "content"}
	cw := csv.NewWriter(w)
	if err := cw.Write(head); err != nil {
		return err
	}

	for _, doc := range data {
		if err := cw.Write([]string{
			doc.StringID(), doc.Title, doc.Heading, doc.Content,
		}); err != nil {
			return err
		}
	}
	cw.Flush()

	return cw.Error()
}

func (s *qaStore) EmbeddingDocVector(ctx context.Context, spec *QaDocumentSpec) error {
	data, _, err := s.ListDocument(ctx, spec)
	if err != nil {
		return err
	}

	for _, doc := range data {
		subject := doc.GetSubject()
		vec, err := GetEmbedding(ctx, subject)
		if err != nil {
			return err
		}
		exist := new(qas.DocVector)
		err = dbGetWithUnique(ctx, s.w.db, exist, "doc_id", doc.ID)
		if err == nil {
			exist.SetWith(qas.DocVectorSet{
				Subject: &subject,
				Vector:  &vec,
			})
			if err = dbUpdate(ctx, s.w.db, exist); err != nil {
				return err
			}
		} else {
			dv := qas.NewDocVectorWithBasic(qas.DocVectorBasic{
				DocID:   doc.ID,
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

func dbAfterDeleteQaDocument(ctx context.Context, db ormDB, obj *qas.Document) error {
	_, err := dbBatchDeleteWithKeyID(ctx, db, qas.DocVectorTable, "doc_id", obj.ID)
	return err
}
