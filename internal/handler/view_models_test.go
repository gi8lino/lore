package handler

import (
	"testing"

	"github.com/gi8lino/lore/internal/navigation"
	"github.com/stretchr/testify/assert"
)

func TestPagePathOptions(t *testing.T) {
	t.Parallel()

	tree := []navigation.Node{
		{
			Slug: "platforms", Title: "Platforms", Children: []navigation.Node{
				{Slug: "platforms/containers", Title: "Containers"},
				{Slug: "platforms/kubernetes", Title: "Kubernetes"},
			},
		},
	}

	assert.Equal(t, []pagePathOption{
		{Slug: "platforms", Label: "Platforms"},
		{Slug: "platforms/containers", Label: "Platforms / Containers"},
		{Slug: "platforms/kubernetes", Label: "Platforms / Kubernetes"},
	}, pagePathOptions(tree, ""))
	assert.Equal(t, []pagePathOption{
		{Slug: "platforms", Label: "Platforms"},
		{Slug: "platforms/kubernetes", Label: "Platforms / Kubernetes"},
	}, pagePathOptions(tree, "platforms/containers"))
}
