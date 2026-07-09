# 前端双主题 UI 改版设计

> 日期：2026-07-09  
> 范围：浅色 / 深色双主题、跟随系统、侧栏与页面样式统一、设置持久化  
> 目标：在不改业务逻辑的前提下，把客户端 GUI 升级为可随系统切换的成套主题界面

## 背景

当前前端已有基础视觉（紫蓝品牌色、深色侧栏、浅色内容区、Element Plus 表格与日志终端），但存在以下问题：

1. **无深色内容区主题**：侧栏固定深色，内容区固定浅色，无法跟随系统。
2. **颜色写死**：表格 hover、表头背景、空状态等使用硬编码浅色值，无法适配暗色。
3. **Element Plus 未启用官方暗色**：仅引入默认 CSS，组件在深色背景下会不协调。
4. **设置无外观项**：无法手动覆盖系统主题。

本次定位为「显示层整体改版」，不改服务器 / 映射 / frpc 业务逻辑与 API 契约。

## 目标与非目标

### 目标

- 提供 **浅色 / 深色** 两套完整主题。
- 默认 **跟随系统**（`prefers-color-scheme`），设置页可选手动覆盖。
- **浅色模式下侧栏也是浅色**；深色模式下侧栏为深色。
- 与 **Element Plus 官方暗色 CSS 变量** 对齐。
- 保留现有 **左侧导航 + 右侧内容** 布局与 **表格** 信息呈现。
- 日志终端区 **始终偏深**（运维阅读习惯）。
- 主题偏好持久化到现有 `settings.json`。

### 非目标

- 不改为顶部导航或卡片列表布局。
- 不重写对话框表单结构（仅随 EP 暗色自动适配）。
- 不改后端业务 API、数据库 schema、frpc/frps 逻辑。
- 不做自定义主题色编辑器或多套品牌皮肤。

## 架构总览

```
settings.json  theme_mode: system|light|dark
        │
        ▼
  settings store / 主题 composable
        │
        ├─ 解析最终主题 resolved = system ? matchMedia : mode
        │
        ▼
document.documentElement.classList.toggle('dark', resolved === 'dark')
        │
        ├─ style.css  :root / html.dark  自定义令牌
        └─ element-plus dark/css-vars.css  组件令牌
```

| 层级 | 职责 |
|------|------|
| 后端 `settings.Settings` | 持久化 `theme_mode` |
| 前端 `api` / `settings` store | 读写设置，触发主题应用 |
| `useTheme`（或等价模块） | 解析 system/light/dark，监听系统变化，切换 `html.dark` |
| `style.css` | 自定义布局与页面令牌（侧栏、内容区、卡片、终端） |
| Element Plus 暗色 CSS | 表格、表单、对话框、按钮等组件色 |

## 详细设计

### 1. 设置字段

后端 `client/internal/settings/settings.go`：

```go
type Settings struct {
    // ... 现有字段 ...
    ThemeMode string `json:"theme_mode"` // system | light | dark，空或未知按 system
}
```

约定：

- 合法值：`system`、`light`、`dark`。
- 缺省 / 空字符串 / 非法值：按 `system` 处理（向后兼容旧配置）。
- 窗口状态字段仍由后端维护，前端保存设置时不得覆盖丢失（沿用现有 save 行为）。

前端 `Settings` 接口同步增加 `theme_mode: 'system' | 'light' | 'dark'`。

### 2. 主题切换机制

新增前端模块（建议 `src/composables/useTheme.ts` 或 `src/theme.ts`）：

1. **解析**：`theme_mode === 'system'` 时用 `window.matchMedia('(prefers-color-scheme: dark)')`；否则用显式 light/dark。
2. **应用**：`document.documentElement.classList.toggle('dark', isDark)`。
3. **监听**：`theme_mode === 'system'` 时订阅 `matchMedia` 的 `change`；切换为手动模式时移除监听。
4. **时机**：
   - 应用启动：设置加载完成后立即应用（避免闪白/闪黑可接受首次极短默认浅色）。
   - 设置页变更：立即应用 + 调用 `saveSettings`。
5. **与 Element Plus**：在 `main.ts` 增加  
   `import 'element-plus/theme-chalk/dark/css-vars.css'`  
   EP 官方暗色依赖 `html.dark`，与上述切换一致。

### 3. CSS 令牌

在 `client/frontend/src/style.css` 中定义两套变量。

#### 浅色（`:root`）

| 令牌 | 用途 | 建议值 |
|------|------|--------|
| `--brand-gradient` / `--brand-color` | 品牌 | 保持现有紫蓝 |
| `--sidebar-bg` | 侧栏背景 | `#f8fafc` |
| `--sidebar-fg` | 侧栏文字 | `#334155` |
| `--sidebar-fg-muted` | 次要文字 | `#94a3b8` |
| `--sidebar-active-bg` | 激活项背景 | 淡紫半透明 |
| `--sidebar-active-bar` | 激活左边条 | 品牌渐变或实色 |
| `--sidebar-border` | 侧栏右边框 | `#e5e7eb` |
| `--content-bg` | 主内容背景 | `#f4f5f7` |
| `--content-fg` / `--content-fg-secondary` | 正文 / 次要 | 现有深灰系 |
| `--card-bg` | 卡片背景 | `#ffffff` |
| `--card-border` | 卡片边框 | `#e5e7eb` 或透明 + 阴影 |
| `--card-shadow` / `--card-shadow-hover` | 阴影 | 现有浅阴影 |
| `--table-header-bg` | 表头 | `#fafbfc` |
| `--table-row-hover` | 行 hover | 淡紫浅底 |
| `--success` / `--warning` / `--danger` / `--info` | 语义色 | 保持现有 |
| `--terminal-bg` / `--terminal-bg-soft` / `--terminal-fg` | 日志终端 | 保持深色终端 |

#### 深色（`html.dark`）

| 令牌 | 建议方向 |
|------|----------|
| `--sidebar-bg` | `#0c0e14` |
| `--sidebar-fg` | 浅灰（如 `#cdd6f4` 系） |
| `--sidebar-border` | `#1f2430` |
| `--content-bg` | `#14161d` |
| `--content-fg` | 近白灰 |
| `--card-bg` | `#1a1d27` |
| `--card-border` | `#2a2f3a` |
| `--card-shadow` | 减弱或取消，靠边框分层 |
| `--table-header-bg` | 略深于卡片 |
| `--table-row-hover` | 半透明紫/白 |
| 终端变量 | 可与浅色相同或略调对比度 |

页面与布局组件 **禁止** 再写死浅色背景/文字色；一律改用变量。

### 4. 侧栏（`App.vue`）

- 背景、文字、边框、激活态、底部分割线全部使用侧栏令牌。
- 浅色：浅底 + 深字 + 淡紫激活；深色：深底 + 浅字 + 半透明激活。
- Logo 渐变两侧共用。
- 折叠行为与断点（≤900px）保持不变。

### 5. 表格页（服务器 / 映射）

- `modern-table` 的表头背景、hover、边框改为变量。
- 卡片容器使用 `--card-bg`、`--card-border`、`--card-shadow`。
- 空状态图标/文案使用 `--content-fg-secondary`。
- 映射状态点、远程 URL 品牌色使用变量；停止态灰点在深色下使用可见的中性色。
- **不改变** 列结构、操作按钮逻辑、轮询与 URL 拼接逻辑。

### 6. 日志页

- **终端区域**（`.log-view`）在浅色与深色下均使用深色终端底板（令牌可共享）。
- 页面标题区、筛选栏、统计徽章：随主题使用内容区/卡片令牌。
- 激活筛选徽章：品牌色底 + 对比文字；未激活随主题边框与背景。
- 级别色使用语义变量，避免仅在浅色下可读。

### 7. 设置页

在「通用」卡片中增加外观主题：

- 控件：`el-radio-group` 或等价单选，三项：
  - 跟随系统（`system`）
  - 浅色（`light`）
  - 深色（`dark`）
- 变更流程：更新 store → 立即 `applyTheme` → `saveSettings` → 成功提示。
- 其余开关/导入导出布局不变，样式走卡片令牌。

### 8. 启动与闪烁

- 可接受：首屏极短默认浅色，待 `getSettings` 返回后应用真实主题。
- 不强制在 `index.html` 内联脚本（可选优化，非本版必做）。

## 涉及文件（预期）

| 文件 | 变更 |
|------|------|
| `client/internal/settings/settings.go` | 增加 `ThemeMode` |
| `client/internal/settings/settings_test.go` | 覆盖缺省与读写 |
| `client/frontend/src/api/index.ts` | `Settings.theme_mode` |
| `client/frontend/src/stores/settings.ts` | 默认值与加载 |
| `client/frontend/src/composables/useTheme.ts`（新建） | 主题解析与应用 |
| `client/frontend/src/main.ts` | 引入 EP 暗色 CSS；启动应用主题 |
| `client/frontend/src/style.css` | 双套令牌 |
| `client/frontend/src/App.vue` | 侧栏变量化 |
| `client/frontend/src/views/ServersView.vue` | 表格/空状态变量化 |
| `client/frontend/src/views/TunnelsView.vue` | 同上 |
| `client/frontend/src/views/LogsView.vue` | 工具栏随主题；终端保持深色 |
| `client/frontend/src/views/SettingsView.vue` | 主题选择 UI |

如 `SaveSettings` 存在字段合并逻辑，需确认 `theme_mode` 往返不丢失窗口状态字段。

## 验收标准

1. 系统为浅色且 `theme_mode=system` 时，侧栏与内容均为浅色，EP 组件为浅色。
2. 系统为深色且 `theme_mode=system` 时，侧栏与内容均为深色，EP 组件为暗色。
3. 手动选浅色 / 深色可覆盖系统，且重启应用后保持。
4. 服务器、映射表格在两种主题下表头、hover、文字对比度可读。
5. 日志终端在两种主题下均为深色底板，筛选栏随主题变化。
6. 设置页主题切换立即生效并成功保存。
7. 旧 `settings.json` 无 `theme_mode` 时行为等同 `system`，不报错。

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| 局部 scoped 样式仍写死浅色 | 实现时全局搜 `#fff`/`#f` 等硬编码并替换 |
| EP 暗色与自定义令牌冲突 | 自定义只覆盖布局层；组件尽量依赖 EP 变量 |
| 保存设置覆盖窗口几何字段 | 沿用现有后端合并/全量写策略，补测往返 |

## 实现顺序建议

1. 后端字段 + 前端类型 / store  
2. `useTheme` + EP 暗色 CSS + 全局令牌  
3. 侧栏与布局  
4. 表格页、日志页、设置页  
5. 手动切换与跟随系统验收  
