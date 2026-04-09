package email

import (
	"context"
)

type EmailerName string

const (
	EmailerName_Stdout EmailerName = "stdout"
	EmailerName_SMTP   EmailerName = "smtp"
	EmailerName_Resend EmailerName = "resend"
)

type FromEmail string

const (
	FromEmail_NoReplyEmail FromEmail = "Saftaja <no-reply@saftaja.com>"
	FromEmail_HelloEmail   FromEmail = "Saftaja <hello@saftaja.com>"
)

func MapStringToFromEmail(from string) FromEmail {
	switch from {
	case string(FromEmail_NoReplyEmail):
		return FromEmail_NoReplyEmail
	case string(FromEmail_HelloEmail):
		return FromEmail_HelloEmail
	}

	return FromEmail_NoReplyEmail
}

type Emailer interface {
	Send(ctx context.Context, from FromEmail, to, subject, body string) error
	SendFromTemplate(ctx context.Context, from FromEmail, to string, template Template) error
}
