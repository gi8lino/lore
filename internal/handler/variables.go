package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/gi8lino/lore/internal/domain"
	md "github.com/gi8lino/lore/internal/markdown"
)

const (
	maxExportVariables          = 128
	maxExportVariableBytes      = 8 << 10
	maxExportVariableTotalBytes = 64 << 10
)

// pageVariable describes only a variable actually expanded by this page. Saved
// values are never replaced in the inspector by a temporary export override.
type pageVariable struct {
	Name        string
	Value       string
	Description string
	Occurrences int
}

type expandedPageKnowledge struct {
	Markdown    string
	Variables   []pageVariable
	Annotations []md.Variable
}

// pageKnowledge keeps lookup results, provenance and overrides local to one
// render. It deliberately exposes no write operations to the snippet store.
type pageKnowledge struct {
	knowledgeContent
	overrides   map[string]string
	prefix      string
	indices     map[string]int
	aliases     map[string]int
	items       []domain.KnowledgeSnippet
	variables   []pageVariable
	annotations []md.Variable
}

// KnowledgeSnippetByName intercepts variables, leaving snippets and include
// expansion to the existing resolver. Repeated names use one consistent value.
func (p *pageKnowledge) KnowledgeSnippetByName(ctx context.Context, kind, name string) (domain.KnowledgeSnippet, error) {
	if kind != "variable" {
		return p.knowledgeContent.KnowledgeSnippetByName(ctx, kind, name)
	}
	index, found := p.aliases[name]
	if !found {
		item, err := p.knowledgeContent.KnowledgeSnippetByName(ctx, kind, name)
		if err != nil {
			return domain.KnowledgeSnippet{}, err
		}
		// PostgreSQL resolves names without regard to case. Use the stored name
		// as the identity so differently cased macros share one field and value.
		canonical := item.Name
		index, found = p.indices[canonical]
		if !found {
			index = len(p.items)
			p.indices[canonical] = index
			p.items = append(p.items, item)
			p.variables = append(p.variables, pageVariable{
				Name: canonical, Value: item.Content, Description: item.Description,
			})
			value := item.Content
			if override, ok := p.overrides[canonical]; ok {
				value = override
			}
			if p.prefix != "" {
				p.annotations = append(p.annotations, md.Variable{
					Token: fmt.Sprintf("%sn%dend", p.prefix, index), Name: canonical, Value: value,
				})
			}
		}
		p.aliases[name] = index
	}
	p.variables[index].Occurrences++
	item := p.items[index] // Never modify a stored item or a caller-owned map.
	if value, ok := p.overrides[item.Name]; ok {
		item.Content = value
	}
	if p.prefix != "" {
		item.Content = p.annotations[index].Token
	}
	return item, nil
}

// expandPageKnowledge follows exactly the same macros and fence rules as normal
// rendering. Annotations use random source tokens, not a search for saved values.
func expandPageKnowledge(
	ctx context.Context,
	content knowledgeContent,
	source string,
	overrides map[string]string,
	annotate bool,
) (expandedPageKnowledge, error) {
	if err := validateVariableOverrides(overrides); err != nil {
		return expandedPageKnowledge{}, err
	}
	collector := &pageKnowledge{
		knowledgeContent: content,
		overrides:        overrides,
		indices:          make(map[string]int),
		aliases:          make(map[string]int),
	}
	if annotate {
		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			return expandedPageKnowledge{}, err
		}
		collector.prefix = "lorevariable" + hex.EncodeToString(nonce[:])
	}
	expanded, err := expandKnowledgeMarkdown(ctx, collector, source, nil, 0)
	if err != nil {
		return expandedPageKnowledge{}, err
	}
	for name := range overrides {
		if _, used := collector.indices[name]; !used {
			return expandedPageKnowledge{}, domain.NewValidationError("variables", "Only variables used by this page can be overridden.")
		}
	}
	return expandedPageKnowledge{Markdown: expanded, Variables: collector.variables, Annotations: collector.annotations}, nil
}

// validateVariableOverrides bounds untrusted export input before expansion. An
// explicitly empty string is a valid replacement; whitespace is not trimmed.
func validateVariableOverrides(overrides map[string]string) error {
	if len(overrides) > maxExportVariables {
		return domain.NewValidationError("variables", "Override at most 128 variables per export.")
	}
	total := 0
	for name, value := range overrides {
		validName := name != "" && len(name) <= 128 && strings.TrimSpace(name) == name &&
			utf8.ValidString(name) && !strings.ContainsAny(name, "\x00\r\n{}")
		if !validName {
			return domain.NewValidationError("variables", "Use an exact variable name of at most 128 UTF-8 bytes.")
		}
		if !utf8.ValidString(value) || strings.ContainsRune(value, '\x00') {
			return domain.NewValidationError("variables", "Variable values must contain valid UTF-8 text without null characters.")
		}
		if len(value) > maxExportVariableBytes {
			return domain.NewValidationError("variables", "Each temporary value must be at most 8 KiB.")
		}
		total += len(name) + len(value)
		if total > maxExportVariableTotalBytes {
			return domain.NewValidationError("variables", "Temporary values must total at most 64 KiB.")
		}
	}
	return nil
}
