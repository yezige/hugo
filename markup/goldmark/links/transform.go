package images

import (
	"github.com/yezige/goldmark"
	"github.com/yezige/goldmark/ast"
	"github.com/yezige/goldmark/parser"
	"github.com/yezige/goldmark/text"
	"github.com/yezige/goldmark/util"
)

type (
	linksExtension struct {
		wrapStandAloneImageWithinParagraph bool
	}
)

const (
	// Used to signal to the rendering step that an image is used in a block context.
	// Dont's change this; the prefix must match the internalAttrPrefix in the root goldmark package.
	AttrIsBlock = "_h__isBlock"
)

func New(wrapStandAloneImageWithinParagraph bool) goldmark.Extender {
	return &linksExtension{wrapStandAloneImageWithinParagraph: wrapStandAloneImageWithinParagraph}
}

func (e *linksExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&Transformer{wrapStandAloneImageWithinParagraph: e.wrapStandAloneImageWithinParagraph}, 300),
		),
	)
}

type Transformer struct {
	wrapStandAloneImageWithinParagraph bool
}

// Transform transforms the provided Markdown AST.
func (t *Transformer) Transform(doc *ast.Document, reader text.Reader, pctx parser.Context) {
	ast.Walk(doc, func(node ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}

		if n, ok := node.(*ast.Image); ok {
			parent := n.Parent()

			if !t.wrapStandAloneImageWithinParagraph {
				isBlock := parent.ChildCount() == 1
				if isBlock {
					n.SetAttributeString(AttrIsBlock, true)
				}

				if isBlock && parent.Kind() == ast.KindParagraph {
					for _, attr := range parent.Attributes() {
						// Transfer any attribute set down to the image.
						// Image elements does not support attributes on its own,
						// so it's safe to just set without checking first.
						n.SetAttribute(attr.Name, attr.Value)
					}
					grandParent := parent.Parent()
					grandParent.ReplaceChild(grandParent, parent, n)
				}
			}

		}

		return ast.WalkContinue, nil

	})

}
