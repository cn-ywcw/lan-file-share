# 俄罗斯方块 — Agent 指南

## 项目结构

单文件项目：`index.html`（HTML + CSS + JS 内嵌）。无构建系统、无依赖、无包管理器。

## 运行方式

**双击 `index.html`** 在浏览器中打开即可游玩。无需 dev server 或构建步骤。

## 架构

- 单个 `Tetris` 类，`requestAnimationFrame` 驱动游戏循环
- 两个 `<canvas>`：主画布（300×600 = 10×20，BLOCK_SIZE=30）和预览画布（120×120，PREVIEW_SIZE=25）
- 状态：`board[][]` 存储已固定的方块类型名，`currentShape/currentX/currentY` 表示当前活动方块

## 关键常量

- `COLS=10`, `ROWS=20`, `BLOCK_SIZE=30`, `PREVIEW_SIZE=25`
- 7 种方块：`I O T S Z J L`，每种定义在 `SHAPES` 中（二维矩阵），颜色在 `COLORS` 中

## 操作方式

| 按键 | 动作 |
|------|------|
| ← → | 左右移动 |
| ↑ | 旋转（含踢墙，8 种偏移） |
| ↓ | 软降（+1 分/格） |
| 空格 | 硬降（+2 分/格） |
| P | 暂停/继续 |
| Enter | 游戏结束时重新开始 |

## 计分与等级

- 消行计分：`SCORE_TABLE = [0, 100, 300, 500, 800]`，乘以当前 `level`
- 软降+1/格，硬降+2/格
- 每消 10 行升 1 级，下落间隔 = `Math.max(50, 800 - (level-1) * 75)`（毫秒）

## 旋转与踢墙

`rotate()` 使用 `rotateMatrix()` 顺时针旋转方块的形状矩阵（I 和 O 为方阵 4×4 和 2×2，其余 3×3）。尝试 8 种偏移 `WALL_KICKS = [[0,0],[-1,0],[1,0],[0,-1],[-1,-1],[1,-1],[-2,0],[2,0]]`，第一个不碰撞的生效。O 方块跳过旋转。

## 幽灵方块

`getGhostY()` 计算当前方块可下落的最低行，用 0.2 透明度绘制。

## 渲染

- `drawBlock()` 绘制一个方块（填充色 + 3 条高光/阴影线 + 内部高光）
- `draw()` 每帧渲染：背景 → 已固定方块 → 幽灵方块 → 当前方块 → 网格线 → 预览
- CSS 深色主题，游戏结束/暂停时显示半透明覆盖层

## 扩展约定

- 添加新方块：在 `SHAPES` 和 `COLORS` 中增加条目，在 `PIECE_NAMES` 中添加名称
- 方块矩阵使用 0/1 表示，旋转要求方阵（2×2, 3×3, 4×4）
- 方块锁定后存入 `board[y][x] = typeName`（字符串，与 COLORS 的键一致）

## 彩蛋

游戏进行中输入 `secret` 触发彩虹方块效果（约 5 秒），伴随彩色粒子下落。相关方法：`triggerEasterEgg()`、`spawnParticles()`、`drawParticles()`、`rainbowColor()`。

## 多语言

支持 4 种语言，默认中文，通过 `#lang-select` 下拉框切换（自动保存到 `localStorage`）。

| 语言 | 代码 |
|------|------|
| 中文 | `zh`（默认） |
| English | `en` |
| 日本語 | `ja` |
| 한국어 | `ko` |

### 国际化机制

- 所有用户可见文本放置在 `<span data-i18n="key">` 等元素中
- `LANGUAGES` 对象（`index.html:318`）按语言代码分组，每个 key 对应一个翻译
- `Tetris.applyLanguage()`（`index.html:661`）遍历 `[data-i18n]` 元素并更新文本
- 含 HTML 标签的值使用 `innerHTML` 设置（如暂停提示中的 `<kbd>`），纯文本使用 `textContent`
- `<html lang>` 同步更新（中文使用 `zh-CN`，其余直接用语言代码）

### 扩展语言

在 `LANGUAGES` 中添加新条目，确保 key 与现有语言一致，然后给 `<select>` 添加 `<option>`。所有 `data-i18n` key 必须覆盖完整。
