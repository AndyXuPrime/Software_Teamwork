# Document 富 DOCX Worker 工具链基线

本文记录 Document 服务富 DOCX 生成 worker 的候选工具链、调用边界、
`SimpleDOCXGenerator` 职责分工和本地 smoke 验证方向。

当前仓库不保留业务服务打包路径。富 DOCX worker 后续如落地，应作为
host-run 工具链接入任务实现，不能把 Document 服务重新接回 Docker 默认启动路径。

## 1. 工具链选型

| 候选工具 | 决策 | 理由 |
| --- | --- | --- |
| Pandoc CLI | 选定为富 DOCX 主候选 | 纯 CLI、无 GUI 依赖、进程外调用简单；Markdown 到 DOCX 转换覆盖标题、段落、表格和列表。 |
| LibreOffice headless | 暂不引入，保留后续候选 | 依赖更重，首期富 DOCX 需求优先由 Pandoc 覆盖。 |

当前 `SimpleDOCXGenerator`（`services/document/internal/service/docx_generator.go`）
使用 Go 标准库 `archive/zip` 和 OOXML 实现基础 DOCX 导出。富 DOCX 阶段需要
更完整的 Markdown、reference template、页眉页脚、自动目录和样式能力。

## 2. 当前边界

- 当前生产路径仍是内置 `SimpleDOCXGenerator`。
- 当前 Document 服务不调用 Pandoc 或 LibreOffice。
- `DOCUMENT_PANDOC_PATH` 和 `DOCUMENT_LIBREOFFICE_PATH` 只是预留 host-run 配置。
- 后续接入时必须同步本文、`docs/architecture/technology-decisions.md`、
  `docs/services/document/README.md` 和实现说明。

## 3. 调用边界

| 项目 | 规格 |
| --- | --- |
| 输入 | GFM Markdown（UTF-8），由 Document worker 从已保存章节内容构造。 |
| 临时文件 | `os.MkdirTemp` 创建隔离目录，权限 0700；路径不得进入 HTTP 响应。 |
| 输出 | Office Open XML DOCX bytes。 |
| 后续 | 上传 File Service，更新 `ReportFile` 元数据，删除临时文件。 |
| 错误 | 返回依赖错误摘要；不得泄露正文、prompt、token、内部路径或 provider 原始响应。 |

## 4. Fallback 策略

后续接入 Pandoc CLI 时应实现：

```text
触发 fallback:
1. exec.LookPath(pandocPath) 失败
2. Pandoc subprocess context 超时
3. 输出文件为空或过小

不触发 fallback:
- subprocess 非零退出码，任务应失败并记录脱敏摘要

fallback 行为:
- 使用 SimpleDOCXGenerator 生成基础 DOCX
- 写 warn 级别日志，包含 reportID/jobID，不含正文
- ReportFile 元数据可标注 generatorHint: "simple"
```

## 5. Host-Run Smoke 方向

后续富 DOCX 任务应在宿主机安装 Pandoc CLI 后验证：

```bash
pandoc --version
pandoc -f gfm -t docx smoke-test.md -o smoke-test.docx
unzip -l smoke-test.docx | grep "word/document.xml"
```

最终人工验证：用 Word 或 LibreOffice Writer 打开 `smoke-test.docx`，确认标题层级、
正文段落、无序列表和表格正确渲染。

## 6. 后续接入约束

| 约束 | 说明 |
| --- | --- |
| `go.mod` | 富 DOCX 通过 subprocess 调用，不引入 Go binding。 |
| 环境变量 | 使用 `DOCUMENT_PANDOC_PATH`，必要时补充超时和模板路径配置。 |
| 安全 | 不允许把用户可控字符串拼接进 shell；使用 `exec.CommandContext` 和固定参数列表。 |
| 文档 | 接入 PR 必须同步服务 README、implementation、技术选型和测试策略。 |
