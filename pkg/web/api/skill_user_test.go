package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cupogo/andvari/models/oid"
	"github.com/go-chi/chi/v5"
	auth "github.com/liut/simpauth"

	"github.com/liut/morign/pkg/models/aigc"
	"github.com/liut/morign/pkg/models/skills"
	"github.com/liut/morign/pkg/services/stores"
)

type memSkillStore struct {
	byName map[string]*skills.Skill
	files  map[string]skills.Files
	seq    int
}

func (f *memSkillStore) ListVisibleMetadata(ctx context.Context, spec *stores.SkillSpec) (skills.Skills, int, error) {
	var data skills.Skills
	for _, obj := range f.byName {
		data = append(data, *obj)
	}
	return data, len(data), nil
}

func (f *memSkillStore) TopRecent(ctx context.Context, limit int) (skills.Skills, error) {
	return nil, nil
}

func (f *memSkillStore) LoadForName(ctx context.Context, name string) (*skills.Skill, error) {
	obj, ok := f.byName[name]
	if !ok {
		return nil, stores.ErrNotFound
	}
	return obj, nil
}

func (f *memSkillStore) ListSkill(ctx context.Context, spec *stores.SkillSpec) (skills.Skills, int, error) {
	return f.ListVisibleMetadata(ctx, spec)
}

func (f *memSkillStore) GetSkill(ctx context.Context, id string) (*skills.Skill, error) {
	if obj, ok := f.byName[id]; ok {
		return obj, nil
	}
	for _, obj := range f.byName {
		if obj.ID.String() == id {
			return obj, nil
		}
	}
	return nil, stores.ErrNotFound
}

func (f *memSkillStore) CreateSkill(ctx context.Context, in skills.SkillBasic) (*skills.Skill, error) {
	if _, ok := f.byName[in.Name]; ok {
		return nil, stores.ErrDuplicate
	}
	f.seq++
	obj := skills.NewSkillWithBasic(in)
	obj.SetID(oid.NewID(oid.OtFile))
	f.byName[in.Name] = obj
	return obj, nil
}

func (f *memSkillStore) CreateSkillWithFiles(ctx context.Context, in skills.SkillBasic, files map[string]string) (*skills.Skill, error) {
	obj, err := f.CreateSkill(ctx, in)
	if err != nil {
		return nil, err
	}
	f.setFiles(obj.Name, files)
	return obj, nil
}

func (f *memSkillStore) UpdateSkill(ctx context.Context, id string, in skills.SkillSet) error {
	for _, obj := range f.byName {
		if obj.ID.String() == id {
			if in.Description != nil {
				obj.Description = *in.Description
			}
			if in.Content != nil {
				obj.Content = *in.Content
			}
			return nil
		}
	}
	return stores.ErrNotFound
}

func (f *memSkillStore) DeleteSkill(ctx context.Context, id string) error {
	for name, obj := range f.byName {
		if obj.ID.String() == id {
			delete(f.byName, name)
			return nil
		}
	}
	return stores.ErrNotFound
}

func (f *memSkillStore) UpdateSkillWithFiles(ctx context.Context, id string, set skills.SkillSet, files map[string]string) error {
	for _, obj := range f.byName {
		if obj.ID.String() == id {
			if set.Description != nil {
				obj.Description = *set.Description
			}
			if set.Content != nil {
				obj.Content = *set.Content
			}
			if files != nil {
				f.setFiles(obj.Name, files)
			}
			return nil
		}
	}
	return stores.ErrNotFound
}

func (f *memSkillStore) ListFileNames(ctx context.Context, name string) (skills.Files, error) {
	if _, ok := f.byName[name]; !ok {
		return nil, stores.ErrNotFound
	}
	return f.files[name], nil
}

func (f *memSkillStore) ListFile(ctx context.Context, spec *stores.FileSpec) (skills.Files, int, error) {
	// 单测未走生成 ListFile 路径，文件访问均经 ListFileNames/ReadFile。
	return nil, 0, nil
}

func (f *memSkillStore) ReadFile(ctx context.Context, name, path string) (*skills.File, error) {
	for i := range f.files[name] {
		if f.files[name][i].Path == path {
			return &f.files[name][i], nil
		}
	}
	return nil, stores.ErrFileNotFound
}

func (f *memSkillStore) setFiles(name string, files map[string]string) {
	rows := make(skills.Files, 0, len(files))
	for path, content := range files {
		rows = append(rows, skills.File{FileBasic: skills.FileBasic{
			Path:    path,
			Content: []byte(content),
			Mime:    "text/plain",
			Kind:    skills.FileKindText,
			Size:    int64(len(content)),
		}})
	}
	f.files[name] = rows
}

type fakeStorage struct {
	skill *memSkillStore
}

func (f *fakeStorage) Preset() aigc.Preset                { return aigc.Preset{} }
func (f *fakeStorage) Corpus() stores.CorpuStore          { return nil }
func (f *fakeStorage) KB() stores.CorpuStore              { return nil }
func (f *fakeStorage) MCP() stores.MCPStore               { return nil }
func (f *fakeStorage) Convo() stores.ConvoStore           { return nil }
func (f *fakeStorage) State() stores.StateStore           { return nil }
func (f *fakeStorage) Capability() stores.CapabilityStore { return nil }
func (f *fakeStorage) Skill() stores.SkillStore           { return f.skill }

func skillAPI() (*api, *memSkillStore) {
	fk := &memSkillStore{byName: map[string]*skills.Skill{}, files: map[string]skills.Files{}}
	return &api{sto: &fakeStorage{skill: fk}}, fk
}

func skillRouter(a *api) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/api/skills", a.listSkills)
	r.Get("/api/skills/{name}", a.getSkillByName)
	r.Post("/api/skills", a.createSkill)
	r.Put("/api/skills/{name}", a.updateSkill)
	r.Delete("/api/skills/{name}", a.deleteSkillByName)
	return r
}

func withUser(req *http.Request, oidStr string) *http.Request {
	ctx := auth.ContextWithUser(req.Context(), &auth.User{OID: oidStr, UID: "u", Name: "u"})
	return req.WithContext(ctx)
}

func doSkillReq(r *chi.Mux, method, path, body, user string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req = withUser(req, user)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func respCode(t *testing.T, rr *httptest.ResponseRecorder) int {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode: %v (body %s)", err, rr.Body.String())
	}
	code, _ := m["status"].(float64)
	return int(code)
}

func TestSkillCreateAPI(t *testing.T) {
	a, fk := skillAPI()
	r := skillRouter(a)
	body := `{"name":"invoice","description":"开发票","content":"---\nname: invoice\ndescription: 开发票\n---\n\n正文"}`
	rr := doSkillReq(r, http.MethodPost, "/api/skills", body, "o1")
	if rr.Code != http.StatusOK {
		t.Fatalf("create status = %d, body %s", rr.Code, rr.Body.String())
	}
	obj := fk.byName["invoice"]
	if obj == nil || obj.Owner != oid.Cast("o1") {
		t.Errorf("owner not forced: %+v", obj)
	}
}

func TestSkillCreateAPIValidation(t *testing.T) {
	a, fk := skillAPI()
	r := skillRouter(a)
	cases := []struct {
		name string
		body string
		want int
	}{
		{"bad name", `{"name":"PDF","description":"d","content":"---\nname: PDF\ndescription: d\n---\nbody"}`, 400},
		{"missing frontmatter", `{"name":"invoice","description":"d","content":"plain"}`, 400},
		{"frontmatter mismatch", `{"name":"invoice","description":"d","content":"---\nname: other\ndescription: d\n---\nbody"}`, 400},
		{"long description", `{"name":"invoice","description":"` + strings.Repeat("a", 125) + `","content":"---\nname: invoice\ndescription: d\n---\nbody"}`, 400},
	}
	for _, c := range cases {
		rr := doSkillReq(r, http.MethodPost, "/api/skills", c.body, "o1")
		if got := respCode(t, rr); got != c.want {
			t.Errorf("%s: code = %d, want %d (body %s)", c.name, got, c.want, rr.Body.String())
		}
	}
	if len(fk.byName) != 0 {
		t.Errorf("no skill should be stored, got %v", fk.byName)
	}
}

func TestSkillCreateAPIDuplicate(t *testing.T) {
	a, fk := skillAPI()
	r := skillRouter(a)
	content := "---\nname: invoice\ndescription: d\n---\nbody"
	fk.byName["invoice"] = &skills.Skill{SkillBasic: skills.SkillBasic{
		Name: "invoice", Description: "d", Content: content, Owner: oid.Cast("o1"),
	}}
	body := `{"name":"invoice","description":"d","content":"---\nname: invoice\ndescription: d\n---\nbody"}`
	rr := doSkillReq(r, http.MethodPost, "/api/skills", body, "o1")
	if got := respCode(t, rr); got != http.StatusBadRequest {
		t.Errorf("duplicate code = %d, want 400", got)
	}
}

func TestSkillUpdateDeleteAPI(t *testing.T) {
	a, fk := skillAPI()
	r := skillRouter(a)
	content := "---\nname: invoice\ndescription: d\n---\nbody"
	fk.byName["invoice"] = &skills.Skill{SkillBasic: skills.SkillBasic{
		Name: "invoice", Description: "d", Content: content, Owner: oid.Cast("o1"),
	}}

	// 非 owner 更新 → 403
	rr := doSkillReq(r, http.MethodPut, "/api/skills/invoice", `{"description":"x"}`, "o2")
	if got := respCode(t, rr); got != http.StatusForbidden {
		t.Errorf("non-owner update code = %d, want 403", got)
	}

	// owner 更新 → 200
	rr = doSkillReq(r, http.MethodPut, "/api/skills/invoice", `{"description":"x"}`, "o1")
	if rr.Code != http.StatusOK {
		t.Errorf("owner update status = %d, body %s", rr.Code, rr.Body.String())
	}
	if fk.byName["invoice"].Description != "x" {
		t.Errorf("description not updated")
	}

	// owner 更新超长描述 → 400
	rr = doSkillReq(r, http.MethodPut, "/api/skills/invoice", `{"description":"`+strings.Repeat("a", 125)+`"}`, "o1")
	if got := respCode(t, rr); got != http.StatusBadRequest {
		t.Errorf("long description update code = %d, want 400", got)
	}

	// 非 owner 删除 → 403；owner 删除 → 200
	rr = doSkillReq(r, http.MethodDelete, "/api/skills/invoice", "", "o2")
	if got := respCode(t, rr); got != http.StatusForbidden {
		t.Errorf("non-owner delete code = %d, want 403", got)
	}
	rr = doSkillReq(r, http.MethodDelete, "/api/skills/invoice", "", "o1")
	if rr.Code != http.StatusOK {
		t.Errorf("owner delete status = %d, body %s", rr.Code, rr.Body.String())
	}
	if _, ok := fk.byName["invoice"]; ok {
		t.Error("skill should be deleted")
	}
}

func TestSkillGetAPI(t *testing.T) {
	a, fk := skillAPI()
	r := skillRouter(a)
	fk.byName["invoice"] = &skills.Skill{SkillBasic: skills.SkillBasic{
		Name: "invoice", Description: "d", Content: "body", Owner: oid.Cast("o1"),
	}}
	rr := doSkillReq(r, http.MethodGet, "/api/skills/invoice", "", "o1")
	if rr.Code != http.StatusOK {
		t.Fatalf("get status = %d", rr.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := resp["result"]; !ok {
		t.Errorf("unexpected response: %s", rr.Body.String())
	}

	rr = doSkillReq(r, http.MethodGet, "/api/skills/nope", "", "o1")
	if got := respCode(t, rr); got != http.StatusNotFound {
		t.Errorf("missing skill code = %d, want 404", got)
	}
}
