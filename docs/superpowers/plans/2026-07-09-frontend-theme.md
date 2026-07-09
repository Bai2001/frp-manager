# 前端双主题 UI 改版实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 为 FRP Manager 客户端实现浅色 / 深色双主题：默认跟随系统，设置页可手动覆盖；浅色侧栏同步浅色；对齐 Element Plus 官方暗色；表格 / 日志 / 设置页样式变量化。

**架构：** 后端 `settings.Settings` 增加 `theme_mode` 并经现有 `GetSettings` / `SaveSettings` 持久化。前端新增 `useTheme` 解析 `system|light|dark`，切换 `html.dark`，并引入 EP `dark/css-vars.css`。全局 CSS 令牌覆盖侧栏、内容区、卡片、表格；日志终端区保持深色底板。

**技术栈：** Go settings 包、Vue 3 + TypeScript + Pinia、Element Plus 2.x 暗色 CSS 变量、Wails v3 绑定。

**约定（覆盖本仓库习惯）：**
- 完整功能完成后再 git commit，不要中途频繁提交。
- 不要运行打包验证（除非用户明确要求）；后端用 `go test`，前端以类型/页面静态检查为主。
- 前端代码风格：4 空格、单引号、无分号、ES5 尾逗号。
- 产物与注释使用中文。

**规格：** `docs/superpowers/specs/2026-07-09-frontend-theme-design.md`

---

## 文件结构

**修改：**
- `client/internal/settings/settings.go` — 增加 `ThemeMode`
- `client/internal/settings/settings_test.go` — 往返与缺省覆盖
- `client/frontend/src/api/index.ts` — `Settings.theme_mode`
- `client/frontend/src/stores/settings.ts` — 默认值 `system`
- `client/frontend/src/main.ts` — 引入 EP 暗色 CSS；启动时加载并应用主题
- `client/frontend/src/style.css` — `:root` / `html.dark` 双套令牌
- `client/frontend/src/App.vue` — 侧栏改用变量（浅侧栏 / 深侧栏）
- `client/frontend/src/views/ServersView.vue` — 表格/空状态硬编码色改变量
- `client/frontend/src/views/TunnelsView.vue` — 同上 + 停止态灰点
- `client/frontend/src/views/LogsView.vue` — 筛选栏/徽章随主题；终端保持深色
- `client/frontend/src/views/SettingsView.vue` — 外观主题单选

**新建：**
- `client/frontend/src/composables/useTheme.ts` — 解析、应用、监听系统主题

**可选（实现时按需）：**
- 重新生成 Wails bindings 中的 `settings/models.ts`（若项目有 generate 流程）；前端以 `src/api/index.ts` 自维护类型为准，不依赖必须重生成。

**不改：**
- `SaveSettings` 窗口状态保留逻辑（已存在，`theme_mode` 随 `in` 正常写入即可）
- 服务器 / 映射业务 API、对话框表单结构

---

### 任务 1：后端 `theme_mode` 字段与测试

**文件：**
- 修改：`client/internal/settings/settings.go`
- 修改：`client/internal/settings/settings_test.go`

- [ ] **步骤 1：编写失败测试（theme 往返 + 缺省）**

在 `settings_test.go` 追加：

```go
func TestStoreThemeModeRoundtrip(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "settings.json"))
	in := Settings{
		CloseToTray:      true,
		ThemeMode:        "dark",
		LogRetentionDays: 3,
	}
	if err := s.Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.ThemeMode != "dark" {
		t.Errorf("ThemeMode want dark, got %q", got.ThemeMode)
	}
}

func TestStoreThemeModeDefaultEmpty(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "settings.json"))
	// 不写 ThemeMode，模拟旧配置
	if err := s.Save(Settings{CloseToTray: true}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.ThemeMode != "" {
		t.Errorf("旧配置缺字段应为空字符串，got %q", got.ThemeMode)
	}
}
```

- [ ] **步骤 2：运行测试，确认字段尚未存在时失败或编译失败**

```powershell
cd client
go test ./internal/settings/ -count=1
```

预期：编译失败（`ThemeMode` 未定义）或相关失败。

- [ ] **步骤 3：实现字段**

在 `Settings` 结构体中、`ConfigDir` 字段后增加：

```go
// ThemeMode 外观主题：system | light | dark。
// 空字符串或未知值由前端按 system 处理（兼容旧配置）。
ThemeMode string `json:"theme_mode"`
```

- [ ] **步骤 4：再跑测试**

```powershell
cd client
go test ./internal/settings/ -count=1
```

预期：PASS。

---

### 任务 2：前端类型、store 默认值、useTheme

**文件：**
- 修改：`client/frontend/src/api/index.ts`
- 修改：`client/frontend/src/stores/settings.ts`
- 新建：`client/frontend/src/composables/useTheme.ts`

- [ ] **步骤 1：扩展 Settings 类型**

在 `api/index.ts` 的 `Settings` 接口中增加：

```typescript
export type ThemeMode = 'system' | 'light' | 'dark'

export interface Settings {
    close_to_tray: boolean
    auto_start: boolean
    log_retention_days: number
    config_dir: string
    /** 外观主题：system | light | dark；缺省按 system */
    theme_mode: ThemeMode
    window_maximised?: boolean
    window_x?: number
    window_y?: number
    window_width?: number
    window_height?: number
}
```

- [ ] **步骤 2：store 默认值**

`settings.ts` 初始对象增加 `theme_mode: 'system'`。`load()` 后若后端返回空 `theme_mode`，规范为 `'system'`：

```typescript
settings.value = await api.getSettings()
if (!settings.value.theme_mode) {
    settings.value.theme_mode = 'system'
}
```

- [ ] **步骤 3：实现 useTheme**

新建 `client/frontend/src/composables/useTheme.ts`：

```typescript
import type { ThemeMode } from '@/api'

const MEDIA = '(prefers-color-scheme: dark)'

/**
 * 将 theme_mode 解析为最终 light/dark
 */
export function resolveTheme(mode: ThemeMode | string | undefined): 'light' | 'dark' {
    const m = mode === 'light' || mode === 'dark' || mode === 'system' ? mode : 'system'
    if (m === 'light' || m === 'dark') return m
    if (typeof window !== 'undefined' && window.matchMedia) {
        return window.matchMedia(MEDIA).matches ? 'dark' : 'light'
    }
    return 'light'
}

/**
 * 给 html 根节点切换 dark class（Element Plus 暗色依赖 html.dark）
 */
export function applyTheme(mode: ThemeMode | string | undefined) {
    const resolved = resolveTheme(mode)
    document.documentElement.classList.toggle('dark', resolved === 'dark')
    return resolved
}

let mediaHandler: ((e: MediaQueryListEvent) => void) | null = null
let mediaList: MediaQueryList | null = null

/**
 * 按 theme_mode 应用主题；system 时监听系统变化
 */
export function watchTheme(mode: ThemeMode | string | undefined) {
    stopWatchTheme()
    applyTheme(mode)
    const m = mode === 'light' || mode === 'dark' || mode === 'system' ? mode : 'system'
    if (m !== 'system' || typeof window === 'undefined' || !window.matchMedia) return
    mediaList = window.matchMedia(MEDIA)
    mediaHandler = () => applyTheme('system')
    mediaList.addEventListener('change', mediaHandler)
}

export function stopWatchTheme() {
    if (mediaList && mediaHandler) {
        mediaList.removeEventListener('change', mediaHandler)
    }
    mediaList = null
    mediaHandler = null
}
```

---

### 任务 3：main.ts 引入 EP 暗色 + 启动应用主题

**文件：**
- 修改：`client/frontend/src/main.ts`

- [ ] **步骤 1：改写 main.ts**

```typescript
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import App from './App.vue'
import router from './router'
import './style.css'
import { useSettingsStore } from '@/stores/settings'
import { watchTheme } from '@/composables/useTheme'

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)
app.use(router)
app.use(ElementPlus)
app.mount('#app')

// 启动后加载设置并应用主题（避免阻塞 mount）
const settingsStore = useSettingsStore()
settingsStore.load().then(() => {
    watchTheme(settingsStore.settings.theme_mode)
}).catch(() => {
    watchTheme('system')
})
```

注意：若 `load` 在设置页也会调用，重复 load 可接受；主题以 store 当前值为准。

---

### 任务 4：全局 CSS 双套令牌

**文件：**
- 修改：`client/frontend/src/style.css`

- [ ] **步骤 1：重写 style.css 令牌区**

保留滚动条与字体设置，替换 `:root` 并新增 `html.dark`。参考实现：

```css
:root {
    --brand-gradient: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    --brand-color: #667eea;

    /* 浅色侧栏 */
    --sidebar-bg: #f8fafc;
    --sidebar-bg-soft: #f1f5f9;
    --sidebar-fg: #334155;
    --sidebar-fg-muted: #94a3b8;
    --sidebar-active-bg: rgba(99, 102, 241, 0.1);
    --sidebar-active-bar: linear-gradient(180deg, #667eea 0%, #764ba2 100%);
    --sidebar-border: #e5e7eb;
    --sidebar-hover-bg: rgba(15, 23, 42, 0.04);

    --content-bg: #f4f5f7;
    --content-fg: #1f2937;
    --content-fg-secondary: #6b7280;

    --card-bg: #ffffff;
    --card-border: transparent;
    --card-radius: 12px;
    --card-shadow: 0 1px 3px rgba(0, 0, 0, 0.06), 0 1px 2px rgba(0, 0, 0, 0.04);
    --card-shadow-hover: 0 8px 24px rgba(0, 0, 0, 0.08);

    --table-header-bg: #fafbfc;
    --table-row-hover: #f0f5ff;
    --empty-icon: #d3d6db;
    --status-stopped: #d1d5db;

    --success: #10b981;
    --warning: #f59e0b;
    --danger: #ef4444;
    --info: #3b82f6;

    /* 日志终端：浅色界面下仍保持深色底板 */
    --terminal-bg: #1a1b26;
    --terminal-bg-soft: #16161e;
    --terminal-fg: #c0caf5;
}

html.dark {
    --brand-gradient: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    --brand-color: #818cf8;

    --sidebar-bg: #0c0e14;
    --sidebar-bg-soft: #12141c;
    --sidebar-fg: #cdd6f4;
    --sidebar-fg-muted: #6c7086;
    --sidebar-active-bg: rgba(129, 140, 248, 0.14);
    --sidebar-active-bar: linear-gradient(180deg, #89b4fa 0%, #cba6f7 100%);
    --sidebar-border: #1f2430;
    --sidebar-hover-bg: rgba(255, 255, 255, 0.04);

    --content-bg: #14161d;
    --content-fg: #e5e7eb;
    --content-fg-secondary: #9ca3af;

    --card-bg: #1a1d27;
    --card-border: #2a2f3a;
    --card-shadow: none;
    --card-shadow-hover: 0 0 0 1px rgba(255, 255, 255, 0.06);

    --table-header-bg: #1c2030;
    --table-row-hover: rgba(129, 140, 248, 0.08);
    --empty-icon: #4b5563;
    --status-stopped: #4b5563;

    --success: #34d399;
    --warning: #fbbf24;
    --danger: #f87171;
    --info: #60a5fa;

    --terminal-bg: #0f1117;
    --terminal-bg-soft: #0c0e14;
    --terminal-fg: #c0caf5;
}

html, body {
    margin: 0;
    padding: 0;
    height: 100%;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC',
        'Hiragino Sans GB', 'Microsoft YaHei', Roboto, 'Helvetica Neue', sans-serif;
    color: var(--content-fg);
    background: var(--content-bg);
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
}

#app {
    height: 100vh;
}

::-webkit-scrollbar {
    width: 8px;
    height: 8px;
}
::-webkit-scrollbar-track {
    background: transparent;
}
::-webkit-scrollbar-thumb {
    background: rgba(128, 128, 128, 0.35);
    border-radius: 4px;
}
::-webkit-scrollbar-thumb:hover {
    background: rgba(128, 128, 128, 0.5);
}
```

---

### 任务 5：侧栏 App.vue 适配浅/深侧栏

**文件：**
- 修改：`client/frontend/src/App.vue`

- [ ] **步骤 1：替换侧栏相关 scoped 样式中的写死色**

关键改动点（保持结构不变）：

```css
.aside {
    background: var(--sidebar-bg);
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--sidebar-border);
}

.brand {
    /* ... */
    border-bottom: 1px solid var(--sidebar-border);
}

.brand-title {
    color: var(--sidebar-fg);
}

.brand-subtitle {
    color: var(--sidebar-fg-muted);
}

.side-menu-item {
    color: var(--sidebar-fg);
    /* ... */
}

.side-menu-item:hover {
    background: var(--sidebar-hover-bg);
    color: var(--sidebar-fg);
}

.side-menu-item.is-active {
    background: var(--sidebar-active-bg);
    color: var(--sidebar-fg);
}

.side-menu-item.is-active::before {
    background: var(--sidebar-active-bar);
}

.aside-footer {
    border-top: 1px solid var(--sidebar-border);
}

.version {
    color: var(--sidebar-fg-muted);
}

/* 折叠按钮文字颜色 */
.collapse-btn {
    color: var(--sidebar-fg-muted);
}
```

若 `el-menu` 在浅色下仍用 EP 默认深色文字冲突，增加：

```css
.side-menu {
    --el-menu-bg-color: transparent;
    --el-menu-text-color: var(--sidebar-fg);
    --el-menu-hover-bg-color: var(--sidebar-hover-bg);
    --el-menu-active-color: var(--sidebar-fg);
}
```

主内容区：

```css
.main {
    background: var(--content-bg);
    /* 保持现有 padding/overflow */
}
```

---

### 任务 6：ServersView / TunnelsView 表格变量化

**文件：**
- 修改：`client/frontend/src/views/ServersView.vue`
- 修改：`client/frontend/src/views/TunnelsView.vue`

- [ ] **步骤 1：ServersView 表格与空状态**

将：

```css
:deep(.modern-table) {
    --el-table-border-color: transparent;
    --el-table-header-bg-color: #fafbfc;
}
:deep(.modern-table .el-table__row:hover > td) {
    background: #f0f5ff !important;
}
.empty-icon {
    color: #d3d6db;
}
```

改为：

```css
.table-card {
    background: var(--card-bg);
    border: 1px solid var(--card-border);
    /* 保留 radius/shadow 变量 */
}
:deep(.modern-table) {
    --el-table-border-color: transparent;
    --el-table-header-bg-color: var(--table-header-bg);
    --el-table-bg-color: var(--card-bg);
    --el-table-tr-bg-color: var(--card-bg);
    --el-table-text-color: var(--content-fg);
    --el-table-header-text-color: var(--content-fg);
    --el-fill-color-blank: var(--card-bg);
}
:deep(.modern-table .el-table__row:hover > td) {
    background: var(--table-row-hover) !important;
}
.empty-icon {
    color: var(--empty-icon);
}
```

- [ ] **步骤 2：TunnelsView 同样替换**，并改：

```css
.status-dot.stopped {
    background: var(--status-stopped);
}
```

`table-card` 同步加 `background` / `border` 变量。

---

### 任务 7：LogsView 随主题工具栏

**文件：**
- 修改：`client/frontend/src/views/LogsView.vue`

- [ ] **步骤 1：统计徽章与筛选栏改用变量**

将 `.stat-badge` 的写死白底/边框改为：

```css
.stat-badge {
    background: var(--card-bg);
    border: 1px solid var(--card-border, #ebeef5);
    color: var(--content-fg);
}
.stat-badge.active {
    background: var(--brand-color);
    border-color: var(--brand-color);
}
.stat-badge.active .stat-label,
.stat-badge.active .stat-count {
    color: #fff;
}
```

`.filter-summary`、页面标题已用 content 变量则保留。

- [ ] **步骤 2：确认 `.log-view` 仍使用 `--terminal-*` 变量**（两套主题下终端均深色即可，无需再改结构）。

若徽章 `border` 在浅色下 `var(--card-border)` 为 `transparent` 看不清，浅色令牌可把 `--card-border` 设为 `#e5e7eb`，深色保持 `#2a2f3a`；表格卡片用阴影 + 边框均可。

**推荐微调（任务 4 一并做）：** 浅色 `--card-border: #e5e7eb`，阴影可减弱；深色保持边框分层。

---

### 任务 8：SettingsView 主题选择 UI

**文件：**
- 修改：`client/frontend/src/views/SettingsView.vue`

- [ ] **步骤 1：引入 watchTheme**

```typescript
import { watchTheme } from '@/composables/useTheme'
import type { ThemeMode } from '@/api'
```

- [ ] **步骤 2：主题变更处理**

```typescript
async function onThemeChange(val: ThemeMode | string) {
    const mode = (val === 'light' || val === 'dark' || val === 'system' ? val : 'system') as ThemeMode
    store.settings.theme_mode = mode
    watchTheme(mode)
    saving.value = true
    try {
        await store.save()
        ElMessage.success('外观主题已保存')
    } finally {
        saving.value = false
    }
}
```

- [ ] **步骤 3：通用卡片内增加表单项**

放在「关闭时最小化到托盘」之前或之后：

```vue
<el-form-item label="外观主题">
    <el-radio-group
        :model-value="store.settings.theme_mode || 'system'"
        :disabled="saving"
        @change="onThemeChange"
    >
        <el-radio-button value="system">跟随系统</el-radio-button>
        <el-radio-button value="light">浅色</el-radio-button>
        <el-radio-button value="dark">深色</el-radio-button>
    </el-radio-group>
</el-form-item>
```

（若当前 Element Plus 版本 `el-radio-button` 使用 `label` 而非 `value`，按项目已有 EP 写法对齐；以能绑定 `system|light|dark` 为准。）

- [ ] **步骤 4：设置卡片背景变量化**

```css
.setting-card {
    background: var(--card-bg);
    border: 1px solid var(--card-border);
    /* 保留 radius/shadow */
}
```

---

### 任务 9：联调检查清单（不打包）

- [ ] **步骤 1：后端测试**

```powershell
cd client
go test ./internal/settings/ -count=1
```

预期：PASS。

- [ ] **步骤 2：静态扫硬编码**

在 `client/frontend/src` 内搜索页面样式中仍写死的浅色值（如 `#fafbfc`、`#f0f5ff`、`#d3d6db`、侧栏 `#fff` 标题色），能改则改。

- [ ] **步骤 3：手动冒烟（用户或 dev 环境）**

1. 设置 → 外观主题：浅色 / 深色 / 跟随系统，切换立即生效。  
2. 浅色：侧栏浅底深字，内容浅底，表格可读。  
3. 深色：侧栏深底，内容深底，EP 对话框/表格暗色。  
4. 日志终端两种主题下均为深色底板。  
5. 重启应用后主题保持。  
6. 删除 `theme_mode` 字段的旧 `settings.json` 行为等同跟随系统。

- [ ] **步骤 4：功能全部完成后再统一 commit（不要中途提交）**

建议 message（中文规范）：

```
feat(client): 前端浅色/深色双主题与跟随系统

增加 theme_mode 持久化，对齐 Element Plus 暗色变量，
侧栏与页面样式令牌化。
```

---

## 规格覆盖自检

| 规格项 | 任务 |
|--------|------|
| `theme_mode` 持久化 | 任务 1、2、8 |
| `html.dark` + EP 暗色 CSS | 任务 2、3 |
| 跟随系统 + 手动覆盖 | 任务 2、3、8 |
| 浅色侧栏 / 深色侧栏 | 任务 4、5 |
| 精致表格变量化 | 任务 6 |
| 日志终端始终深色 | 任务 4、7 |
| 设置页 UI | 任务 8 |
| 旧配置兼容 | 任务 1、2 |
| 不改业务逻辑 | 全文约定 |

## 占位符扫描

无 TODO / 待定 /「类似任务 N」步骤。
