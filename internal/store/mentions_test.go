package store

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMentionedUsernames(t *testing.T) {
	for _, tt := range []struct {
		source string
		want   []string
	}{
		{"No mentions", nil},
		{"@Alice, @BOB and @alice", []string{"alice", "bob"}},
		{"mail@example.com prefix@user _@hidden 9@hidden", nil},
		{"(@a.b-c_d)\n@next", []string{"a.b-c_d", "next"}},
		{"@ @! @é", nil},
		{"é@alice @@bob", []string{"alice", "bob"}},
		{"@a.@b @c-@d", []string{"a.", "c-"}},
		{"@one,@two", []string{"one", "two"}},
	} {
		t.Run(tt.source, func(t *testing.T) { assert.Equal(t, tt.want, mentionedUsernames(tt.source)) })
	}
}
