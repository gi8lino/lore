package markdown

import (
	"bytes"
	"html"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	goldhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Variable binds a request-local source token to its value and identity. Tokens
// come from macro expansion, so equal ordinary text is never annotated.
type Variable struct {
	Token string
	Name  string
	Value string
}

type variableRange struct {
	start, end int
	name       string
}

var variableTokenPattern = regexp.MustCompile(`lorevariable[0-9a-f]{32}n[0-9]+end`)

// resolveVariableTokens restores the real Markdown before Goldmark parses it,
// retaining byte ranges for text annotations. Replacements are never rescanned.
func resolveVariableTokens(source string, variables []Variable) (string, []variableRange) {
	if len(variables) == 0 {
		return source, nil
	}
	byToken := make(map[string]Variable, len(variables))
	for _, variable := range variables {
		byToken[variable.Token] = variable
	}
	var output strings.Builder
	var ranges []variableRange
	position := 0
	for _, match := range variableTokenPattern.FindAllStringIndex(source, -1) {
		variable, ok := byToken[source[match[0]:match[1]]]
		if !ok {
			continue
		}
		output.WriteString(source[position:match[0]])
		start := output.Len()
		output.WriteString(variable.Value)
		ranges = append(ranges, variableRange{start: start, end: output.Len(), name: variable.Name})
		position = match[1]
	}
	output.WriteString(source[position:])
	return output.String(), ranges
}

var kindVariable = ast.NewNodeKind("LoreVariable")

type variableNode struct {
	ast.BaseInline
	name string
}

func (n *variableNode) Kind() ast.NodeKind { return kindVariable }
func (n *variableNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"Name": n.name}, nil)
}

type variableTransformer struct{ ranges []variableRange }

// Transform wraps only the text segments originating from a variable. Image alt
// text and code spans have specialized renderers and must not acquire AST spans.
func (v variableTransformer) Transform(document *ast.Document, _ text.Reader, _ parser.Context) {
	if len(v.ranges) == 0 {
		return
	}
	var nodes []*ast.Text
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node.Kind() {
		case ast.KindImage, ast.KindCodeSpan:
			return ast.WalkSkipChildren, nil
		}
		if node, ok := node.(*ast.Text); ok {
			nodes = append(nodes, node)
		}
		return ast.WalkContinue, nil
	})
	for _, node := range nodes {
		v.wrapText(node)
	}
}

func (v variableTransformer) wrapText(node *ast.Text) {
	start, stop := node.Segment.Start, node.Segment.Stop
	if start == stop || node.Segment.Padding != 0 {
		return
	}
	parent := node.Parent()
	position := start
	changed := false
	appendText := func(from, to int, name string) {
		if from >= to {
			return
		}
		part := ast.NewTextSegment(text.NewSegment(from, to))
		part.SetRaw(node.IsRaw())
		if to == stop {
			part.SetSoftLineBreak(node.SoftLineBreak())
			part.SetHardLineBreak(node.HardLineBreak())
		}
		if name == "" {
			parent.InsertBefore(parent, node, part)
			return
		}
		span := &variableNode{name: name}
		span.AppendChild(span, part)
		parent.InsertBefore(parent, node, span)
	}
	for _, region := range v.ranges {
		from, to := max(position, region.start), min(stop, region.end)
		if from >= to {
			continue
		}
		appendText(position, from, "")
		appendText(from, to, region.name)
		position = to
		changed = true
	}
	if !changed {
		return
	}
	appendText(position, stop, "")
	parent.RemoveChild(parent, node)
}

type variableNodeRenderer struct{ ranges []variableRange }

func (v variableNodeRenderer) RegisterFuncs(register renderer.NodeRendererFuncRegisterer) {
	register.Register(kindVariable, v.renderVariable)
	if len(v.ranges) != 0 {
		register.Register(ast.KindCodeSpan, v.renderCodeSpan)
	}
}

func writeVariableStart(w util.BufWriter, name string) {
	_, _ = w.WriteString(`<span class="page-variable" data-page-variable="` + html.EscapeString(name) + `">`)
}

func (v variableNodeRenderer) renderVariable(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		writeVariableStart(w, node.(*variableNode).name)
	} else {
		_, _ = w.WriteString("</span>")
	}
	return ast.WalkContinue, nil
}

// renderCodeSpan keeps Goldmark's literal-text and newline behavior while
// annotating the exact source ranges, rather than interpreting HTML inside code.
func (v variableNodeRenderer) renderCodeSpan(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	_, _ = w.WriteString("<code")
	goldhtml.RenderAttributes(w, node, goldhtml.CodeAttributeFilter)
	_, _ = w.WriteString(">")
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		segment := child.(*ast.Text).Segment
		value := segment.Value(source)
		newline := bytes.HasSuffix(value, []byte("\n"))
		stop := segment.Stop
		if newline {
			stop--
		}
		position := segment.Start
		for _, region := range v.ranges {
			from, to := max(position, region.start), min(stop, region.end)
			if from >= to {
				continue
			}
			goldhtml.DefaultWriter.RawWrite(w, source[position:from])
			writeVariableStart(w, region.name)
			goldhtml.DefaultWriter.RawWrite(w, source[from:to])
			_, _ = w.WriteString("</span>")
			position = to
		}
		goldhtml.DefaultWriter.RawWrite(w, source[position:stop])
		if newline {
			_, _ = w.WriteString(" ")
		}
	}
	_, _ = w.WriteString("</code>")
	return ast.WalkSkipChildren, nil
}

// equivalentVariableHTML makes annotations fail closed for syntax that changes
// during Lore's block preprocessing (for example a variable that contains an
// entire callout). Reading must never change just to make inspection possible.
func equivalentVariableHTML(annotated, normal string) bool {
	left, err := normalizedVariableHTML(annotated)
	if err != nil {
		return false
	}
	right, err := normalizedVariableHTML(normal)
	return err == nil && left == right
}

func normalizedVariableHTML(value string) (string, error) {
	root := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := xhtml.ParseFragment(strings.NewReader(value), root)
	if err != nil {
		return "", err
	}
	for _, node := range nodes {
		root.AppendChild(node)
	}
	var unwrap func(*xhtml.Node)
	unwrap = func(parent *xhtml.Node) {
		for node := parent.FirstChild; node != nil; {
			next := node.NextSibling
			unwrap(node)
			if node.Type == xhtml.ElementNode && node.Data == "span" && htmlAttribute(node, "class") == "page-variable" && htmlAttribute(node, "data-page-variable") != "" {
				for node.FirstChild != nil {
					child := node.FirstChild
					node.RemoveChild(child)
					parent.InsertBefore(child, node)
				}
				parent.RemoveChild(node)
			}
			node = next
		}
	}
	unwrap(root)
	var output strings.Builder
	for node := root.FirstChild; node != nil; node = node.NextSibling {
		if err := xhtml.Render(&output, node); err != nil {
			return "", err
		}
	}
	return output.String(), nil
}
