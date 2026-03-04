// This file is generated - Do Not Edit.

package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/liut/morrigan/pkg/models/convo"
	"github.com/liut/morrigan/pkg/services/stores"
)

func init() {
	regHI(true, "GET", "/convo/sessions", "", func(a *api) http.HandlerFunc {
		return a.getConvoSessions
	})
	regHI(true, "GET", "/convo/sessions/:id", "", func(a *api) http.HandlerFunc {
		return a.getConvoSession
	})
	regHI(true, "DELETE", "/convo/sessions/:id", "convo-sessions-id-delete", func(a *api) http.HandlerFunc {
		return a.deleteConvoSession
	})
	regHI(true, "GET", "/convo/messages", "", func(a *api) http.HandlerFunc {
		return a.getConvoMessages
	})
	regHI(true, "GET", "/convo/messages/:id", "", func(a *api) http.HandlerFunc {
		return a.getConvoMessage
	})
	regHI(true, "DELETE", "/convo/messages/:id", "convo-messages-id-delete", func(a *api) http.HandlerFunc {
		return a.deleteConvoMessage
	})
}

// @Tags 默认 文档生成
// @Summary 列出会话
// @Accept json
// @Produce json
// @Param token    header   string  true "登录票据凭证"
// @Param   query  query   stores.ConvoSessionSpec  true   "Object"
// @Success 200 {object} Done{result=ResultData{data=convo.Sessions}}
// @Failure 400 {object} Failure "请求或参数错误"
// @Failure 401 {object} Failure "未登录"
// @Failure 404 {object} Failure "目标未找到"
// @Failure 503 {object} Failure "服务端错误"
// @Router /api/convo/sessions [get]
func (a *api) getConvoSessions(w http.ResponseWriter, r *http.Request) {
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
// @Param token    header   string  true "登录票据凭证"
// @Param   id    path   string  true   "编号"
// @Success 200 {object} Done{result=convo.Session}
// @Failure 400 {object} Failure "请求或参数错误"
// @Failure 401 {object} Failure "未登录"
// @Failure 404 {object} Failure "目标未找到"
// @Failure 503 {object} Failure "服务端错误"
// @Router /api/convo/sessions/{id} [get]
func (a *api) getConvoSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var obj *convo.Session
	var err error
	obj, err = a.sto.Convo().GetSession(r.Context(), id)
	if err != nil {
		fail(w, r, 503, err)
		return
	}

	success(w, r, obj)
}

// @Tags 默认 文档生成
// @ID convo-sessions-id-delete
// @Summary 删除会话 🔑
// @Accept json
// @Produce json
// @Param token    header   string  true "登录票据凭证"
// @Param   id    path   string  true   "编号"
// @Success 200 {object} Done
// @Failure 400 {object} Failure "请求或参数错误"
// @Failure 401 {object} Failure "未登录"
// @Failure 403 {object} Failure "无权限"
// @Failure 503 {object} Failure "服务端错误"
// @Router /api/convo/sessions/{id} [delete]
func (a *api) deleteConvoSession(w http.ResponseWriter, r *http.Request) {
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
// @Param token    header   string  true "登录票据凭证"
// @Param   query  query   stores.ConvoMessageSpec  true   "Object"
// @Success 200 {object} Done{result=ResultData{data=convo.Messages}}
// @Failure 400 {object} Failure "请求或参数错误"
// @Failure 401 {object} Failure "未登录"
// @Failure 404 {object} Failure "目标未找到"
// @Failure 503 {object} Failure "服务端错误"
// @Router /api/convo/messages [get]
func (a *api) getConvoMessages(w http.ResponseWriter, r *http.Request) {
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
// @Param token    header   string  true "登录票据凭证"
// @Param   id    path   string  true   "编号"
// @Success 200 {object} Done{result=convo.Message}
// @Failure 400 {object} Failure "请求或参数错误"
// @Failure 401 {object} Failure "未登录"
// @Failure 404 {object} Failure "目标未找到"
// @Failure 503 {object} Failure "服务端错误"
// @Router /api/convo/messages/{id} [get]
func (a *api) getConvoMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var obj *convo.Message
	var err error
	obj, err = a.sto.Convo().GetMessage(r.Context(), id)
	if err != nil {
		fail(w, r, 503, err)
		return
	}

	success(w, r, obj)
}

// @Tags 默认 文档生成
// @ID convo-messages-id-delete
// @Summary 删除会话 🔑
// @Accept json
// @Produce json
// @Param token    header   string  true "登录票据凭证"
// @Param   id    path   string  true   "编号"
// @Success 200 {object} Done
// @Failure 400 {object} Failure "请求或参数错误"
// @Failure 401 {object} Failure "未登录"
// @Failure 403 {object} Failure "无权限"
// @Failure 503 {object} Failure "服务端错误"
// @Router /api/convo/messages/{id} [delete]
func (a *api) deleteConvoMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := a.sto.Convo().DeleteMessage(r.Context(), id)
	if err != nil {
		fail(w, r, 503, err)
		return
	}

	success(w, r, "ok")
}
