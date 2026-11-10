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
	"fmt"
	"log/slog"
	"time"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/middleware/selector"
	"github.com/yylego/kratos-auth/authkratos"
)

// Config holds the routes that get the fast timeout and the timeout value
// Config 保存要用快速超时的路由范围和超时时间
type Config struct {
	routeScope *authkratos.RouteScope // Routes that get the fast timeout // 使用快速超时的路由范围
	newTimeout time.Duration          // The fast timeout // 快速超时的时间
}

// NewConfig creates a Config with the route scope and the fast timeout, which must be above zero
// NewConfig 用路由范围和快速超时时间创建配置，超时时间必须大于零
func NewConfig(routeScope *authkratos.RouteScope, newTimeout time.Duration) *Config {
	return &Config{
		routeScope: routeScope,
		newTimeout: newTimeout,
	}
}

// NewMiddleware creates middleware with shorter timeout on specific routes
// In practice extending timeout is more common than shortening
// Since ctx timeout can just shorten not extend, use exclusion filtering approach:
// Set long timeout on entire service, then limit other routes with shorter timeouts
// Use EXCLUDE mode to exclude routes needing long timeout, others get fast timeout
// This satisfies the "extend timeout" requirement
// Panics when the fast timeout is not above zero, since each matched request would then time out at once
// Each match and timeout outcome goes to applog at debug level, so the level of applog decides if it shows
//
// NewMiddleware 这个函数得到个middleware让某些接口具有更短的超时时间
// 但现实中我们遇到的问题往往是需要延长某个接口的超时时间
// 这样"设置更长超时时间"的需求更常见，以下是解决的思路
// 由于 ctx 的超时时间只能缩短而不能延长，因此整个设计是用"排除法过滤"
// 就是先给整个服务的接口配置很长的超时时间，再限制其余接口的超时时间为更短的时间
// 配置时使用 "EXCLUDE" 排除这些接口，其它的都是快速超时的
// 即可满足"设置更长超时时间"的需求
// 快速超时不大于零时 panic，否则匹配到的请求会全部立刻超时
// 每次匹配和超时结果以 debug 级别写进 applog，显示与否由 applog 的日志级别决定
func NewMiddleware(cfg *Config, applog *slog.Logger) middleware.Middleware {
	if cfg.newTimeout <= 0 {
		panic(fmt.Errorf("fast-kratos-timeout: new-timeout must be above zero, got %v", cfg.newTimeout))
	}
	applog.Info(
		"fast-kratos-timeout: new middleware",
		"side", cfg.routeScope.Side,
		"operations", len(cfg.routeScope.OperationSet),
		"new-timeout", cfg.newTimeout,
	)
	applog.Debug("fast-kratos-timeout: new middleware route-scope", "route-scope", cfg.routeScope)
	return selector.Server(middlewareFunc(cfg, applog)).Match(matchFunc(cfg, applog)).Build()
}

func matchFunc(cfg *Config, applog *slog.Logger) selector.MatchFunc {
	return func(ctx context.Context, operation string) bool {
		match := cfg.routeScope.Match(operation)
		applog.Debug("fast-kratos-timeout: route match", "operation", operation, "side", cfg.routeScope.Side, "match", match)
		return match
	}
}

func middlewareFunc(cfg *Config, applog *slog.Logger) middleware.Middleware {
	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			// The context keeps the earliest deadline, so the new timeout can shorten the deadline but not extend it
			// context 取所有超时里最早的那个，所以新超时只能缩短、不能延长
			ctx, cancel := context.WithTimeout(ctx, cfg.newTimeout)
			defer cancel()
			applog.Debug("fast-kratos-timeout: context with new-timeout", "new-timeout", cfg.newTimeout)
			return handleFunc(ctx, req)
		}
	}
}
