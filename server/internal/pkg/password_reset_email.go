package pkg

import (
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"
)

type PasswordResetEmail struct {
	Subject   string
	HTML      string
	PlainText string
	ResetURL  string
}

// BuildResetPasswordURL returns {base}?email=...&otp=... (query values URL-encoded).
func BuildResetPasswordURL(baseURL, email, otp string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return ""
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	q := u.Query()
	q.Set("email", email)
	q.Set("otp", otp)
	u.RawQuery = q.Encode()
	return u.String()
}

func BuildPasswordResetEmail(fullName, email, otp, resetPageURL string, validFor time.Duration) PasswordResetEmail {
	name := strings.TrimSpace(fullName)
	if name == "" {
		name = "bạn"
	}
	minutes := int(validFor.Minutes())
	if minutes < 1 {
		minutes = 15
	}

	resetURL := BuildResetPasswordURL(resetPageURL, email, otp)
	escapedName := html.EscapeString(name)
	escapedOTP := html.EscapeString(otp)
	escapedURL := html.EscapeString(resetURL)

	subject := "Vidio Guard — Đặt lại mật khẩu"

	plain := fmt.Sprintf(
		"Xin chào %s,\n\nBạn vừa yêu cầu đặt lại mật khẩu Vidio Guard.\n\nMã xác minh: %s\nThời hạn: %d phút\n",
		name, otp, minutes,
	)
	if resetURL != "" {
		plain += fmt.Sprintf("\nMở liên kết để đặt lại mật khẩu:\n%s\n", resetURL)
	}
	plain += "\nNếu bạn không yêu cầu, hãy bỏ qua email này.\n\n— Vidio Guard\n"

	buttonBlock := ""
	if resetURL != "" {
		buttonBlock = fmt.Sprintf(`
          <tr>
            <td style="padding:8px 0 24px;text-align:center;">
              <a href="%s" target="_blank" rel="noopener noreferrer"
                 style="display:inline-block;background:linear-gradient(135deg,#6366f1 0%%,#8b5cf6 100%%);color:#ffffff;font-size:16px;font-weight:600;text-decoration:none;padding:14px 32px;border-radius:10px;box-shadow:0 4px 14px rgba(99,102,241,0.35);">
                Đặt lại mật khẩu
              </a>
            </td>
          </tr>
          <tr>
            <td style="padding:0 0 20px;font-size:13px;color:#64748b;text-align:center;word-break:break-all;">
              Hoặc sao chép liên kết:<br/>
              <a href="%s" style="color:#6366f1;">%s</a>
            </td>
          </tr>`,
			escapedURL, escapedURL, escapedURL)
	}

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="vi">
<head>
  <meta charset="UTF-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
  <title>%s</title>
</head>
<body style="margin:0;padding:0;background-color:#f1f5f9;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;">
  <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="background-color:#f1f5f9;padding:32px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="max-width:520px;background:#ffffff;border-radius:16px;overflow:hidden;box-shadow:0 10px 40px rgba(15,23,42,0.08);">
          <tr>
            <td style="background:linear-gradient(135deg,#0f172a 0%%,#1e293b 50%%,#312e81 100%%);padding:28px 32px;text-align:center;">
              <div style="font-size:22px;font-weight:700;color:#ffffff;letter-spacing:-0.02em;">Vidio Guard</div>
              <div style="font-size:13px;color:#94a3b8;margin-top:6px;">Đặt lại mật khẩu</div>
            </td>
          </tr>
          <tr>
            <td style="padding:32px 32px 8px;">
              <p style="margin:0 0 16px;font-size:16px;line-height:1.6;color:#334155;">Xin chào <strong>%s</strong>,</p>
              <p style="margin:0 0 24px;font-size:15px;line-height:1.6;color:#64748b;">
                Bạn vừa yêu cầu đặt lại mật khẩu. Nhấn nút bên dưới hoặc dùng mã xác minh — mã có hiệu lực <strong>%d phút</strong>.
              </p>
            </td>
          </tr>
          <tr>
            <td style="padding:0 32px;">
              <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="background:#f8fafc;border:1px dashed #cbd5e1;border-radius:12px;">
                <tr>
                  <td style="padding:20px;text-align:center;">
                    <div style="font-size:12px;text-transform:uppercase;letter-spacing:0.08em;color:#64748b;margin-bottom:8px;">Mã xác minh</div>
                    <div style="font-size:36px;font-weight:700;letter-spacing:0.35em;color:#4f46e5;font-family:ui-monospace,'Cascadia Code',monospace;">%s</div>
                  </td>
                </tr>
              </table>
            </td>
          </tr>
          %s
          <tr>
            <td style="padding:24px 32px 32px;">
              <p style="margin:0;font-size:13px;line-height:1.5;color:#94a3b8;text-align:center;">
                Nếu bạn không yêu cầu đặt lại mật khẩu, hãy bỏ qua email này.<br/>Tài khoản của bạn vẫn an toàn.
              </p>
            </td>
          </tr>
        </table>
        <p style="margin:16px 0 0;font-size:12px;color:#94a3b8;">© Vigilant Lens</p>
      </td>
    </tr>
  </table>
</body>
</html>`,
		html.EscapeString(subject),
		escapedName,
		minutes,
		escapedOTP,
		buttonBlock,
	)

	return PasswordResetEmail{
		Subject:   subject,
		HTML:      htmlBody,
		PlainText: plain,
		ResetURL:  resetURL,
	}
}
