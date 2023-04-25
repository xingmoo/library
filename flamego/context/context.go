package context

import (
	"errors"
	"github.com/flamego/flamego"
	"github.com/mileusna/useragent"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
)

const (
	viewDataContextKey = "__vdata__"
	contextKey         = "__xingmo_http_context__"
)

type Views interface {
	Load() error
	ReanderBytes(string, any) ([]byte, error)
	Render(io.Writer, string, any) error
}

type ViewData map[string]any

type Context struct {
	flamego.Context
	flamego.Render
	logger    *zap.Logger
	views     Views
	vdata     ViewData
	userAgent *useragent.UserAgent
}

type Options struct {
	Views       Views
	ContentType string
	Logger      *zap.Logger
}

func Contexter(opts ...Options) flamego.Handler {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}

	parseRenderOptions := func(opts Options) Options {
		if opts.ContentType == "" {
			opts.ContentType = "text/html; charset=utf-8"
		}
		return opts
	}

	opt = parseRenderOptions(opt)

	return func(ctx flamego.Context, render flamego.Render) {

		c := &Context{
			Context: ctx,
			Render:  render,
			views:   opt.Views,
			logger:  opt.Logger,
		}
		c.vdata = ViewData{"ctx": c}

		ctx.Map(c)
	}

}

func (c *Context) UserAgent() *useragent.UserAgent {
	if c.userAgent == nil {
		ua := useragent.Parse(c.Request().UserAgent())
		c.userAgent = &ua
	}

	return c.userAgent
}

func (c *Context) IsPost() bool {
	return c.Request().Method == http.MethodPost
}

func (c *Context) IsGet() bool {
	return c.Request().Method == http.MethodGet
}

func (c *Context) IsPut() bool {
	return c.Request().Method == http.MethodPut
}

func (c *Context) IsDelete() bool {
	return c.Request().Method == http.MethodDelete
}

func (c *Context) IsPatch() bool {
	return c.Request().Method == http.MethodPatch
}

func (c *Context) IsHead() bool {
	return c.Request().Method == http.MethodHead
}

func (c *Context) IsOptions() bool {
	return c.Request().Method == http.MethodOptions
}

func (c *Context) IsAjax() bool {
	return c.Request().Header.Get("X-Requested-With") == "XMLHttpRequest"
}

func (c *Context) IsPjax() bool {
	return c.Request().Header.Get("X-PJAX") == "true"

}

func (c *Context) IsMobile() bool {
	return c.UserAgent().Mobile
}

func (c *Context) IsTablet() bool {
	return c.UserAgent().Tablet
}

func (c *Context) IsDesktop() bool {
	return !c.UserAgent().Mobile && !c.UserAgent().Tablet
}

// IsWechat 判断是不是微信
func (c *Context) IsWechat() bool {
	var ua = strings.ToLower(c.UserAgent().String)
	return strings.Contains(ua, "micromessenger")
}

// IsAlipay 判断是不是支付宝
func (c *Context) IsAlipay() bool {
	var ua = strings.ToLower(c.UserAgent().String)
	return strings.Contains(ua, "alipayclient")
}

func (c *Context) ViewData(key string, val any) {
	c.vdata[key] = val
}

func (c *Context) Views() Views {
	return c.views
}

func (c *Context) HtmlByte(tpl string, data ...ViewData) ([]byte, error) {
	if c.views == nil {
		return nil, errors.New("not support template")
	}

	for _, v := range data {
		for k, val := range v {
			c.vdata[k] = val
		}
	}

	return c.views.ReanderBytes(tpl, map[string]any(c.vdata))
}

// Html 渲染模板
func (c *Context) Html(status int, tpl string, data ...ViewData) {
	buf, err := c.HtmlByte(tpl, data...)
	if err != nil {
		c.logger.Error("render template error", zap.Error(err))
		if flamego.Env() == flamego.EnvTypeDev {
			http.Error(c.ResponseWriter(), err.Error(), http.StatusInternalServerError)
		} else {
			http.Error(c.ResponseWriter(), http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	c.ResponseWriter().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.ResponseWriter().WriteHeader(status)
	_, err = c.ResponseWriter().Write(buf)
	if err != nil {
		c.logger.Error("write response error", zap.Error(err))
	}
}
