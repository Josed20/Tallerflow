package mail

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	htmltmpl "html/template"
	"strings"
	"sync"
	texttmpl "text/template"
)

//go:embed templates/*
var templateFS embed.FS

// Message defines the email message payload.
type Message struct {
	To      string
	Subject string
	Text    string // Siempre presente
	HTML    string // Opcional
}

// Sender is the common interface implemented by all mail dispatchers.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// Fake is an in-memory thread-safe implementation of Sender for tests.
type Fake struct {
	mu       sync.Mutex
	messages []Message
	sendErr  error
}

// NewFake creates a new empty Fake mail sender.
func NewFake() *Fake {
	return &Fake{messages: make([]Message, 0)}
}

// Send records the message in memory or returns an error if configured.
func (f *Fake) Send(_ context.Context, msg Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sendErr != nil {
		return f.sendErr
	}
	f.messages = append(f.messages, msg)
	return nil
}

// SetError forces subsequent Send calls to return this error.
func (f *Fake) SetError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sendErr = err
}

// Messages returns a copy of all sent messages.
func (f *Fake) Messages() []Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	copied := make([]Message, len(f.messages))
	copy(copied, f.messages)
	return copied
}

// LastMessage returns the most recently sent message, if any.
func (f *Fake) LastMessage() (Message, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.messages) == 0 {
		return Message{}, false
	}
	return f.messages[len(f.messages)-1], true
}

// Reset clears recorded messages and any set error.
func (f *Fake) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = f.messages[:0]
	f.sendErr = nil
}

type templateDefinition struct {
	subject string
	text    string
}

var registeredTemplates = map[string]templateDefinition{
	"verify-email": {
		subject: "Verifica tu correo — TallerFlow",
		text: `Hola{{if .Name}} {{.Name}}{{end}},

¡Bienvenido a TallerFlow! Se ha creado el taller {{if .WorkshopName}}«{{.WorkshopName}}»{{else}}en la plataforma{{end}}.

Para activar tu cuenta y acceder a tu panel, verifica tu correo en el siguiente enlace:
{{.URL}}

Este enlace es válido por {{if .ExpiresIn}}{{.ExpiresIn}}{{else}}24 horas{{end}} y solo puede ser utilizado una vez.

— Equipo TallerFlow`,
	},
	"team-invitation": {
		subject: "Invitación para unirte a {{if .WorkshopName}}{{.WorkshopName}}{{else}}un taller{{end}} en TallerFlow",
		text: `Hola,

{{if .InviterName}}{{.InviterName}}{{else}}Un administrador{{end}} te ha invitado a unirte al taller «{{.WorkshopName}}» con el rol de {{.Role}} en TallerFlow.

Para aceptar la invitación y configurar tu cuenta, ingresa al siguiente enlace:
{{.URL}}

Este enlace es válido por {{if .ExpiresIn}}{{.ExpiresIn}}{{else}}7 días{{end}}.

— Equipo TallerFlow`,
	},
	"password-reset": {
		subject: "Restablecimiento de contraseña — TallerFlow",
		text: `Hola,

Recibimos una solicitud para restablecer la contraseña de tu cuenta en TallerFlow.

Puedes crear una nueva contraseña ingresando en el siguiente enlace:
{{.URL}}

Este enlace es válido por {{if .ExpiresIn}}{{.ExpiresIn}}{{else}}1 hora{{end}} y solo puede ser utilizado una vez.
Si no solicitaste este cambio, puedes ignorar este correo de forma segura.

— Equipo TallerFlow`,
	},
	"workshop-suspended": {
		subject: "Tu taller ha sido suspendido — TallerFlow",
		text: `Hola,

El taller «{{.WorkshopName}}» ha sido suspendido temporalmente{{if .Reason}}: {{.Reason}}{{end}}.

Si crees que se trata de un error o deseas reactivar tu cuenta, contáctanos a {{if .SupportEmail}}{{.SupportEmail}}{{else}}soporte@tallerflow.pe{{end}}.

— Equipo TallerFlow`,
	},
	"workshop-deletion-scheduled": {
		subject: "Eliminación programada de taller — TallerFlow",
		text: `Hola,

Se ha programado la eliminación definitiva del taller «{{.WorkshopName}}» para el {{.DeletionDate}}.

Durante los próximos 30 días puedes solicitar la cancelación de este proceso comunicándote a {{if .SupportEmail}}{{.SupportEmail}}{{else}}soporte@tallerflow.pe{{end}}.

Pasado ese plazo, todos los datos del taller serán borrados de manera irreversible.

— Equipo TallerFlow`,
	},
}

// Render builds Subject, Text and rich branded HTML from a named template and provided data.
func Render(templateName string, data any) (Message, error) {
	cleanName := strings.TrimSpace(templateName)
	tmplDef, ok := registeredTemplates[cleanName]
	if !ok {
		return Message{}, fmt.Errorf("unknown email template: %q", templateName)
	}

	subjTmpl, err := texttmpl.New("subject").Parse(tmplDef.subject)
	if err != nil {
		return Message{}, fmt.Errorf("parse subject template: %w", err)
	}
	var subjectBuf bytes.Buffer
	if err := subjTmpl.Execute(&subjectBuf, data); err != nil {
		return Message{}, fmt.Errorf("render subject: %w", err)
	}

	textTmpl, err := texttmpl.New("text").Parse(tmplDef.text)
	if err != nil {
		return Message{}, fmt.Errorf("parse text template: %w", err)
	}
	var textBuf bytes.Buffer
	if err := textTmpl.Execute(&textBuf, data); err != nil {
		return Message{}, fmt.Errorf("render text: %w", err)
	}

	var htmlContent string
	htmlFileName := fmt.Sprintf("templates/%s.html", cleanName)
	if parsedHTML, err := htmltmpl.ParseFS(templateFS, htmlFileName); err == nil {
		var htmlBuf bytes.Buffer
		if err := parsedHTML.ExecuteTemplate(&htmlBuf, cleanName, data); err == nil {
			htmlContent = htmlBuf.String()
		}
	}

	subject := strings.TrimSpace(subjectBuf.String())
	if subject == "" {
		return Message{}, errors.New("rendered email subject is empty")
	}

	return Message{
		Subject: subject,
		Text:    strings.TrimSpace(textBuf.String()),
		HTML:    strings.TrimSpace(htmlContent),
	}, nil
}
