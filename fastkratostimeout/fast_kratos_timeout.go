// Package fastkratostimeout: Selective timeout override middleware with route-based control
// Provides fast timeout settings with flexible route scope configuration
// Enables shortening timeout on specific routes while maintaining defaults on others
// Good fit in mixed workload scenarios needing distinct timeout tactics
//
// fastkratostimeout: 选择性超时覆盖中间件，支持基于路由的控制
// 提供快速超时设置和灵活的路由范围配置
// 可在特定路由上缩短超时时间，同时在其他地方保持默认值
// 适用于需要不同超时策略的混合工作负载场景
package fastkratostimeout

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/middleware/selector"
	"github.com/yylego/kratos-auth/authkratos"
	"github.com/yylego/neatjson/neatjsons"
)

type Config struct {
	routeScope *authkratos.RouteScope
	newTimeout time.Duration                // 快速超时的时间
	spanHooks  []authkratos.NewSpanHookFunc // span 钩子回调函数列表
	debugMode  bool
}

func NewConfig(routeScope *authkratos.RouteScope, newTimeout time.Duration) *Config {
	return &Config{
		routeScope: routeScope,
		newTimeout: newTimeout,
		spanHooks:  nil,
		debugMode:  false,
	}
}

func (c *Config) WithDebugMode(debugMode bool) *Config {
	c.debugMode = debugMode
	return c
}

// WithNewSpanHook appends a span hook callback function for tracing
//
// WithNewSpanHook 追加一个追踪 span 钩子的回调函数
func (c *Config) WithNewSpanHook(fn authkratos.NewSpanHookFunc) *Config {
	c.spanHooks = append(c.spanHooks, fn)
	return c
}

// NewMiddleware creates middleware with shorter timeout on specific routes
// In practice extending timeout is more common than shortening
// Since ctx timeout can just shorten not extend, use exclusion filtering approach:
// Set long timeout on entire service, then limit other routes with shorter timeouts
// Use EXCLUDE mode to exclude routes needing long timeout, others get fast timeout
// This satisfies the "extend timeout" requirement
//
// NewMiddleware 这个函数得到个middleware让某些接口具有更短的超时时间
// 但现实中我们遇到的问题往往是需要延长某个接口的超时时间
// 这样"设置更长超时时间"的需求更常见，以下是解决的思路
// 由于 ctx 的超时时间只能缩短而不能延长，因此整个设计是用"排除法过滤"
// 就是先给整个服务的接口配置很长的超时时间，再限制其余接口的超时时间为更短的时间
// 配置时使用 "EXCLUDE" 排除这些接口，其它的都是快速超时的
// 即可满足"设置更长超时时间"的需求
func NewMiddleware(cfg *Config, applog *slog.Logger) middleware.Middleware {
	applog.Info(
		"fast-kratos-timeout: new middleware",
		"side", cfg.routeScope.Side,
		"operations", len(cfg.routeScope.OperationSet),
		"new-timeout", cfg.newTimeout,
		"debug-mode", authkratos.BooleanToNum(cfg.debugMode),
	)
	if cfg.debugMode {
		applog.Debug("fast-kratos-timeout: new middleware route-scope", "route-scope", neatjsons.S(cfg.routeScope))
	}
	return selector.Server(middlewareFunc(cfg, applog)).Match(matchFunc(cfg, applog)).Build()
}

func matchFunc(cfg *Config, applog *slog.Logger) selector.MatchFunc {
	return func(ctx context.Context, operation string) bool {
		defer authkratos.RunSpanHooks(ctx, cfg.spanHooks, "fast-kratos-timeout-match")()

		match := cfg.routeScope.Match(operation)
		if cfg.debugMode {
			if match {
				applog.Debug("fast-kratos-timeout: match next -> fast-timeout", "operation", operation, "side", cfg.routeScope.Side, "match", authkratos.BooleanToNum(match))
			} else {
				applog.Debug("fast-kratos-timeout: match skip -- slow-timeout", "operation", operation, "side", cfg.routeScope.Side, "match", authkratos.BooleanToNum(match))
			}
		}
		return match
	}
}

func middlewareFunc(cfg *Config, applog *slog.Logger) middleware.Middleware {
	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			defer authkratos.RunSpanHooks(ctx, cfg.spanHooks, "fast-kratos-timeout")()

			// 设置新超时时间，由于 ctx 是所有超时时间里取最短的
			// 因此只能缩短而不能延长，因此需要选择快速超时的
			ctx, can := context.WithTimeout(ctx, cfg.newTimeout)
			defer can()
			if cfg.debugMode {
				applog.Debug("fast-kratos-timeout: context with new-timeout", "new-timeout", cfg.newTimeout)
			}
			return handleFunc(ctx, req)
		}
	}
}
