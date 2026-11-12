package system

import (
	"context"
	"fmt"
	"html"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

var pendingNotifications sync.WaitGroup

// WaitNotifications blocks until notifications already sent have been handed to the OS (5s max each).
func WaitNotifications() {
	pendingNotifications.Wait()
}

// SendNotification sends a non-blocking desktop notification.
func SendNotification(title, message string) {
	pendingNotifications.Add(1)
	go func() {
		defer pendingNotifications.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		switch runtime.GOOS {
		case "linux":
			if bin, err := exec.LookPath("notify-send"); err == nil {
				args := []string{"-a", "anpan", "-i", "anpan", title, message}
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
`, html.EscapeString(title), html.EscapeString(message))
			_ = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psScript).Run()
		}
	}()
}
