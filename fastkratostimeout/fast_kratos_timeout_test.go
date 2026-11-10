package fastkratostimeout

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/transport"
	"github.com/stretchr/testify/require"
	"github.com/yylego/kratos-auth/authkratos"
)

// mockHeader is a blank transport.Header, the middleware does not read headers
// mockHeader 是空的 transport.Header，这个中间件不读请求头
type mockHeader struct{}

func (mockHeader) Get(string) string      { return "" }
func (mockHeader) Set(string, string)     {}
func (mockHeader) Add(string, string)     {}
func (mockHeader) Values(string) []string { return nil }
func (mockHeader) Keys() []string         { return nil }

// mockTransport is a stub transport.Transporter on the service side, carrying just the operation
// mockTransport 是服务端 transport.Transporter 的桩，只带接口名
type mockTransport struct {
	operation string
}

func (t *mockTransport) Kind() transport.Kind            { return transport.KindGRPC }
func (t *mockTransport) Endpoint() string                { return "" }
func (t *mockTransport) Operation() string               { return t.operation }
func (t *mockTransport) RequestHeader() transport.Header { return mockHeader{} }
func (t *mockTransport) ReplyHeader() transport.Header   { return mockHeader{} }

const (
	operationQuick = "/api.v1.Demo/Quick"
	operationSlow  = "/api.v1.Demo/Slow"
)

// runWithDeadline runs the middleware on one request and returns the time left on the deadline the handle func sees
// runWithDeadline 用中间件处理一次请求，返回 handle 函数看到的截止时间还剩多久
func runWithDeadline(t *testing.T, cfg *Config, ctx context.Context, operation string) time.Duration {
	handle := NewMiddleware(cfg, slog.New(slog.DiscardHandler))(func(ctx context.Context, req any) (any, error) {
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		return time.Until(deadline), nil
	})
	ctx = transport.NewServerContext(ctx, &mockTransport{operation: operation})
	reply, err := handle(ctx, nil)
	require.NoError(t, err)
	return reply.(time.Duration)
}

func TestNewMiddleware(t *testing.T) {
	// The service timeout is one minute, each route except the slow one gets 2 seconds
	// 服务整体超时一分钟，除了慢接口，其余接口都是 2 秒
	cfg := NewConfig(authkratos.NewExclude(operationSlow), 2*time.Second)
	ctx, stop := context.WithTimeout(context.Background(), time.Minute)
	defer stop()

	t.Run("matched route gets the fast timeout", func(t *testing.T) {
		left := runWithDeadline(t, cfg, ctx, operationQuick)
		require.LessOrEqual(t, left, 2*time.Second)
	})

	t.Run("excluded route keeps the service timeout", func(t *testing.T) {
		left := runWithDeadline(t, cfg, ctx, operationSlow)
		require.Greater(t, left, 50*time.Second)
	})
}

// TestNewMiddleware_TightUpstream checks the fast timeout does not stretch an upstream deadline that is tight to begin with
// TestNewMiddleware_TightUpstream 检查快速超时不会把本来更短的截止时间拉长
func TestNewMiddleware_TightUpstream(t *testing.T) {
	cfg := NewConfig(authkratos.NewExclude(), time.Minute)
	ctx, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()

	left := runWithDeadline(t, cfg, ctx, operationQuick)
	require.LessOrEqual(t, left, time.Second)
}

// TestNewMiddleware_ZeroTimeout checks a timeout not above zero stops the startup
// TestNewMiddleware_ZeroTimeout 检查超时时间不大于零时启动就 panic
func TestNewMiddleware_ZeroTimeout(t *testing.T) {
	require.Panics(t, func() {
		NewMiddleware(NewConfig(authkratos.NewExclude(), 0), slog.New(slog.DiscardHandler))
	})
}

func TestNewConfig(t *testing.T) {
	cfg := NewConfig(authkratos.NewExclude("/api.v1.Demo/Export"), 3*time.Second)
	require.Equal(t, 3*time.Second, cfg.newTimeout)
	require.Equal(t, authkratos.EXCLUDE, cfg.routeScope.Side)
}
