package django

import (
	"fmt"
	"github.com/flosch/pongo2/v6"
	"github.com/valyala/bytebufferpool"
	"sort"
)

type tagLayouter interface {
	Layout(string)
	DisableLayout(bool)
	SetViewBlock(string, any)
	GetViewBlock(string) (any, bool)
}

type tagLayoutNode struct {
	name string
}

func (node *tagLayoutNode) Execute(ctx *pongo2.ExecutionContext, writer pongo2.TemplateWriter) *pongo2.Error {

	context, ok := ctx.Public["ctx"].(tagLayouter)

	if !ok {
		return nil
	}

	if node.name == "none" {
		context.DisableLayout(true)
		return nil
	}

	context.DisableLayout(false)
	context.Layout(node.name)

	return nil
}

func tagLayoutParser(doc *pongo2.Parser, start *pongo2.Token, arguments *pongo2.Parser) (pongo2.INodeTag, *pongo2.Error) {
	if arguments.Count() == 0 {
		return nil, arguments.Error("Tag 'section' requires an identifier.", nil)
	}
	nameToken := arguments.MatchType(pongo2.TokenString)
	if nameToken == nil {
		return nil, arguments.Error("First argument for tag 'layout' must be an identifier.", nil)
	}

	if arguments.Remaining() != 0 {
		return nil, arguments.Error("Tag 'layout' takes exactly 1 argument (an identifier).", nil)
	}

	return &tagLayoutNode{name: nameToken.Val}, nil
}

type tagBlockNode struct {
	name    string
	score   pongo2.IEvaluator
	wrapper *pongo2.NodeWrapper
}

type blockContent struct {
	content []byte
	score   int
}

func (node *tagBlockNode) Execute(ctx *pongo2.ExecutionContext, writer pongo2.TemplateWriter) *pongo2.Error {

	context, ok := ctx.Public["ctx"].(tagLayouter)
	if !ok {
		return nil
	}

	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	err := node.wrapper.Execute(ctx, buf)
	if err != nil {
		return err
	}

	bs := make([]byte, buf.Len())
	copy(bs, buf.Bytes())

	score := 0
	if node.score != nil {
		if val, err := node.score.Evaluate(ctx); err == nil {
			score = val.Integer()
		}
	}

	content := &blockContent{
		content: bs,
		score:   score,
	}

	val, _ := context.GetViewBlock(node.name)
	contents, ok := val.([]*blockContent)
	if ok {
		contents = append(contents, content)
	} else {
		contents = []*blockContent{content}
	}
	context.SetViewBlock(node.name, contents)

	return nil
}

func tagBlockParser(doc *pongo2.Parser, start *pongo2.Token, arguments *pongo2.Parser) (pongo2.INodeTag, *pongo2.Error) {
	if arguments.Count() == 0 {
		return nil, arguments.Error("Tag 'Block' requires an identifier.", nil)
	}

	nameToken := arguments.MatchType(pongo2.TokenIdentifier)
	if nameToken == nil {
		return nil, arguments.Error("First argument for tag 'block' must be an identifier.", nil)
	}

	scoreIEval, err := arguments.ParseExpression()
	//if err != nil {
	//	return nil, arguments.Error("Can not parse with score.", nameToken)
	//}
	wrapper, endtagargs, err := doc.WrapUntilTag("endblock")
	if err != nil {
		return nil, err
	}
	if endtagargs.Remaining() > 0 {
		endtagnameToken := endtagargs.MatchType(pongo2.TokenIdentifier)
		if endtagnameToken != nil {
			if endtagnameToken.Val != nameToken.Val {
				return nil, endtagargs.Error(fmt.Sprintf("Name for 'endblock' must equal to 'block'-tag's name ('%s' != '%s').",
					nameToken.Val, endtagnameToken.Val), nil)
			}
		}

		if endtagnameToken == nil || endtagargs.Remaining() > 0 {
			return nil, endtagargs.Error("Either no or only one argument (identifier) allowed for 'endblock'.", nil)
		}
	}
	return &tagBlockNode{name: nameToken.Val, score: scoreIEval, wrapper: wrapper}, nil
}

type tagSectionNode struct {
	name    string
	wrapper *pongo2.NodeWrapper
}

func (node *tagSectionNode) Execute(ctx *pongo2.ExecutionContext, writer pongo2.TemplateWriter) *pongo2.Error {

	context, ok := ctx.Public["ctx"].(tagLayouter)
	if !ok {
		return nil
	}

	val, _ := context.GetViewBlock(node.name)

	contents, ok := val.([]*blockContent)

	if !ok || len(contents) == 0 {
		return node.wrapper.Execute(ctx, writer)
	}

	sort.Slice(contents, func(i, j int) bool {
		return contents[i].score > contents[j].score
	})

	for _, content := range contents {
		_, _ = writer.Write(content.content)
		_, _ = writer.Write([]byte{'\n'})
	}

	return nil
}

func tagSectionParser(doc *pongo2.Parser, start *pongo2.Token, arguments *pongo2.Parser) (pongo2.INodeTag, *pongo2.Error) {
	if arguments.Count() == 0 {
		return nil, arguments.Error("Tag 'section' requires an identifier.", nil)
	}

	nameToken := arguments.MatchType(pongo2.TokenIdentifier)
	if nameToken == nil {
		return nil, arguments.Error("First argument for tag 'section' must be an identifier.", nil)
	}

	if arguments.Remaining() != 0 {
		return nil, arguments.Error("Tag 'section' takes exactly 1 argument (an identifier).", nil)
	}

	wrapper, endtagargs, err := doc.WrapUntilTag("endsection")
	if err != nil {
		return nil, err
	}
	if endtagargs.Remaining() > 0 {
		endtagnameToken := endtagargs.MatchType(pongo2.TokenIdentifier)
		if endtagnameToken != nil {
			if endtagnameToken.Val != nameToken.Val {
				return nil, endtagargs.Error(fmt.Sprintf("Name for 'endsection' must equal to 'section'-tag's name ('%s' != '%s').",
					nameToken.Val, endtagnameToken.Val), nil)
			}
		}

		if endtagnameToken == nil || endtagargs.Remaining() > 0 {
			return nil, endtagargs.Error("Either no or only one argument (identifier) allowed for 'endsection'.", nil)
		}
	}

	return &tagSectionNode{name: nameToken.Val, wrapper: wrapper}, nil
}

func init() {
	_ = pongo2.RegisterTag("layout", tagLayoutParser)
	_ = pongo2.ReplaceTag("block", tagBlockParser)
	_ = pongo2.RegisterTag("section", tagSectionParser)
}
