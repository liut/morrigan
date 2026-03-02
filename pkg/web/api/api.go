//go:build api

// TODO: migrate all api handelers from pkg/web/handle*

package apim1

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/liut/morrigan/pkg/services/stores"
	"github.com/liut/morrigan/pkg/web/resp"
	"github.com/liut/morrigan/pkg/web/routes"
)

var handles = []handleIn{}

type haFunc func(a *api) http.HandlerFunc

type handleIn struct {
	auth   bool
	method string
	path   string
	rid    string
	hafn   haFunc
}

func regHI(auth bool, method string, path string, rid string, hafn haFunc) {
	handles = append(handles, handleIn{auth, method, path, rid, hafn})
}

// nolint
type api struct {
	sto stores.Storage
}

// init 注册到 routes
func init() {
	routes.Register("api", routes.StrapFunc(strap))
}

func strap(r chi.Router) {
	a := newapi(stores.Sgt())
	a.Strap(r)
}

func newapi(sto stores.Storage) *api {
	return &api{sto: sto}
}

// Strap 注册路由到 chi.Router
func (a *api) Strap(r chi.Router) {
	r.Route("/api", func(r chi.Router) {
		r.Get("/ping", ping)
		// 遍历 handles 注册路由
		pr := r.With(routes.AuthMw(false))
		for _, hi := range handles {
			if hi.auth {
				if len(hi.rid) > 0 {
					pr.With(a.authPerm(hi.rid)).Method(hi.method, hi.path, hi.hafn(a))
				} else {
					pr.Method(hi.method, hi.path, hi.hafn(a))
				}
			} else {
				r.Method(hi.method, hi.path, hi.hafn(a))
			}
		}
	})
}

func (a *api) authPerm(permID string) func(next http.Handler) http.Handler {
	// TODO:
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !stores.IsKeeper(r.Context()) {
				w.WriteHeader(403)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// @Summary API health check
// @Description API health check
// @Produce plain
// @Success 200 {string} pong
// @Router /api/m1/ping [get]
func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong")) //nolint
}

// success 成功响应
func success(w http.ResponseWriter, r *http.Request, result interface{}) {
	resp.Ok(w, r, result)
}

// fail 失败响应
func fail(w http.ResponseWriter, r *http.Request, code int, args ...interface{}) {
	resp.Fail(w, r, code, args...)
}

// dtResult 构建分页结果
func dtResult(data any, total int) *resp.ResultData {
	return &resp.ResultData{
		Data:  data,
		Total: total,
	}
}
