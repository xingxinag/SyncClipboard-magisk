# SyncClipboard 兼容性说明

## 剪贴板访问方法

为了确保在各种深度定制的 Android 系统上都能正常工作，本模块实现了**4种剪贴板访问方法**，按优先级依次尝试：

### 方法 1: cmd clipboard (推荐)
- **适用系统**: Android 10+
- **命令**: `cmd clipboard get-text` / `cmd clipboard set-text`
- **优点**: 官方API，最稳定
- **缺点**: 仅 Android 10+ 支持

### 方法 2: service call (通用)
- **适用系统**: Android 8.0+
- **命令**: `service call clipboard 2 s16 com.android.shell`
- **优点**: 兼容性好，大部分系统支持
- **缺点**: 输出格式需要解析

### 方法 3: am broadcast (备用)
- **适用系统**: 定制系统
- **命令**: `am broadcast -a clipper.set -e text 'content'`
- **优点**: 某些定制系统的专用方法
- **缺点**: 需要特定接收器

### 方法 4: content provider (最后尝试)
- **适用系统**: 部分定制系统
- **命令**: `content query --uri content://clipboard/text`
- **优点**: 通过 content provider 访问
- **缺点**: 并非所有系统都支持

## 测试过的系统

### ✅ 已测试兼容
- AOSP (Android 8.0 - 14)
- LineageOS
- Pixel Experience
- MIUI (小米)
- ColorOS (OPPO)
- Flyme (魅族)
- OneUI (三星)

### ⚠️ 可能需要额外配置
- HarmonyOS (华为) - 可能需要关闭纯净模式
- OriginOS (vivo) - 可能需要授予额外权限
- Funtouch OS (vivo 旧版) - 可能需要 SELinux 规则

### ❌ 已知不兼容
- 无（目前未发现完全不兼容的系统）

## SELinux 支持

模块包含 `sepolicy.rule` 文件，用于 KernelSU/APatch 环境：

```
# 允许访问剪贴板服务
allow system_server clipboard_service:service_manager find
allow untrusted_app clipboard_service:service_manager find

# 允许读写剪贴板
allow system_server clipboard:file { read write open }
allow untrusted_app clipboard:file { read write open }

# 允许执行 su 命令
allow untrusted_app su:file { execute execute_no_trans }
allow untrusted_app su:process { transition }
```

## 环境支持

### Magisk
- **版本要求**: 26.4+
- **特性**: 完全支持，无需额外配置
- **SELinux**: 自动处理

### KernelSU
- **版本要求**: 0.6.6+ (kernel) + 11575+ (ksud)
- **特性**: 完全支持
- **SELinux**: 自动加载 sepolicy.rule

### APatch
- **版本要求**: 0.10.7+
- **特性**: 完全支持
- **SELinux**: 自动加载 sepolicy.rule

## 故障排除

### 问题 1: 剪贴板读取失败

**症状**: 日志显示 "failed to access system clipboard: all methods failed"

**解决方案**:
1. 确认已授予 Root 权限
2. 检查 SELinux 状态: `getenforce`
3. 如果是 Enforcing，检查 sepolicy.rule 是否正确加载
4. 尝试手动执行命令测试:
   ```bash
   su -c "cmd clipboard get-text"
   ```

### 问题 2: 某些系统上无法写入剪贴板

**症状**: 读取正常，但写入失败

**解决方案**:
1. 检查系统是否有剪贴板保护功能
2. 在系统设置中关闭"剪贴板保护"或"隐私保护"
3. 尝试使用不同的方法:
   ```bash
   # 方法 1
   su -c "cmd clipboard set-text" <<< "test content"
   
   # 方法 2
   su -c "service call clipboard 1 i32 1 s16 com.android.shell s16 'test'"
   ```

### 问题 3: 深度定制系统兼容性

**症状**: 标准方法都失败

**解决方案**:
1. 查看系统日志: `logcat | grep clipboard`
2. 检查系统是否有自定义的剪贴板服务
3. 联系开发者提供系统信息以添加支持

## 性能优化

### 剪贴板访问频率
- 默认同步间隔: 60 秒
- 建议范围: 30-300 秒
- 过于频繁可能影响电池续航

### 内存占用
- Go 后端: ~10-15MB
- 剪贴板缓存: 最大 1MB
- WebUI: 静态文件，无额外占用

### 网络流量
- 仅在内容变化时上传
- 压缩传输（WebDAV 支持）
- 建议在 WiFi 下使用

## 安全性

### Root 权限
- 仅用于访问剪贴板
- 不修改系统文件
- 不收集用户数据

### WebDAV 连接
- 支持 HTTPS
- 密码存储在本地配置文件
- 建议使用应用专用密码

### 隐私保护
- 剪贴板内容仅上传到用户指定的 WebDAV
- 不经过第三方服务器
- 本地日志可随时清除

## 更新日志

### v1.0.0 (2026-02-15)
- ✅ 实现 4 种剪贴板访问方法
- ✅ 添加 SELinux 规则支持
- ✅ 优化服务启动流程
- ✅ 改进架构检测
- ✅ 增强错误处理

## 反馈

如果在您的设备上遇到兼容性问题，请提供以下信息：

1. 设备型号和系统版本
2. Root 方案（Magisk/KernelSU/APatch）
3. 错误日志: `/data/adb/syncclipboard/clipserver.log`
4. 手动测试结果:
   ```bash
   su -c "cmd clipboard get-text"
   su -c "service call clipboard 2 s16 com.android.shell"
   ```

提交 Issue: https://github.com/yourusername/syncclipboard/issues
