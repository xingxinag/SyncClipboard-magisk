package clipboard

import (
	"fmt"
	"os/exec"
	"strings"
)

// getClipboardRoot 通过 root 权限直接访问剪贴板（无需 APK）
// 使用 am broadcast 触发系统剪贴板读取
func getClipboardRoot() (string, error) {
	// 方案1: 通过 am broadcast 触发剪贴板读取
	// 创建一个临时的 broadcast receiver
	script := `
# 创建临时脚本
cat > /data/local/tmp/read_clip.sh << 'EOF'
#!/system/bin/sh
# 通过 am 命令读取剪贴板
am broadcast -a android.intent.action.GET_CONTENT --es mime "text/plain" > /data/local/tmp/clip_result.txt 2>&1
EOF

chmod 755 /data/local/tmp/read_clip.sh
/data/local/tmp/read_clip.sh
cat /data/local/tmp/clip_result.txt
`

	cmd := exec.Command("su", "-c", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute root script: %w", err)
	}

	content := strings.TrimSpace(string(output))
	if content == "" {
		return "", fmt.Errorf("clipboard is empty")
	}

	return content, nil
}

// setClipboardRoot 通过 root 权限直接写入剪贴板（无需 APK）
func setClipboardRoot(content string) error {
	// 方案: 通过 am broadcast 设置剪贴板
	escapedContent := strings.ReplaceAll(content, "'", "'\\''")

	script := fmt.Sprintf(`
# 通过 am broadcast 设置剪贴板
am broadcast -a android.intent.action.SEND --es android.intent.extra.TEXT '%s'
`, escapedContent)

	cmd := exec.Command("su", "-c", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set clipboard via root: %w", err)
	}

	return nil
}

// getClipboardRootDirect 通过 root 直接调用 ClipboardManager（最激进方案）
func getClipboardRootDirect() (string, error) {
	// 使用 su 上下文执行 Java 代码
	javaCode := `
import android.content.ClipboardManager;
import android.content.ClipData;
import android.content.Context;

Context ctx = android.app.ActivityThread.systemMain().getSystemContext();
ClipboardManager cm = (ClipboardManager) ctx.getSystemService(Context.CLIPBOARD_SERVICE);
ClipData clip = cm.getPrimaryClip();
if (clip != null && clip.getItemCount() > 0) {
    System.out.println(clip.getItemAt(0).getText());
}
`

	// 保存 Java 代码
	tmpFile := "/data/local/tmp/ReadClip.java"
	if err := exec.Command("su", "-c", fmt.Sprintf("echo '%s' > %s", javaCode, tmpFile)).Run(); err != nil {
		return "", err
	}

	// 通过 app_process 执行
	cmd := exec.Command("su", "-c", fmt.Sprintf("app_process -Djava.class.path=/system/framework/framework.jar /system/bin %s", tmpFile))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute java code: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
