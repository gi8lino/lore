package store

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMutationErrorPreservesKnownConflictCauses(t *testing.T) {
	t.Parallel()
	for _, constraint := range []string{
		"pages_slug_key", "page_aliases_pkey", "wiki_groups_name_key", "wiki_groups_name_ci_idx",
		"page_templates_name_key", "page_templates_name_ci_idx", "knowledge_snippets_kind_name_key",
		"knowledge_snippets_kind_name_ci_idx", "saved_searches_user_id_name_key", "saved_searches_user_name_ci_idx",
	} {
		t.Run(constraint, func(t *testing.T) {
			t.Parallel()
			cause := &pgconn.PgError{Code: "23505", ConstraintName: constraint}
			err := mutationError(fmt.Errorf("persist: %w", cause))
			assert.ErrorIs(t, err, domain.ErrAlreadyExists)
			assert.ErrorIs(t, err, cause)
			databaseError, ok := errors.AsType[*pgconn.PgError](err)
			require.True(t, ok)
			assert.Same(t, cause, databaseError)
		})
	}
}

func TestMutationErrorPreservesMissingGroupFields(t *testing.T) {
	t.Parallel()
	for constraint, field := range map[string]string{
		"user_groups_group_id_fkey":         "group_ids",
		"page_groups_group_id_fkey":         "group_id",
		"pages_owner_group_id_fkey":         "owner_group_id",
		"oidc_group_mappings_group_id_fkey": "oidc_group_mappings",
	} {
		t.Run(constraint, func(t *testing.T) {
			t.Parallel()
			cause := &pgconn.PgError{Code: "23503", ConstraintName: constraint, Detail: "private database detail"}
			err := mutationError(cause)
			validation, ok := errors.AsType[*domain.ValidationError](err)
			require.True(t, ok)
			require.Len(t, validation.Fields, 1)
			assert.Equal(t, field, validation.Fields[0].Field)
			assert.NotContains(t, validation.Fields[0].Message, cause.Detail)
			assert.ErrorIs(t, err, cause)
		})
	}
}

func TestMutationErrorDoesNotReclassifyInfrastructureFailures(t *testing.T) {
	t.Parallel()
	for _, err := range []error{
		nil, errors.New("connection lost"),
		&pgconn.PgError{Code: "23505", ConstraintName: "api_tokens_token_hash_key"},
		&pgconn.PgError{Code: "23503", ConstraintName: "page_revisions_page_id_fkey"},
		&pgconn.PgError{Code: "40001", ConstraintName: "pages_slug_key"},
	} {
		assert.Equal(t, err, mutationError(err))
	}
}
