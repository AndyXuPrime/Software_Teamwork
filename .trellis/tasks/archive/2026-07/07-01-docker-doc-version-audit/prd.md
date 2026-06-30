# 检查 Docker 文档镜像版本一致性

## Goal

检查仓库内 Docker、Compose、运行手册和相关文档/配置是否存在镜像版本不一致问题，重点确认是否会启动两个不同版本的 MinIO，并发现类似的基础设施镜像版本漂移风险。

## What I Already Know

- 用户听说当前会起两个不同版本的 MinIO，需要验证是否属实。
- 检查范围应覆盖 Docker 文档以及实际 Compose/Docker 配置，因为文档和配置可能互相漂移。
- 本轮先做审计，必要时再修正文档或配置。

## Requirements

- 扫描仓库中 Dockerfile、Compose YAML、环境示例、运行手册和 Docker 相关 Markdown。
- 汇总 MinIO 镜像、`mc` 镜像、端口、服务名、bucket 初始化和依赖关系的所有出现位置。
- 对比其他基础镜像版本引用，例如 Postgres、Redis、Qdrant、Go、Node/Bun、Python、PaddleOCR/Parser 相关镜像。
- 判断是否存在：
  - 同一组件多个版本 tag；
  - 文档描述和 Compose 实际配置不一致；
  - 同一运行路径会同时启动两个版本；
  - 使用 `latest` 或未 pin 版本导致不可复现；
  - 同类问题的其他镜像漂移。
- 产出清晰结论：问题是否存在、影响范围、建议修复项。

## Acceptance Criteria

- [ ] 列出所有 MinIO 相关镜像引用和版本。
- [ ] 明确回答“是否会起两个不同版本的 MinIO”。
- [ ] 列出发现的其他 Docker 镜像版本不一致或未 pin 风险。
- [ ] 如需修改，改动范围清晰；如仅审计，说明无需改动。
- [ ] 运行必要的文本扫描或配置解析检查。

## Definition of Done

- 审计结论可追溯到具体文件路径和行号。
- 工作树保持可解释；若修改文件，走 Trellis check 和提交流程。

## Out of Scope

- 不实际启动 Docker 服务，除非文本审计无法判断。
- 不升级镜像版本或重构 Compose 架构，除非用户确认需要修复。
- 不改业务服务代码。

## Technical Notes

- 重点命令：`rg`、`find`、YAML 解析/文本汇总。
- 重点目录：`docs/`、`deploy/`、根目录 Compose 文件、`services/*/Dockerfile`、`services/*/api` 或 runtime docs。
