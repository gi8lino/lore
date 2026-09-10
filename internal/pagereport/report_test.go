package pagereport

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("uses defaults", func(t *testing.T) {
		t.Parallel()

		options, ok := Parse(`{{pages query="tag:service status:verified"}}`)

		require.True(t, ok)
		assert.Equal(t, "tag:service status:verified", options.Query)
		assert.Equal(t, []string{"title", "status", "owner", "updated"}, options.Columns)
		assert.Equal(t, "table", options.View)
		assert.Equal(t, 20, options.Limit)
	})

	t.Run("accepts report options", func(t *testing.T) {
		t.Parallel()

		options, ok := Parse(`{{pages query="owner:\"Platform\"" columns="title,property:version,path" view=cards sort=title limit=12}}`)

		require.True(t, ok)
		assert.Equal(t, `owner:"Platform"`, options.Query)
		assert.Equal(t, []string{"title", "property:version", "path"}, options.Columns)
		assert.Equal(t, "cards", options.View)
		assert.Equal(t, "title", options.Sort)
		assert.Equal(t, 12, options.Limit)
	})

	t.Run("rejects unknown columns", func(t *testing.T) {
		t.Parallel()

		_, ok := Parse(`{{pages query="tag:service" columns="title,secret"}}`)
		assert.False(t, ok)
	})
}
