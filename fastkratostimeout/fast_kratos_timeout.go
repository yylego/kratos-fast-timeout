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

// NewMiddleware creates the timeout middleware: routes inside the scope get the fast timeout, routes outside the scope keep the service timeout
// A context timeout can be shortened but not extended, so "give a few slow routes more time" is set up in reverse:
// set the service timeout to the long time the slow routes need, then leave the slow routes out with authkratos.NewExclude, and each remaining route gets the fast timeout
// An upstream deadline that is tight to begin with stays as it is
// Panics when the fast timeout is not above zero, since each matched request would then time out at once
// Each match and each timeout applied goes to applog at debug level, so the level of applog decides if it shows
//
// NewMiddleware 创建超时中间件：范围内的路由用快速超时，范围外的路由保持服务整体的超时
// context 的超时只能缩短、不能延长，所以「给几个慢接口放宽超时」要反过来配：
// 服务整体的超时配成慢接口需要的长时间，再用 authkratos.NewExclude 把慢接口排除掉，其余接口都缩短成快速超时
// 上游传来的截止时间本来就更短时，照原样保留
// 快速超时不大于零时 panic，否则匹配到的请求会全部立刻超时
// 每次匹配和每次设置超时以 debug 级别写进 applog，显示与否由 applog 的日志级别决定
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
			applog.Debug("fast-kratos-timeout: timeout applied", "new-timeout", cfg.newTimeout)
			return handleFunc(ctx, req)
		}
	}
}
