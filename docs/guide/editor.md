# 远程文件编辑

vShell 内置 Monaco Editor，可以直接编辑**远程文件**（经 SFTP 读写）与**本地文件**，保存即写回。

---

## 1. 打开文件

在 [SFTP 面板](sftp.md)中**双击**文件即可打开：

| 来源 | 标签前缀 | 读取 | 保存（⌘S / Ctrl+S） |
|------|----------|------|----------------------|
| 远端文件 | `[远端]` + 蓝 | `SFTPReadFileContent` | `SFTPWriteFileContent` 写回服务器 |
| 本地文件 | `[本地]` + 绿 | `ReadLocalFileContent` | `WriteLocalFileContent` 写回本机 |
| `~/.ssh/config` | 编辑器标签 | `ReadSSHConfigRaw` | `WriteSSHConfigRaw`，并刷新 SSH 配置面板 |

- 标签悬停 Tooltip 显示完整路径（远端为 `user@host:/path`）
- 同一文件重复打开时直接切换到已有标签，不会开第二个

### 可编辑性限制

| 限制 | 说明 |
|------|------|
| 大小上限 | **5 MB**（远端与本地相同），超过提示「文件过大，不支持编辑」 |
| 二进制类型 | 按扩展名黑名单拦截（图片 / 压缩包 / 可执行文件等），提示「该文件类型不支持编辑」 |

## 2. 语言识别

按扩展名自动检测，覆盖 40+ 种语言：`ts / js / json / html / css / scss / vue / md / yaml / xml / go / py / java / c / cpp / rust / ruby / php / sh / sql / toml / dockerfile / tf / lua / swift / kt / scala / r / dart` 等；对 `Makefile`、`Dockerfile`、`CMakeLists.txt` 做文件名特判，其余兜底纯文本。

## 3. 编辑器行为

- 主题跟随应用明暗（`vs-dark` / `vs`），字号 13、自动换行、显示行号、**无 minimap**
- 内容一有修改，标签标题出现橙色 `•`（未保存标记）
- `⌘S` 保存当前标签；原生菜单 File → Save（同样 `⌘S`）只对**当前激活**的编辑器标签生效

## 4. 注意事项

- 关闭有未保存修改的标签**没有确认提示**，改动会直接丢失——养成先保存的习惯
- 无「另存为」能力
- 保存采用整文件覆写（`Create` + 单次 `Write`），并非增量 diff

---

## 延伸阅读

- [SFTP 文件管理](sftp.md) —— 文件的浏览入口
- [~/.ssh/config 导入导出](ssh-config.md) —— config 原文编辑
