package system

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// SendNotification sends a non-blocking desktop notification.
func SendNotification(title, message string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		switch runtime.GOOS {
		case "linux":
			if bin, err := exec.LookPath("notify-send"); err == nil {
				home, _ := os.UserHomeDir()
				iconPaths := []string{
					filepath.Join(home, ".local", "share", "icons", "hicolor", "256x256", "apps", "anpan.png"),
					filepath.Join(home, ".local", "share", "pixmaps", "anpan.png"),
					"/usr/share/icons/hicolor/256x256/apps/anpan.png",
					"/usr/share/pixmaps/anpan.png",
				}
				icon := ""
				for _, ip := range iconPaths {
					if _, err := os.Stat(ip); err == nil {
						icon = ip
						break
					}
				}

				args := []string{"-a", "anpan"}
				if icon != "" {
					args = append(args, "-i", icon)
				}
				args = append(args, title, message)
				_ = exec.CommandContext(ctx, bin, args...).Run()
			}

		case "darwin":
			script := fmt.Sprintf(`display notification %q with title %q sound name "default"`, message, title)
			_ = exec.CommandContext(ctx, "osascript", "-e", script).Run()

		case "windows":
			psScript := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
$template = @"
<toast>
    <visual>
        <binding template="ToastGeneric">
            <text>%s</text>
            <text>%s</text>
        </binding>
    </visual>
</toast>
"@
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($template)
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("anpan").Show($toast)
`, strings.ReplaceAll(title, `"`, `\"`), strings.ReplaceAll(message, `"`, `\"`))
			_ = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psScript).Run()
		}
	}()
}
