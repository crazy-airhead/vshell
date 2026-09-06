# ISSUE-0008 — 集成 HTTPS 证书自动签发与续签（acme.sh 远程管理）

> **编号**：0008　**状态**：🟡 进行中（功能已实现，待实机验收）　**严重程度**：💡 体验（新功能需求，非缺陷）
> **发现日期**：2026-09-06　**相关任务**：证书管理 / acme.sh 集成

## 问题描述

为 Web 服务器配置 / 续签 HTTPS 证书是高频且繁琐的运维工作，手动申请、部署、续签效率低，且易因遗忘导致服务中断。需求：在 vShell 中集成基于 **acme.sh** 的远程证书管理能力——客户端仅提供向导式配置界面与状态监控，**签发 / 安装 / 定时续签全部由服务器端 acme.sh + cron 独立完成，不依赖客户端常驻在线**。支持单域名与泛域名（DNS-01 验证，无需开放 80/443）。

> 本条目为**新功能需求登记**（完整需求与设计文档见文末附录），非缺陷修复。

## 期望行为（功能需求清单）

| # | 需求 | 要点 |
|---|------|------|
| FR-1 | 服务器连接管理 | 复用现有 SSH 连接能力（密码 / 密钥认证） |
| FR-2 | acme.sh 自动安装 | 检测远端是否已安装；未装则 `curl https://get.acme.sh \| sh` 安装，默认 CA 设为 Let's Encrypt；网络受限时支持 SFTP 上传本地安装包（可选） |
| FR-3 | 证书申请 | 多域名（含泛域名 `*.example.com`）+ DNS 服务商凭据（阿里云 / DNSPod / Cloudflare 等）→ 凭据经 SSH 临时注入服务器 → `--issue --dns <provider>` 签发；凭据保存至 `account.conf` 供后续自动续签复用 |
| FR-4 | 证书安装 | `--install-cert` 将证书 / 私钥复制到用户指定路径（如 `/etc/nginx/ssl/`）及文件名；可配 reload 命令（如 `systemctl reload nginx`）；目标目录缺失自动创建 |
| FR-5 | 自动续签 | 服务器端 cron 每日 `acme.sh --cron`；客户端检测该 cron 任务是否存在、缺失则补加；客户端异常退出不影响服务器端续签；支持手动触发 `--renew --force` |
| FR-6 | 状态监控 | 解析 `acme.sh --list` 展示域名 / 签发时间 / 到期时间 / 存储路径；即将到期（<30 天）醒目提示 |
| FR-7 | 日志查看 | 获取最近一次 acme.sh 操作日志（签发 / 续签）用于排错 |

非功能要点（NFR）：

- **安全**：凭据传输走 SSH 隧道；客户端不落明文（建议钥匙串 / 加密存储）；服务器上临时凭据文件用后即删；`account.conf` 权限 600；日志脱敏。
- **可靠**：任一步骤失败给出明确错误并可重试；网络不稳定设合理超时与重试。
- **跨平台**：客户端 Win / macOS / Linux；服务器端仅主流 Linux 发行版（acme.sh 限制，需在文档注明）。
- **性能**：常规签发 30 秒内返回（不含 DNS 传播等待）；状态查询轻量、避免频繁远程执行。

## 实际行为

功能不存在：当前 vShell 无任何证书管理能力，需前后端新增模块。

## 影响范围

新增功能模块，涉及两端与远端服务器：

- **后端（`../vshell-artifacts`）**：新增 CertManager（环境检测、acme.sh 安装、命令构建与执行、输出解析、临时文件清理）、DNS 凭据安全存储、本地证书任务配置；复用 `internal/ssh`、`internal/sftp`、`internal/crypto`。
- **前端**：新增证书管理页签（证书列表 / 添加证书向导 / 操作按钮）、设置项（默认 CA、超时等）、i18n 文案。
- **服务器端**：安装 acme.sh（`~/.acme.sh/`）+ 每日 cron 任务；客户端只做编排，不常驻。

## 相关文件 / 符号

（代码位于 `../vshell-artifacts` 工作区）

**复用（已存在）**

- `internal/ssh/client.go` — `Manager`：SSH 连接与会话管理，证书操作的远程命令执行基础。
- `internal/sftp/manager.go` / `client.go` — SFTP 上传能力（临时凭据文件、离线安装包上传）。
- `internal/crypto/crypto.go` — AES-256-GCM 加密：DNS API 凭据落库加密应复用此模块。
- `internal/db/db.go` + `migrations.go` + `internal/models/` — 新增证书任务表 / 模型在此扩展。
- `internal/app/app.go` — `AppService` 绑定挂载点；新增证书相关绑定方法；长任务进度走 Wails Events（同 `monitor` / `sftp:progress` 模式）。

**新增（规划）**

- `internal/cert/` — CertManager：acme.sh 安装检测与部署、远程命令构造、`--list` 输出解析、cron 任务检查补加、临时文件清理。
- `frontend/src/components/cert/` + `frontend/src/stores/cert.ts` — 证书管理界面与状态。

## 建议方案（可选）

设计要点（完整文档见附录）：

- **架构分工**：客户端只做「编排 + 展示」；签发 / 续签 / reload 全由服务器端 acme.sh 承担。cron 每日检查，到期 <60 天自动续签，续签后自动执行 `--install-cert` 保存的 reload 命令。
- **凭据注入**：凭据以环境变量写入临时文件（权限 600）经 SFTP 上传 → `source /tmp/acme_dns_env.sh && acme.sh --issue --dns dns_ali -d … --dnssleep 120` → 执行完立即 `rm -f` 删除；凭据不进命令行历史。
- **证书安装**：`acme.sh --install-cert -d <domain> --key-file … --fullchain-file … --reloadcmd "systemctl reload nginx"`；先 `mkdir -p` 保证目录存在；reload 命令可按 Web 服务器类型（Nginx / Apache / Caddy）自动生成。
- **续签保障**：`crontab -l | grep -q acme.sh` 检查，缺失则补加 `0 0 * * * "/root/.acme.sh"/acme.sh --cron --home "/root/.acme.sh" > /dev/null`。
- **状态解析**：解析 `acme.sh --list` 列表输出（Main_Domain / SAN_Domains / Created / Renew）；精确到期时间读证书文件 `openssl x509 -enddate -noout -in <crt>`。
- **DNS 服务商映射**：客户端维护映射表 `aliyun→dns_ali`、`dnspod→dns_dp`、`cloudflare→dns_cf` 等，随 acme.sh 更新同步维护。

**登记评审备注（实现时需核实 / 决策）**：

1. **acme.sh 参数需按实际版本核实**：设计文档中 `--issue --save` 并非 acme.sh 现有参数——acme.sh 首次使用 DNS API 时会自动将凭据写入 `account.conf`；`--dns-check-tries` 等参数同样以官方文档为准。命令构建函数落地前先核对。
2. **安装路径勿硬编码 `/root`**：acme.sh 实际安装于登录用户 `$HOME/.acme.sh/`（root 与 sudo / 普通用户场景不同），应运行时探测 `$HOME`，文档中 `/root/.acme.sh` 仅示例。
3. **凭据存储选型待定夺**：设计文档提议引入 `go-keyring`（系统钥匙串）；但本项目已有 `internal/crypto`（AES-256-GCM）+ SQLite 的加密存储模式（CLAUDE.md 关键规则 2），建议优先复用现有机制，避免两套凭据存储并存。
4. **长流程必须走 Events**：签发含 DNS 传播等待可达数分钟，进度与日志输出必须经 Wails Events 推送（同终端 I/O / 监控模式），不得同步 Call 阻塞 UI。
5. cron 任务 acme.sh 安装时默认已注册，「检查 + 补加」逻辑保留即可。

## 解决记录

**第 1 轮**（2026-09-06，功能实现）

- **处理**（按「建议方案」全量落地，5 条登记评审备注逐条落实）：
  - **后端 `internal/cert/`（新包，6 文件 + 3 测试）**：
    - `runner.go` — 流式远程执行：`GetSSHClient` → 新 session（不申请 PTY）→ Stdout/StderrPipe 各起 goroutine 按行读 → `cert:log` 事件逐行推送 → `sess.Wait` 取退出码；取消/超时以 `sess.Close()` 兜底。
    - `commands.go` — 纯函数命令构建：`shQuote` POSIX 单引号安全引用；`BuildIssueCmd` 为 `chmod 600 env && . env && rm -f env && acme.sh --issue … ; rc=$?; rm -f env 兜底; exit $rc`（凭据 source 后即删、失败也删、退出码保真）；`BuildInstallCertCmd`（`mkdir -p` + reloadcmd）；`BuildRenewCmd --force`；`BuildEnsureCronCmd`（`grep -q acme.sh` 缺失才补加，运行时 `$HOME`）；`BuildTailLogCmd`。acme.sh 一律 `"$HOME"/.acme.sh/acme.sh` 绝对路径（备注 2），**无 `--save`**（备注 1，核实官方 README 确认该参数不存在，凭据首次调用 DNS API 自动写入 `account.conf`），ECC 密钥自动附 `--ecc`，issue/renew 带 `--log` 支撑 FR-7。
    - `parser.go` — 宽容解析：`--list` 兼容竖线/多空格两种版式、找不到表头返回空不报错，格式漂移时回退 `ls ~/.acme.sh/*/` 目录枚举；`--info` key=value 解析（剥引号，`Le_NextRenewTime` epoch → 剩余天数）；检测脚本 `vshell:key=value` 行解析。
    - `dnsproviders.go` — 注册表：aliyun(dns_ali)/dnspod(dns_dp)/cloudflare(dns_cf)/custom（自填 `dns_xx` 插件名 + 任意 env 键值对）；`ValidateCredentials` 校验必填项与 env 键名合法性（防注入）；`BuildEnvFileContent` 生成 `export KEY='value'`；临时文件 `/tmp/.vshell_acme_<uuid>.env` 随机名。
    - `manager.go` + `issue.go` — 编排状态机：detect →（未装则装 + `--set-default-ca --server letsencrypt`）→ SFTP 上传凭据 → issue（20min 超时）→（auto_install：install-cert）→ cron 检查补加（备注 5）→ done；每步 emit `cert:stage`，命令输出逐行 emit `cert:log`。`Renew`/`Remove`/`InstallAcmeSh` 同构。
  - **后端模型/绑定**：`internal/models/cert.go`（CertTask/CertTaskForm 读写分离、凭据字段 `json:"-"`）；`db/migrations.go` 追加 `cert_tasks` 表；`internal/app/cert.go` 同步方法（ListCertTasks/Create/Update/Delete/GetCertTaskCredentials/ListDNSProviders/DetectCertEnvironment/ListRemoteCerts/GetCertServerLog/CancelCertOp）+ 异步方法（StartAcmeShInstall/StartCertIssue/StartCertRenew/StartCertRemove：goroutine + 事件，IsRunning 防重入，收尾更新 last_status 并 emit `cert:task-updated`）。凭据 AES-256-GCM 加密落库、复用 `db.Crypto()`（备注 3，未引入 go-keyring）。启动时 `resetInterruptedCertTasks` 把卡在 running 的任务复位为 failed。
  - **前端**：`stores/cert.ts`（`cert:log/cert:stage/cert:task-updated/cert:task-deleted/cert:op-done` 一次性订阅守卫，日志 cap 2000 行）；`components/cert/` 五组件——CertPanel（按连接分组列表、状态/剩余天数 NTag、<30 天黄色提示、行内续签/日志/编辑/移除/删除）、CertWizard（NSteps 四步：连接+环境检测+在线装 acme.sh→域名+DNS 凭据（数据驱动表单，custom 自填）→部署设置（certDir/文件名/reloadcmd preset：nginx/apache/caddy/自定义；高级折叠含 keyLength/dnsSleep/**testMode 默认开**走 staging）→执行+阶段进度+实时日志+失败重试）、CertEditModal（凭据留空=保留，reveal 按钮解密回显）、CertLogModal（本地操作日志 + 服务器 acme.sh.log 双页签）、CertLogView；入口 3 处（layout.ts `'certs'`、ActivityBar shield-check 图标、App.vue 条件渲染）；i18n en/zh-CN `certs` 块 ~70 key 对齐。
- **评审备注落实**：1 无 `--save`（参数已对照官方 README 核实）；2 无 `/root` 硬编码（单测断言）；3 复用 `internal/crypto`；4 长流程全部走 Events（签发含 DNS 等待最长 20min，不阻塞 UI，可取消）；5 cron 仅检查补加。
- **范围裁剪**：FR-2 的「网络受限 SFTP 上传离线安装包」为可选项，本轮未做。
- **校验**：`go test ./internal/cert/`（命令构建 golden 断言含无 `--save`/无 `/root`/`--ecc` 联动/shQuote 转义，解析器竖线/定宽/空 fixture）、`go build ./...`、`go vet ./...`、`pnpm typecheck`、`pnpm build` 全部通过，0 新错。
- **提交**：artifacts 分支 `ae89ab5`。

**第 2 轮**（2026-09-06，容器实机集成验证）

- **反馈 / 触发**：按验收计划做无 GUI 的实机验证——docker 起 openssh-server 容器（Alpine 非 root + Ubuntu root 两种），以 `integration` build tag 的集成测试直连真实 sshd 跑 detect / list / exit code / stderr 分流 / 真实安装 acme.sh 全链路。
- **根因**（发现 3 个真缺陷 + 1 个 UX 缺陷，均已修复，提交 `2c27610`）：
  1. **runner 漏调 `sess.Start(cmd)`**（最严重）——只建 Stdout/StderrPipe 即 `Wait`，exec 请求从未发出，远程命令不执行、所有远程操作无限挂死；goroutine 堆栈定位后补上 Start。
  2. **非零退出码未转 error**——`exit=7 err=nil`，意味着 `acme.sh --issue` 失败（DNS 凭据错误等）会被编排层当成功签发；现 exit ≠ 0 一律返回 error（Combined 输出保留供诊断）。
  3. **`curl | sh` 管道陷阱**——curl/wget 缺失或下载失败时 `sh` 读空输入退出 0，安装"假成功"；且 get.acme.sh 引导脚本自身在缺 crontab 时打印 "Pre-check failed, cannot install" 仍退出 0。改为 `mktemp` 下载临时文件 → 校验下载退出码 → 再执行；编排层安装后复核兜底，缺 cron 场景给出明确错误提示。
  4. **ListCerts UX**——acme.sh 未安装时 `--list` exit 127 会报错；改为目录枚举回退优雅返回空列表，仅传输级错误才上报。
- **校验**：容器实测（Alpine 非 root：detect/列表优雅空/cron 权限错误如实透出；Ubuntu root：真实安装 acme.sh 成功、安装器自动注册 cron、`--set-default-ca` 成功、真实 `--list` 表头解析通过）；`go test ./...`、`go build`、`go vet`、`pnpm typecheck` 全绿。
- **实测确认的 `--list` 真实格式**（acme.sh 3.x，2026-09）：Tab 分隔，列为 Main_Domain / KeyLength / SAN_Domains / **Profile** / CA / Created / Renew——比设计文档多一列 `Profile`；解析器按空白定宽切列对该格式天然兼容（域名/密钥/日期列均正确，CA 列会多带上 Profile 值，仅展示层面可忽略），已把真实表头固化为 fixture。
- **遗留**：
  - 完整签发仍需真实域名 + DNS API 凭据（staging `--test`），无法在容器内无域名验证——待用户提供实机验收。
  - 非 root 部署到 `/etc/nginx/ssl` 的权限场景（错误会原样透出 acme.sh 输出尾部）。

**第 3 轮**（2026-09-06，实机验收反馈：zai500.com 签发失败分析）

- **反馈 / 触发**：用户实机对 `zai500.com`（dns_dp）发起签发，失败。服务器 acme.sh.log 显示 DNSPod API（dnsapi.cn）返回 nginx 401，TXT 记录未加上，签发在 DNS 验证前终止：`Error adding TXT record to domain: _acme-challenge.zai500.com`。
- **根因**（两个，均已修复，提交 `52ddeef`）：
  1. **凭据类型用错**：用户填的是**腾讯云 CAM 密钥**（`login_token=AKID…`，SecretId/SecretKey 格式），而 `dns_dp` 插件要求 **DNSPod 传统 Token**（纯数字 ID + token，dnspod.cn 控制台单独签发）——两套体系不通用，dnsapi.cn 拒绝时返回的是 nginx 401 HTML 而非 JSON，难以直观辨认。
     - 修复：新增 `tencent` 提供商（`dns_tencent` 插件，`Tencent_SecretId`/`Tencent_SecretKey`，已核对 acme.sh dnsapi/dns_tencent.sh 源码，API 即 `dnspod.tencentcloudapi.com`——用户现有 AKID 密钥直接可用）；`dnspod` 字段标签/占位符明确「数字 Token ID，非腾讯云 SecretId」。
  2. **列表解析缺陷（vShell 侧）**：签发失败后 `--list` 只有表头无数据行，目录枚举回退误把 acme.sh 内部目录（`ca`/`deploy`/`dnsapi`/`notify`）当域名，并对每个内部目录发起 `--info`（用户日志 14:12 段可见）。
     - 修复：`ParseListOutput` 增加第二返回值（表头是否被识别）——表头识别但零行 = 真无证书，不触发回退；`ParseAcmeDirsOutput` 过滤不含点号的条目（公网验证 FQDN 必含点号）。
- **校验**：`go test ./...`（新增表头识别、内部目录过滤、tencent 提供商用例）、`go build`、`go vet`、`pnpm typecheck` 全绿。
- **备注**：该次签发跑在 Let's Encrypt **生产环境**（日志为 acme-v02），说明向导的 staging 默认勾选被用户取消；正式签发前建议先走 staging（测试模式）确认凭据与 DNS 生效。
- **遗留**：
  - 用户改用「Tencent Cloud DNS (SecretId/SecretKey)」提供商或补办 DNSPod 传统 Token 后重试签发（建议先开测试模式）。
  - 签发成功后的部署 / cron / 续签链路仍待实机确认。

**第 4 轮**（2026-09-06，续签路径同因失败 + 凭据治愈机制）

- **反馈 / 触发**：用户点「续签」再次失败，日志与第 3 轮完全一致（`Renewing: 'zai500.com'` → dnsapi.cn 401 → `Error adding TXT record`）。
- **根因**：续签命令（`--renew --force`）不推送凭据，acme.sh 沿用**上次失败签发持久化到服务器 `account.conf` 的错误凭据**（AKID 配 dns_dp）——用户即使把任务凭据改对，点续签也会拿旧错误凭据反复失败，且不易察觉。
- **处理**（提交 `44ff366`）：
  - `BuildRenewCmd` 增加可选 envFile：续签前 source 任务凭据再执行（acme.sh dns 插件 env 优先于 account.conf，且读后回写 account.conf——一次正确续签即治愈服务器端凭据，后续 cron 续签全自动）；命令链与 issue 同款（chmod 600、source 后即删、失败兜底删、退出码保真）。
  - `Manager.Renew` 接收凭据并经 SFTP 上传临时 env 文件；`app.startCertOperation` 续签路径同样解密传递。
  - `CertEditModal` 切换 DNS 提供商时清空旧凭据（对齐向导行为），避免旧字段名凭据（DP_*）被一并上传。
- **校验**：`go test ./...`（新增续签 env 变体断言）、`go build`、`go vet`、`pnpm typecheck` 全绿。
- **给用户的恢复路径**（二选一，改完点**签发**而非续签——修复后的续签也能治愈凭据，但签发路径最直接）：
  1. 重建应用后编辑任务：DNS 提供商改选「Tencent Cloud DNS (SecretId/SecretKey)」，凭据不变（AKID 两串），开测试模式签发；
  2. 或到 dnspod.cn 控制台申请传统 API Token（数字 ID + token），提供商保持 DNSPod，替换凭据后签发。
- **遗留**：签发成功 → 部署 → cron → 续签全链路仍待实机确认。

**第 5 轮**（2026-09-06，日志体验调整）

- **反馈 / 触发**：「renew 的时候把 acme 的日志也先清理掉，两个日志窗口都增加复制按钮，复制所有内容。」
- **处理**（提交 `2baa2c1`）：
  - 后端 `BuildTruncateLogCmd`（`: > "$HOME/.acme.sh/acme.sh.log"` 截断非删除），在签发 / 续签 / 移除 / 安装 acme.sh 四个编排入口各调用一次——服务器 `acme.sh.log` 跨运行累积，操作前截断使「Server acme.sh log」窗口只反映本次操作（不只 renew，issue 同样受益）；截断失败仅在操作日志记一条 note 不中断。
  - 前端 `CertLogModal` 服务器日志页补复制按钮（复制全部，带成功/失败提示）；操作日志窗口 `CertLogView` 原有复制按钮同步加成功提示。
- **校验**：`go test ./...`、`go build`、`go vet`、`pnpm typecheck` 全绿。
- **遗留**：同第 4 轮——全链路待实机确认。

**第 5 轮补充**（2026-09-06，复制报错修复）

- **反馈 / 触发**：「日志复制报错 NotAllowedError: The request is not allowed by the user agent or the platform…」——Wails 的 WKWebView 中 `navigator.clipboard.writeText` 即使在用户手势内也会被拒。
- **处理**（提交 `3d667e6`）：证书日志窗口改为复用项目现成的 `utils/clipboard.writeClipboard`（问题 0004 已为此建立：Wails 原生 `Clipboard.SetText` 优先、web API 兜底），不再各自调用 `navigator.clipboard`；复制失败给出明确 toast（新增 `certs.copyFailed` 文案）。
- **教训**：新增剪贴板相关功能前先查 `utils/clipboard.ts`——项目已有统一封装。
- **追加**（同日）：两个复制按钮统一放右上角（`CertLogView` 从底部移到顶部，对齐服务器日志页）。

**第 6 轮**（2026-09-06，续签无法切换 DNS 提供商）

- **反馈 / 触发**：用户改任务提供商后点续签，服务器日志仍 `_currentRoot='dns_dp'`，继续对 dnsapi.cn 发 AKID 凭据 → 401。
- **根因**：acme.sh 的 `--renew` **原样复用首次签发时写入域名 conf 的 `Le_Webroot`（插件）与凭据**——对服务器上已登记的域名，`--renew` 永远无法应用 vShell 里改过的提供商/凭据，第 4 轮「续签推送凭据」的修复因此对换提供商场景完全无效（env 只影响插件内部读取，不改变用哪个插件）。
- **处理**（提交 `a6643b9`）：续签改为走 `--issue --force`（`BuildIssueCmd` 增加 force 参数）——`--issue` 按命令行参数**重写域名 conf** 的插件与凭据，任务配置始终生效；`--force` 保证未到期也强制重签。手动「签发/重试」同样带 `--force`（否则已有未到期证书时 acme.sh 直接 Skip，配置同样不生效）。`Renew` 编排解析插件、显式重跑 install-cert 部署（不依赖 conf hook 在 `--issue` 路径下的自动行为）；`BuildRenewCmd` 删除。
- **校验**：`go test ./...`（含 --force 出现/缺席断言）、`go build`、`go vet`、`pnpm typecheck` 全绿。
- **给用户**：重建后编辑任务（提供商 → Tencent Cloud DNS，AKID 两串，测试模式勾选）→ 点「签发」或「续签」均可，现在两条路径都会应用 dns_tencent。操作日志出现 `dns_tencent.sh` / `dnspod.tencentcloudapi.com` 即为生效标志。
- **遗留**：全链路待实机确认。

**第 7 轮**（2026-09-06，「看不到签发按钮」）

- **反馈 / 触发**：「没有看到签发的按钮，只看到 update 的按钮」——用户编辑任务保存后找不到重新发起签发的入口（编辑弹窗只有 Update；「签发」按钮在向导里，向导只用于新建）。
- **根因**：面板行内的续签按钮带 `v-if="!isRunning"`（运行中隐藏），且「Renew」语义让人以为只对已有证书有效——修改配置后的重签没有明显入口。
- **处理**（提交 `4674604`）：行内按钮改为「**立即签发 / 续签**」二合一，始终显示：空闲可点、运行中 loading/禁用（tooltip 显示状态）。后端签发与续签已统一走 `--issue --force`，同一按钮覆盖首次签发、改配置后重签、到期续签三种场景；ACME 邮箱取自向导记住的 localStorage（仅服务器未装 acme.sh 时使用）。
- **校验**：`pnpm typecheck` 全绿（本轮纯前端）。
- **遗留**：全链路待实机确认。

**第 8 轮**（2026-09-06，「没点操作日志也被清空」）

- **反馈 / 触发**：「没有点续签/签发的，怎么日志也会被清空？」
- **排查**：grep 全部清日志路径——服务器日志截断只在签发/续签/移除/安装四个操作入口；前端 clearLog 只在三个 start 动作。无隐藏触发。
- **根因**：**操作日志（Logs 页）为纯内存态**，应用重启（当天多次重新构建）即全部丢失——不是被某个操作清掉的，是没持久化（原设计有 last_log 落库，实现时砍掉了）。服务器 acme.sh 日志不受重启影响。
- **处理**（提交 `f1bb567`）：补上持久化——`cert_tasks.last_log` 列（additive 迁移）；`cert.Manager` 流式执行时同步缓冲任务日志（尾留 256KB，stderr 行带前缀），`TakeTaskLog` 取走；签发/续签/移除收尾落库（移除+删任务成功除外）；新增 `GetCertTaskLog` 绑定；前端 `ensureOpLog` 在内存为空时（重启后）回填，日志弹窗打开即显示上次操作输出。
- **校验**：`go test ./...`、`go build`、`go vet`、bindings 再生成、`pnpm typecheck`、`pnpm build` 全绿。
- **遗留**：全链路待实机确认。

**验收进展**（2026-09-06 17:43，zai500.com 实机签发成功）

- **签发链路全部打通**：dns_tencent（腾讯云 SecretId/SecretKey）验证通过 → Let's Encrypt **生产环境**签发成功（90 天，至 2026-12-05）→ 证书/私钥部署至 `/etc/nginx/ssl/zai500.com.{crt,key}` → acme.sh 已注册 cron 自动续签。问题 0008 的核心链路（FR-1~FR-6）实机验证通过。
- **残留一项环境问题（非本功能缺陷）**：服务器无 systemd `nginx.service`（`Unit not found`），reload 命令失败导致任务状态显示 failed（证书本体已就位不受影响）——待用户确认服务器 web 服务形态（docker / 服务名不同 / 未装）后修改任务的 reload 命令。
- 提醒已告知用户：该次签发走的**生产环境**（测试模式未勾选），LE 对同域名 168 小时内最多 5 张重复证书，后续调试应走 staging。
- **追加修复**（同日，提交 `5a18c8f`）：签发/安装阶段 acme.sh 会把整张证书 PEM 打进日志（数百行），向导内容区只有 min-h 无上限，日志区把弹窗一直撑高——改为内容区固定 `h-[520px]`，各步内容/日志在边界内滚动。

---

## 附录：完整需求与设计文档（登记原文，2026-09-06）

<details>
<summary>展开全文</summary>

# SSH 客户端集成 HTTPS 证书自动续签功能需求与设计文档

## 1. 引言

### 1.1 背景
现有 SSH 客户端工具基于 Go 语言的 Wails3 框架开发，主要面向服务器运维人员，提供图形化 SSH 连接管理、远程命令执行等功能。在日常运维中，为 Web 服务器配置和管理 HTTPS 证书是一项高频且繁琐的工作。传统方式需要手动申请、部署、续签证书，效率低且容易因遗忘导致服务中断。本需求旨在将 HTTPS 证书的自动签发与续签能力集成到 SSH 客户端中，实现一键配置、服务器端自动续签，提升运维效率。

### 1.2 目标
- 通过 SSH 客户端远程管理服务器的 HTTPS 证书，支持单域名和泛域名证书。
- 采用 acme.sh 作为服务器端 ACME 客户端，利用其强大的 DNS 验证和自动续签能力。
- 实现证书的申请、安装、自动续签全流程自动化，客户端仅需提供配置界面和状态监控。
- 证书定时续签任务运行在服务器端，不依赖客户端常驻在线。
- 保证敏感信息（DNS API 密钥、SSH 凭证）的安全传输与存储。

### 1.3 术语
- **ACME**：Automatic Certificate Management Environment，自动证书管理环境协议。
- **acme.sh**：纯 Shell 编写的 ACME 客户端，支持多种 DNS 服务商 API，广泛用于 Let's Encrypt 证书签发。
- **DNS-01 验证**：通过添加 DNS TXT 记录验证域名所有权，支持泛域名，无需开放 80/443 端口。
- **Wails3**：使用 Go 和 Web 技术构建桌面应用的框架。

---

## 2. 需求描述

### 2.1 功能需求

#### FR-1 服务器连接管理
- 复用现有 SSH 客户端连接功能，支持密码和密钥认证。
- 在证书管理模块中选择目标服务器，建立 SSH 会话。

#### FR-2 acme.sh 自动安装
- 客户端能够检测远程服务器是否已安装 acme.sh。
- 若未安装，则自动执行安装脚本（`curl https://get.acme.sh | sh`），并设置默认 CA 为 Let's Encrypt。
- 安装过程需要处理网络受限情况，支持从本地上传安装包（可选）。

#### FR-3 证书申请
- 用户可输入证书域名（支持多个域名，含泛域名 `*.example.com`）。
- 选择 DNS 服务商（如阿里云、腾讯云 DNSPod、Cloudflare 等），并填写对应的 API 凭据。
- 客户端通过 SSH 将凭据临时上传至服务器，执行 acme.sh 签发命令（`--issue --dns <provider>`）。
- 签发成功后，证书存储在服务器 acme.sh 默认目录。
- 支持将 DNS 凭据使用 `--save` 参数保存至 acme.sh 的 `account.conf`，以便后续自动续签使用。

#### FR-4 证书安装
- 用户可指定证书在服务器上的安装路径（如 `/etc/nginx/ssl/`）及文件名称。
- 客户端通过 SSH 执行 `acme.sh --install-cert`，将证书和私钥复制到指定路径。
- 用户可配置 reload 命令（如 `systemctl reload nginx`），用于证书更新后重载服务。
- 客户端应验证目标目录存在，若不存在可自动创建（或提示用户）。

#### FR-5 自动续签
- acme.sh 安装后会自动在 root 用户的 crontab 中添加每日续签任务（`acme.sh --cron`）。
- 客户端应检查该 cron 任务是否存在，若缺失则自动补加。
- 自动续签在服务器本地执行，无需客户端参与。
- 用户可通过客户端查看证书到期时间，并手动触发立即续签（`--renew --force`）。

#### FR-6 状态监控
- 客户端可查询服务器上已管理的证书列表（`acme.sh --list`）并解析显示。
- 显示证书域名、到期时间、签发时间、存储路径等信息。
- 对于即将到期的证书（例如小于 30 天），前端给出醒目提示。

#### FR-7 日志查看
- 客户端可获取最近一次 acme.sh 操作日志（签发、续签）用于排错。

### 2.2 非功能需求

#### NFR-1 安全性
- DNS API 凭据和 SSH 密钥在传输过程中必须加密（SSH 隧道）。
- 凭据不应以明文形式长期存储在客户端，建议使用操作系统钥匙串加密。
- 临时上传到服务器的凭据文件应在使用后立即删除。
- acme.sh 配置文件（`account.conf`）权限应设为 600。

#### NFR-2 可靠性
- 自动安装和签发过程中，任何步骤失败应给出明确错误提示，并支持重试。
- 对于网络不稳定情况，应设置合理的超时和重试机制。
- 客户端异常退出不应影响服务器端已配置的自动续签。

#### NFR-3 跨平台
- 客户端支持 Windows、macOS、Linux。
- 服务器端仅支持主流 Linux 发行版（acme.sh 限制），需在文档中注明。

#### NFR-4 易用性
- 提供友好的向导式界面，引导用户完成配置。
- 尽可能自动化，减少用户手动操作（如自动生成 reload 命令）。

#### NFR-5 性能
- 操作响应时间取决于网络和 acme.sh 执行时间，应在 30 秒内完成常规签发（不含 DNS 传播等待）。
- 状态查询应轻量，避免频繁远程执行命令。

---

## 3. 系统设计

### 3.1 整体架构
```
┌──────────────────────────── Wails3 桌面应用 ────────────────────────────┐
│  前端 (HTML/JS)                                                       │
│  ├─ 服务器管理界面（复用现有）                                        │
│  ├─ 证书管理页签：                                                    │
│  │   ├─ 证书列表（域名、到期时间、状态）                              │
│  │   ├─ 添加证书向导（服务器、域名、DNS 凭据、安装路径）              │
│  │   └─ 操作按钮（立即续签、查看日志）                                │
│  └─ 设置页（默认 CA、超时等）                                         │
└───────────────────────────────┬───────────────────────────────────────┘
                                │ 绑定调用 (Go bindings)
┌───────────────────────────────▼───────────────────────────────────────┐
│  Go 后端                                                              │
│  ├─ SSH 管理模块（已有）                                              │
│  ├─ 证书管理器 (CertManager)                                          │
│  │   ├─ acme.sh 安装检测与部署                                        │
│  │   ├─ 远程命令构造与执行                                            │
│  │   ├─ 配置解析与状态解析                                            │
│  │   └─ 临时文件清理                                                  │
│  ├─ 加密存储模块 (SecretStore)                                        │
│  └─ 事件通知 (Wails events)                                           │
└───────────────────────────────┬───────────────────────────────────────┘
                                │ SSH
┌───────────────────────────────▼───────────────────────────────────────┐
│  远程 Linux 服务器                                                   │
│  ├─ acme.sh (安装在 /root/.acme.sh/)                                  │
│  ├─ cron 定时任务 (每日 acme.sh --cron)                               │
│  ├─ 证书文件 (如 /etc/nginx/ssl/*.crt, *.key)                        │
│  └─ Web 服务器 (Nginx/Apache)                                        │
└──────────────────────────────────────────────────────────────────────┘
```

### 3.2 模块划分

| 模块 | 职责 |
|------|------|
| **SSH Client** | 复用现有 SSH 连接、命令执行、SFTP 文件传输能力。 |
| **CertManager** | 核心业务逻辑：检测环境、构建 acme.sh 命令、执行远程操作、解析结果。 |
| **SecretStore** | 加密存储 DNS API 凭据、SSH 密钥等敏感信息，对接系统钥匙串。 |
| **ConfigManager** | 管理客户端本地配置（已配置的服务器、证书任务等），使用 JSON 配置文件。 |
| **UI** | 前端界面，调用 Go 后端绑定方法，展示状态。 |

### 3.3 关键流程

#### 3.3.1 添加证书流程
1. 用户在前端选择目标服务器，填写域名列表、DNS 服务商及凭据、安装路径。
2. 前端调用后端 `AddCertificate` 方法，传入相关参数。
3. 后端建立 SSH 连接，执行环境检测：
   - 检查 acme.sh 是否安装，若未安装则执行安装命令。
   - 检查目标目录是否存在，不存在则创建。
4. 后端构造 DNS 环境变量导出命令，通过 SSH 执行 acme.sh 签发命令，添加 `--save` 保存凭据。
5. 签发成功后，执行 `--install-cert` 安装证书并设置 reload 命令。
6. 验证 cron 任务存在，若缺失则添加。
7. 后端返回结果给前端，前端刷新证书列表。

#### 3.3.2 自动续签流程（服务器端）
1. 服务器 cron 每天执行 `acme.sh --cron`。
2. acme.sh 检查所有已管理证书的到期时间。
3. 对即将到期（<60天）的证书自动续签。
4. 续签成功后，自动执行 `--install-cert` 时保存的 reload 命令。
5. 若续签失败，acme.sh 会输出错误日志，客户端可通过状态查询得知。

#### 3.3.3 状态查询流程
1. 用户点击"刷新"或进入证书页签。
2. 后端通过 SSH 执行 `acme.sh --list` 获取简要列表。
3. 解析输出，提取域名、到期时间等信息。
4. 如需详细到期时间，可读取证书文件（`openssl x509 -enddate`）。
5. 返回数据给前端渲染。

---

## 4. 技术选型

### 4.1 服务器端 ACME 客户端：acme.sh
- 纯 Shell，无需额外依赖，适合远程部署。
- 支持 150+ DNS 服务商，满足泛域名需求。
- 自带 cron 自动续签和 reload 钩子，契合"定时任务在服务器"的要求。
- 安装简单，一条命令即可完成。

### 4.2 SSH 通信：golang.org/x/crypto/ssh
- 已有模块可复用。
- 支持密码、公钥认证，SFTP 用于文件上传（临时凭据文件或安装包）。

### 4.3 敏感信息存储：go-keyring
- 跨平台钥匙串访问（macOS Keychain、Windows Credential Manager、Linux Secret Service）。
- 用于加密存储 DNS API 密钥，避免明文写入配置文件。

### 4.4 前端框架：Wails3 内置 Web 技术
- 使用 React/Vue 等，根据现有项目技术栈。

---

## 5. 详细实现方案

### 5.1 远程 acme.sh 自动安装

**检测脚本**：
```bash
test -x /root/.acme.sh/acme.sh && echo "installed" || echo "not_installed"
```

**安装命令**：
```bash
curl -fsSL https://get.acme.sh | sh -s email=<user_email>
```
如果服务器无法访问外网，可通过 SFTP 上传本地 acme.sh 压缩包，然后解压并执行安装脚本。

**设置默认 CA**（可选，推荐）：
```bash
/root/.acme.sh/acme.sh --set-default-ca --server letsencrypt
```

### 5.2 DNS 凭据注入

为避免凭据出现在命令行历史中，采用以下方式：

1. 将凭据以环境变量形式写入临时文件（如 `/tmp/acme_dns_env.sh`），权限 600。
2. 通过 SSH 上传该文件，然后执行：
   ```bash
   source /tmp/acme_dns_env.sh && /root/.acme.sh/acme.sh --issue --dns <provider> -d <domains> --save
   ```
3. 执行完毕后立即删除临时文件：
   ```bash
   rm -f /tmp/acme_dns_env.sh
   ```

**临时文件内容示例**（阿里云）：
```bash
export Ali_Key="LTAI5t..."
export Ali_Secret="vGgY..."
```

### 5.3 签发命令构建

```bash
/root/.acme.sh/acme.sh --issue \
  --dns dns_ali \
  -d example.com \
  -d '*.example.com' \
  --dnssleep 120 \
  --save
```

`dns_ali` 根据用户选择的 DNS 服务商动态生成（映射表：`aliyun -> dns_ali`, `dnspod -> dns_dp`, `cloudflare -> dns_cf` 等）。

### 5.4 证书安装命令

```bash
/root/.acme.sh/acme.sh --install-cert -d example.com \
  --key-file /etc/nginx/ssl/example.com.key \
  --fullchain-file /etc/nginx/ssl/example.com.crt \
  --reloadcmd "systemctl reload nginx"
```

- 用户可自定义 reload 命令，客户端可根据 Web 服务器类型自动生成（Nginx、Apache、Caddy 等）。
- 确保目标目录存在：`mkdir -p <dir>`。

### 5.5 自动续签保障

- 确认 crontab 中有 acme.sh 条目：
  ```bash
  crontab -l | grep -q acme.sh && echo "exists" || (crontab -l; echo '0 0 * * * "/root/.acme.sh"/acme.sh --cron --home "/root/.acme.sh" > /dev/null') | crontab -
  ```
- acme.sh 在安装时通常已自动添加，客户端只需检查并补加。

### 5.6 状态解析

`acme.sh --list` 输出示例：
```
Main_Domain  KeyLength  SAN_Domains  Created  Renew
example.com  ec-256     www.example.com,*.example.com  2025-01-01  2025-03-01
```

客户端解析该输出，并可通过读取证书文件获取精确 `NotAfter`：
```bash
openssl x509 -enddate -noout -in /etc/nginx/ssl/example.com.crt
```

### 5.7 错误处理与重试

- SSH 命令执行失败：返回错误信息，前端展示。
- acme.sh 签发失败：解析 stderr，提示常见错误（如 DNS 验证超时、凭据错误）。
- 对 DNS 传播等待，acme.sh 内置重试，可配置 `--dnssleep` 和 `--dns-check-tries`。

---

## 6. 安全设计

### 6.1 传输安全
- 所有远程操作均在 SSH 加密通道内进行。
- 推荐使用密钥认证，密码可通过钥匙串加密保存。

### 6.2 凭据安全
- DNS API 凭据不存储在客户端配置文件中，仅在签发时由用户输入或从钥匙串临时读取。
- 服务器端凭据由 acme.sh 的 `--save` 功能加密存储于 `account.conf`，权限 600。
- 临时环境文件用完即删。

### 6.3 权限控制
- acme.sh 安装目录及证书文件路径均以 root 执行（SSH 连接需具备 root 权限或 sudo）。
- 客户端可提示用户使用具有 sudo 权限的账号登录。

### 6.4 日志脱敏
- 客户端日志中禁止输出完整 API 密钥、密码等。
- 远程命令输出中如包含敏感信息，需过滤后再展示。

---

## 7. 测试方案

### 7.1 单元测试
- 命令构建函数：验证不同 DNS 服务商、域名组合生成的命令正确。
- 状态解析函数：构造 `acme.sh --list` 输出，验证解析结果。
- SSH 执行封装：模拟执行器，测试错误处理。

### 7.2 集成测试
- 搭建测试服务器（虚拟机），运行客户端执行完整流程。
- 使用 Let's Encrypt staging 环境，避免频率限制。
- 验证自动续签：手动修改系统时间或强制续签，检查 cron 是否正常工作。
- 测试网络不稳定场景：断开 SSH 连接，检查错误提示与重试。

### 7.3 安全测试
- 检查临时文件是否被删除。
- 检查服务器上 `account.conf` 权限是否为 600。
- 检查客户端日志无敏感信息泄露。

---

## 8. 部署与维护

### 8.1 客户端分发
- 将证书管理功能作为 Wails3 应用的一部分随版本发布。
- 更新 acme.sh 支持的 DNS 服务商映射表时，需同步更新客户端。

### 8.2 服务器端维护
- acme.sh 版本更新：可提供手动升级按钮（`acme.sh --upgrade`）。
- 证书监控：客户端定期提醒用户查看即将到期的证书。

### 8.3 文档
- 提供用户指南，说明支持的操作系统、DNS 服务商、注意事项。

---

## 9. 总结

本方案通过在 SSH 客户端中集成 acme.sh 远程管理能力，实现了 HTTPS 证书的自动化部署与续签。服务器端独立运行 acme.sh 并配置 cron 任务，确保证书续签不依赖客户端在线。客户端提供友好的向导界面和状态监控，降低了运维门槛。同时，方案充分考虑了安全性、可靠性和跨平台兼容性，为后续扩展（如支持更多 DNS 服务商、支持 HTTP 验证等）预留了空间。

</details>
