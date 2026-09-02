import { defineConfig } from 'vitepress'

// 站点内容：docs/guide（使用指南）+ docs/dev（开发文档）。
// docs/issues 与 **/_*.md（写作规范等）通过 srcExclude 排除，不发布到站点；
// prompt.md / default.json / dark.json 为开发内部文件，同样不发布。
export default defineConfig({
  lang: 'zh-CN',
  title: 'vShell',
  description:
    '纯本地的桌面 SSH 客户端管理工具 —— Wails 3（Go + Vue 3），终端 / SFTP / 监控 / 密钥管理，无云端、无账号',
  base: '/vshell/',
  lastUpdated: true,
  sitemap: { hostname: 'https://crazy-airhead.github.io/vshell/' },
  srcExclude: ['**/issues/**', '**/_*.md', 'prompt.md', '*.json'],
  // head 标签不会自动加 base 前缀，favicon 需写全路径
  head: [['link', { rel: 'icon', href: '/vshell/logo.png' }]],
  themeConfig: {
    logo: '/logo.png',
    nav: [
      { text: '首页', link: '/' },
      { text: '使用指南', link: '/guide/', activeMatch: '/guide/' },
      { text: '开发文档', link: '/dev/', activeMatch: '/dev/' },
    ],
    sidebar: {
      '/guide/': [
        {
          text: '开始',
          collapsed: false,
          items: [
            { text: '指南总览', link: '/guide/' },
            { text: '快速开始', link: '/guide/getting-started' },
          ],
        },
        {
          text: '连接与认证',
          collapsed: false,
          items: [
            { text: '连接与分组管理', link: '/guide/connections' },
            { text: 'SSH 密钥管理', link: '/guide/keys' },
            { text: '~/.ssh/config 导入导出', link: '/guide/ssh-config' },
          ],
        },
        {
          text: '终端',
          collapsed: false,
          items: [
            { text: '终端使用', link: '/guide/terminal' },
            { text: '远程文件编辑', link: '/guide/editor' },
          ],
        },
        {
          text: '文件与传输',
          collapsed: false,
          items: [
            { text: 'SFTP 文件管理', link: '/guide/sftp' },
          ],
        },
        {
          text: '监控与网络',
          collapsed: false,
          items: [
            { text: '服务器监控', link: '/guide/monitor' },
            { text: '端口转发', link: '/guide/port-forwarding' },
          ],
        },
        {
          text: '个性化',
          collapsed: false,
          items: [
            { text: '设置与主题', link: '/guide/settings' },
          ],
        },
      ],
      '/dev/': [
        {
          text: '开发',
          collapsed: false,
          items: [
            { text: '开发文档总览', link: '/dev/' },
            { text: '架构总览', link: '/dev/architecture' },
            { text: '终端 I/O 数据流', link: '/dev/terminal-io' },
            { text: '数据存储与加密', link: '/dev/storage-crypto' },
            { text: '构建与开发环境', link: '/dev/development' },
          ],
        },
      ],
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/crazy-airhead/vshell' },
    ],
    editLink: {
      pattern: 'https://github.com/crazy-airhead/vshell/edit/main/docs/:path',
      text: '在 GitHub 上编辑此页',
    },
    search: {
      provider: 'local',
    },
    outline: { level: [2, 3], label: '本页目录' },
    docFooter: { prev: '上一篇', next: '下一篇' },
    lastUpdated: { text: '最后更新' },
    returnToTopLabel: '回到顶部',
    sidebarMenuLabel: '菜单',
    darkModeSwitchLabel: '主题',
    lightModeSwitchTitle: '切换到浅色',
    darkModeSwitchTitle: '切换到深色',
    footer: {
      message: '纯本地应用 · 无云端 · 无遥测',
      copyright: 'Copyright © 2026 crazy-airhead',
    },
  },
})
