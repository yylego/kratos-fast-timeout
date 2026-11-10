<!-- TEMPLATE (ZH) BEGIN: BADGES -->

[![GitHub Workflow Status (branch)](https://img.shields.io/github/actions/workflow/status/yylego/kratos-fast-timeout/release.yml?branch=main&label=BUILD)](https://github.com/yylego/kratos-fast-timeout/actions/workflows/release.yml?query=branch%3Amain)
[![GoDoc](https://pkg.go.dev/badge/github.com/yylego/kratos-fast-timeout)](https://pkg.go.dev/github.com/yylego/kratos-fast-timeout)
[![Coverage Status](https://img.shields.io/coveralls/github/yylego/kratos-fast-timeout/main.svg)](https://coveralls.io/github/yylego/kratos-fast-timeout?branch=main)
[![Supported Go Versions](https://img.shields.io/badge/Go-1.26%2B-lightgrey.svg)](https://go.dev/)
[![GitHub Release](https://img.shields.io/github/release/yylego/kratos-fast-timeout.svg)](https://github.com/yylego/kratos-fast-timeout/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/yylego/kratos-fast-timeout)](https://goreportcard.com/report/github.com/yylego/kratos-fast-timeout)

<!-- TEMPLATE (ZH) CLOSE: BADGES -->

# kratos-fast-timeout

选择性超时覆盖中间件，支持基于路由的快速超时控制。

---

<!-- TEMPLATE (ZH) BEGIN: LANGUAGE NAVIGATION -->

## 英文文档

[ENGLISH README](README.md)

<!-- TEMPLATE (ZH) CLOSE: LANGUAGE NAVIGATION -->

## 核心特性

- ⏱️ **快速超时**: 通过路由范围配置在特定路由上缩短超时时间
- 🎯 **路由范围**: INCLUDE/EXCLUDE 模式选择哪些路由启用快速超时
- 🔄 **基于上下文**: 集成 Go context 超时机制（取最短时间）
- 🪵 **调试日志**: 每次路由匹配以 debug 级别写进日志，显示与否由日志级别决定

## 安装

```bash
go get github.com/yylego/kratos-fast-timeout
```

## 设计思路

context 的超时只能缩短、不能延长，所以「给几个慢接口放宽超时」要反过来配：

1. 服务整体的超时配成慢接口需要的长时间；
2. 用 `NewExclude` 把慢接口排除掉，其余接口都用这个中间件缩短成快速超时。

上游传来的截止时间本来就更短时，照原样保留，不会被拉长。

## 使用方法

```go
package server

import (
    "log/slog"
    "time"

    "github.com/go-kratos/kratos/v3/transport/grpc"
    "github.com/yylego/kratos-auth/authkratos"
    "github.com/yylego/kratos-fast-timeout/fastkratostimeout"
)

func NewGRPCServer(applog *slog.Logger) *grpc.Server {
    // 除了导出报表这个慢接口，其余接口都是 3 秒超时
    slowScope := authkratos.NewExclude("/api.v1.Demo/ExportReport")
    cfg := fastkratostimeout.NewConfig(slowScope, 3*time.Second)

    return grpc.NewServer(
        grpc.Timeout(5*time.Minute), // 服务整体超时，按最慢的接口配
        grpc.Middleware(fastkratostimeout.NewMiddleware(cfg, applog)),
    )
}
```

快速超时必须大于零，配成 0 或负数时 `NewMiddleware` 在启动时就 panic，否则匹配到的请求会全部立刻超时。

---

<!-- TEMPLATE (ZH) BEGIN: STANDARD PROJECT FOOTER -->
<!-- VERSION 2025-11-25 03:52:28.131064 +0000 UTC -->

## 📄 许可证类型

MIT 许可证 - 详见 [LICENSE](LICENSE)。

---

## 💬 联系与反馈

非常欢迎贡献代码！报告 BUG、建议功能、贡献代码：

- 🐛 **问题报告？** 在 GitHub 上提交问题并附上重现步骤
- 💡 **新颖思路？** 创建 issue 讨论
- 📖 **文档疑惑？** 报告问题，帮助我们完善文档
- 🚀 **需要功能？** 分享使用场景，帮助理解需求
- ⚡ **性能瓶颈？** 报告慢操作，协助解决性能问题
- 🔧 **配置困扰？** 询问复杂设置的相关问题
- 📢 **关注进展？** 关注仓库以获取新版本和功能
- 🌟 **成功案例？** 分享这个包如何改善工作流程
- 💬 **反馈意见？** 欢迎提出建议和意见

---

## 🔧 代码贡献

新代码贡献，请遵循此流程：

1. **Fork**：在 GitHub 上 Fork 仓库（使用网页界面）
2. **克隆**：克隆 Fork 的项目（`git clone https://github.com/yourname/repo-name.git`）
3. **导航**：进入克隆的项目（`cd repo-name`）
4. **分支**：创建功能分支（`git checkout -b feature/xxx`）
5. **编码**：实现您的更改并编写全面的测试
6. **测试**：（Golang 项目）确保测试通过（`go test ./...`）并遵循 Go 代码风格约定
7. **文档**：面向用户的更改需要更新文档
8. **暂存**：暂存更改（`git add .`）
9. **提交**：提交更改（`git commit -m "Add feature xxx"`）确保向后兼容的代码
10. **推送**：推送到分支（`git push origin feature/xxx`）
11. **PR**：在 GitHub 上打开 Merge Request（在 GitHub 网页上）并提供详细描述

请确保测试通过并包含相关的文档更新。

---

## 🌟 项目支持

非常欢迎通过提交 Merge Request 和报告问题来贡献此项目。

**项目支持：**

- ⭐ **给予星标**如果项目对您有帮助
- 🤝 **分享项目**给团队成员和（golang）编程朋友
- 📝 **撰写博客**关于开发工具和工作流程 - 我们提供写作支持
- 🌟 **加入生态** - 致力于支持开源和（golang）开发场景

**祝你用这个包编程愉快！** 🎉🎉🎉

<!-- TEMPLATE (ZH) CLOSE: STANDARD PROJECT FOOTER -->
