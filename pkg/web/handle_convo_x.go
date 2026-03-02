package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	querybinder "github.com/wgarunap/url-query-binder"

	"github.com/liut/morrigan/pkg/services/stores"
)

var queryBinder = querybinder.NewQueryBinder()

// init 注册路由处理函数
func init() {
	regHI(false, "GET", "/convo/sessions", "", func(a *server) http.HandlerFunc {
		return a.getConvoSessions
	})
	regHI(false, "GET", "/convo/sessions/:id", "", func(a *server) http.HandlerFunc {
		return a.getConvoSession
	})
	regHI(true, "DELETE", "/convo/sessions/:id", "m1-convo-sessions-id-delete", func(a *server) http.HandlerFunc {
		return a.deleteConvoSession
	})
	regHI(false, "GET", "/convo/messages", "", func(a *server) http.HandlerFunc {
		return a.getConvoMessages
	})
	regHI(false, "GET", "/convo/messages/:id", "", func(a *server) http.HandlerFunc {
		return a.getConvoMessage
	})
	regHI(true, "DELETE", "/convo/messages/:id", "m1-convo-messages-id-delete", func(a *server) http.HandlerFunc {
		return a.deleteConvoMessage
	})
}

// @Tags 默认 文档生成
// @Summary 列出会话
// @Accept json
// @Produce json
// @Param query query stores.ConvoSessionSpec true "查询条件"
// @Success 200 {object} Done{result=ResultData{data=convo.Sessions}}
// @Failure 400 {object} Failure "请求或参数错误"
// @Failure 503 {object} Failure "服务端错误"
// @Router /api/m1/convo/sessions [get]
func (a *server) getConvoSessions(w http.ResponseWriter, r *http.Request) {
	var spec stores.ConvoSessionSpec
	if err := queryBinder.Bind(&spec, r.URL); err != nil {
		fail(w, r, 400, err)
		return
	}

	ctx := r.Context()
	data, total, err := a.sto.Convo().ListSession(ctx, &spec)
	if err != nil {
		fail(w, r, 503, err)
		return
	}

	success(w, r, dtResult(data, total))
}

// @Tags 默认 文档生成
// @Summary 获取会话
// @Accept json
// @Produce json
// @Param id path string true "编号"
// @Success 200 {object} Done{result=convo.Session}
// @Failure 400 {object} Failure "请求或参数错误"
// @Failure 503 {object} Failure "服务端错误"
// @Router /api/m1/convo/sessions/{id} [get]
func (a *server) getConvoSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	obj, err := a.sto.Convo().GetSession(r.Context(), id)
	if err != nil {
		fail(w, r, 503, err)
		return
	}

	success(w, r, obj)
}

// @Tags 默认 文档生成
// @ID m1-convo-sessions-id-delete
// @Summary 删除会话
// @Accept json
// @Produce json
// @Param id path string true "编号"
// @Success 200 {object} Done
// @Failure 400 {object} Failure "请求或参数错误"
// @Failure 503 {object} Failure "服务端错误"
// @Router /api/m1/convo/sessions/{id} [delete]
func (a *server) deleteConvoSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := a.sto.Convo().DeleteSession(r.Context(), id)
	if err != nil {
		fail(w, r, 503, err)
		return
	}

	success(w, r, "ok")
}

// @Tags 默认 文档生成
// @Summary 列出会话
// @Accept json
// @Produce json
// @Param query query stores.ConvoMessageSpec true "查询条件"
// @Success 200 {object} Done{result=ResultData{data=convo.Messages}}
// @Failure 400 {object} Failure "请求或参数错误"
// @Failure 503 {object} Failure "服务端错误"
// @Router /api/m1/convo/messages [get]
func (a *server) getConvoMessages(w http.ResponseWriter, r *http.Request) {
	var spec stores.ConvoMessageSpec
	if err := queryBinder.Bind(&spec, r.URL); err != nil {
		fail(w, r, 400, err)
		return
	}

	ctx := r.Context()
	data, total, err := a.sto.Convo().ListMessage(ctx, &spec)
	if err != nil {
		fail(w, r, 503, err)
		return
	}

	success(w, r, dtResult(data, total))
}

// @Tags 默认 文档生成
// @Summary 获取会话
// @Accept json
// @Produce json
// @Param id path string true "编号"
// @Success 200 {object} Done{result=convo.Message}
// @Failure 400 {object} Failure "请求或参数错误"
// @Failure 503 {object} Failure "服务端错误"
// @Router /api/m1/convo/messages/{id} [get]
func (a *server) getConvoMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	obj, err := a.sto.Convo().GetMessage(r.Context(), id)
	if err != nil {
		fail(w, r, 503, err)
		return
	}

	success(w, r, obj)
}

// @Tags 默认 文档生成
// @ID m1-convo-messages-id-delete
// @Summary 删除会话
// @Accept json
// @Produce json
// @Param id path string true "编号"
// @Success 200 {object} Done
// @Failure 400 {object} Failure "请求或参数错误"
// @Failure 503 {object} Failure "服务端错误"
// @Router /api/m1/convo/messages/{id} [delete]
func (a *server) deleteConvoMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := a.sto.Convo().DeleteMessage(r.Context(), id)
	if err != nil {
		fail(w, r, 503, err)
		return
	}

	success(w, r, "ok")
}
