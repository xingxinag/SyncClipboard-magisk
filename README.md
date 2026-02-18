# SyncClipboard

Root/system-level cross-device clipboard sync module for Magisk/KernelSU/APatch.

Current stable architecture baseline: **v2.6.2 (system hook route)**.

---

## 中文

## 1. 项目定位

SyncClipboard 通过 Root 环境在 Android 设备侧运行 `clipserver`，将本机剪贴板与 WebDAV 进行双向同步。

从 `v2.6.2` 开始，核心链路升级为：

- `system_server` ClipboardService Hook（LSPosed）
- 本地协议目录（`/data/system/syncclipboard_hook`）
- `clipserver` 通过 `system_hook` 优先读写
- Web UI 功能保持完整，不删减

---

## 2. 模块架构（重点）

系统架构：

1. **系统剪贴板层**（system_server）
   - 由 `SyncClipboardSystemHook.apk` 在 LSPosed 中注入 Hook。
   - 捕获系统剪贴板读写事件。

2. **协议层**（本地文件协议）
   - 目录：`/data/system/syncclipboard_hook`
   - 文件：
     - `clipboard_state.json`（当前剪贴板状态）
     - `clipboard_command.json`（clipserver 下发写命令）
     - `clipboard_ack.json`（hook 执行回执）
     - `hook_events.log`（hook 事件日志）

3. **服务层**（clipserver）
   - API：`/api/clipboard`、`/api/sync/now`、`/api/accounts/test` 等。
   - 策略优先级：`system_hook` 优先；其余方法仅作兜底诊断。

4. **同步层**（WebDAV）
   - 负责上传/下载 `SyncClipboard.json`。

5. **UI 层**（Web UI）
   - 地址：`http://127.0.0.1:8964`
   - 保留账号管理、连接测试、自动同步、立即同步、状态显示等全部功能。

---

## 3. 系统要求

- Android: 8.0+（建议 10+）
- Root 环境（三选一）：
  - Magisk
  - KernelSU
  - APatch
- 建议开启：Zygisk
- 必需：LSPosed（并正确启用本模块作用域）

---

## 4. 安装与启用

1. 下载最新 `SyncClipboard-magisk_vX.Y.Z.zip`
2. 在 Magisk/KernelSU/APatch 管理器安装模块
3. 重启设备
4. 在 LSPosed 中启用 `SyncClipboardSystemHook`
5. 作用域勾选：`android`（system_server）
6. 再次重启（建议）
7. 访问 `http://127.0.0.1:8964` 配置 WebDAV

> 说明：第 4~6 步缺失时，通常会出现 `system hook state not available` 或 `set timeout`。

---

## 5. Web UI 功能（完整保留）

- 多账号管理（新增、删除、切换）
- 连接测试（保存前验证）
- 自动同步开关
- 同步间隔设置
- 立即同步
- 同步状态与错误提示

---

## 6. API 说明

| Endpoint | Method | Description |
|---|---|---|
| `/health` | GET | 健康检查 |
| `/api/config` | GET/POST | 获取/更新配置 |
| `/api/clipboard` | GET | 获取当前剪贴板 |
| `/api/clipboard` | POST | 设置当前剪贴板 |
| `/api/sync/now` | POST | 立即执行一次同步 |
| `/api/sync/status` | GET | 获取同步状态 |
| `/api/accounts/test` | POST | 测试 WebDAV 连接 |

---

## 7. 使用方法（推荐流程）

1. 在 Web UI 添加并测试 WebDAV 账号
2. 开启自动同步
3. 在任意 App 复制文本（微信/浏览器/输入框）
4. 在另一端触发同步或等待自动同步
5. 另一端粘贴验证

---

## 8. 故障排查（system-hook 路线）

### 8.1 快速判断

- WebDAV 正常但剪贴板失败：优先检查 LSPosed 作用域
- `api/clipboard` 返回 500：优先检查 hook 协议文件是否更新

### 8.2 关键检查命令

```bash
# 1) 模块版本
grep -E "version=|versionCode=" /data/adb/modules/syncclipboard/module.prop

# 2) 服务进程
pidof clipserver

# 3) hook 协议目录
ls -la /data/system/syncclipboard_hook

# 4) system hook 包
pm path com.syncclipboard.systemhook

# 5) 最近日志（重点看 method=system_hook）
tail -n 200 /data/adb/syncclipboard/clipserver.log
```

### 8.3 判定标准

- 成功读剪贴板：`[clipboard/get] ok method=system_hook`
- 成功写剪贴板：`[clipboard/set] ok method=system_hook`

---

## 9. 反馈格式（请尽量按模板）

为快速定位，请按以下模板反馈：

```text
[设备信息]
机型:
系统版本:
Root方案: (Magisk/KernelSU/APatch)
LSPosed版本:
模块版本:

[现象]
1) 系统复制 -> /api/clipboard 结果:
2) /api/accounts/test 结果:
3) /api/sync/now 结果:
4) WebDAV下发后能否粘贴:

[日志/命令输出]
1) grep -E "version=|versionCode=" /data/adb/modules/syncclipboard/module.prop
2) pidof clipserver
3) ls -la /data/system/syncclipboard_hook
4) pm path com.syncclipboard.systemhook
5) tail -n 200 /data/adb/syncclipboard/clipserver.log
6) logcat -d | grep -i SyncClipboardHook | tail -n 100

[补充说明]
是否刚重启:
是否刚切换LSPosed作用域:
是否存在其他剪贴板/Xposed模块:
```

---

## 10. 变更记录

- 详细版本记录见：`CHANGELOG.md`
- `v2.6.2` 为 system-hook 首个稳定基线版本

---

## 11. English (Brief)

SyncClipboard now uses a system hook route (LSPosed + system_server ClipboardService hook) as primary clipboard backend.

Quick start:

1. Install module ZIP
2. Reboot
3. Enable `SyncClipboardSystemHook` in LSPosed with `android` scope
4. Reboot
5. Configure WebDAV at `http://127.0.0.1:8964`

For troubleshooting and reporting format, use Sections 8 and 9 above.
