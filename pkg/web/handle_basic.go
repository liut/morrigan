package web

import (
	"net/http"

	"github.com/liut/morrigan/pkg/services/stores"
	"github.com/liut/morrigan/pkg/web/resp"
)

var handles = []handleIn{}

type haFunc func(a *server) http.HandlerFunc

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

func authPerm(permID string) func(next http.Handler) http.Handler {
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

// nolint
func handleNoContent(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(204)
}

// @Summary API health check
// @Description API health check
// @Produce plain
// @Success 200 {string} pong
// @Router /api/ping [get]
func handlerPing(w http.ResponseWriter, r *http.Request) {
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
