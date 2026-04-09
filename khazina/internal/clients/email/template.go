package email

import (
	"fmt"
)

type Template struct {
	Subject string
	HTML    string
}

type Templates interface {
	MagicLinkTemplate(token string) (*Template, error)
	WelcomeTemplate(name string) (*Template, error)
}

type templates struct {
	domain string
}

func NewTemplates(domain string) Templates {
	return &templates{
		domain: domain,
	}
}

func (t *templates) MagicLinkTemplate(token string) (*Template, error) {
	subject := "Saftaja Magic Link ✉️"

	htmlContent := fmt.Sprintf(`<h1>%s</h1>
	
Use the link below to login to your account, or create one if this is your first time:

Link: %s/magic-link?token=%s

If you didn't request this email, you can ignore it safely :D

Thanks,
Your Friendly Saftaja Team`, subject, t.domain, token)

	return &Template{
		Subject: subject,
		HTML:    htmlContent,
	}, nil
}

func (t *templates) WelcomeTemplate(name string) (*Template, error) {
	subject := fmt.Sprintf("%s, You are Cafu! 🚀", name)

	htmlContent := fmt.Sprintf(`Hello %s!

We are happy that you decided to use the best payment checkout system to exist :D

Thanks,
Your Friendly Saftaja Team`, name)

	return &Template{
		Subject: subject,
		HTML:    htmlContent,
	}, nil
}
