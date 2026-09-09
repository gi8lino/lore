package handler

import (
	"bytes"
	"html/template"
	"os"
	"testing"
	"time"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationHeaderUnreadClass(t *testing.T) {
	t.Parallel()

	source, err := os.ReadFile("../../web/src/templates/header.gohtml")
	require.NoError(t, err)

	tmpl, err := template.New("header").Funcs(template.FuncMap{
		"icon":    func(string, int) template.HTML { return "" },
		"logo":    func() template.HTML { return "" },
		"timeago": func(time.Time) string { return "now" },
	}).Parse(string(source))
	require.NoError(t, err)

	data := ViewData{
		ApplicationSettings: domain.ApplicationSettings{ExternalLinks: []domain.ExternalLink{{
			Label:       "Repository",
			URL:         "https://github.com/gi8lino/lore",
			Icon:        "code",
			Description: "v2.4.1",
		}}},
		Notifications: []domain.Notification{{
			ID:        7,
			Kind:      "mention",
			Title:     "Mention",
			CreatedAt: time.Now(),
		}},
		UnreadNotifications: 1,
	}

	var output bytes.Buffer
	require.NoError(t, tmpl.ExecuteTemplate(&output, "header", data))

	assert.Contains(t, output.String(), `class="notification-item unread"`)
	assert.Contains(t, output.String(), `href="https://github.com/gi8lino/lore"`)
	assert.Contains(t, output.String(), ">Repository</strong>")
	assert.Contains(t, output.String(), ">v2.4.1</small>")
	assert.NotContains(t, output.String(), `class="notification-itemunread"`)
}
