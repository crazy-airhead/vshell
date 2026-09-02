---
layout: home

hero:
  name: vShell
  text: 桌面 SSH 客户端管理工具
  tagline: Wails 3（Go + Vue 3）· 纯本地应用 —— 无云端、无账号、无遥测
  image:
    src: /logo-512.png
    alt: vShell
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/getting-started
    - theme: alt
      text: 使用指南
      link: /guide/
    - theme: alt
      text: GitHub
      link: https://github.com/crazy-airhead/vshell

features:
  - icon: 🌳
    title: 连接管理
    details: 分组树形组织连接，支持拖拽调整归属；密码 / 私钥 / 交互式三种认证，敏感信息 AES-256-GCM 加密存储。
  - icon: 🖥️
    title: 终端仿真
    details: 基于 xterm.js 的 PTY 终端（xterm-256color），WebGL 渲染、右键菜单复制粘贴、断开后按回车即刻重连。
  - icon: 📁
    title: SFTP 文件管理
    details: 远端目录树 + 双栏文件浏览，上传 / 下载 / 删除，支持从系统文件浏览器拖入上传与应用内双向拖拽，实时进度跟踪。
  - icon: ✏️
    title: 远程文件编辑
    details: Monaco Editor 直接编辑远程文件（⌘S 保存写回服务器），按扩展名智能识别 40+ 种语言。
  - icon: 📊
    title: 服务器监控
    details: CPU / 内存 / 磁盘 / 进程 / 网络全覆盖，ECharts 网卡流量趋势图，复用 SSH 连接免重复认证。
  - icon: 🔑
    title: SSH 密钥管理
    details: 生成（Ed25519 / RSA / ECDSA）、导入、管理 SSH 密钥对；被连接引用的密钥受删除保护。
  - icon: 🔀
    title: 端口转发
    details: 本地端口转发（SSH 隧道），按连接分组管理，内置常用服务预设（MySQL / Redis / PostgreSQL…），支持自动启动。
  - icon: 📝
    title: SSH Config 导入
    details: 结构化编辑 ~/.ssh/config，一键导入主机为连接（自动识别 IdentityFile）。
  - icon: 🌍
    title: 中英双语
    details: vue-i18n 完整国际化，浅色 / 深色主题与 7 套终端配色随心切换。
---
