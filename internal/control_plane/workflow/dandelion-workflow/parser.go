package dandelion_workflow

import (
	"bufio"
	"cluster_manager/internal/control_plane/workflow"
	"fmt"
	"github.com/sirupsen/logrus"
)

type Parser struct {
	l        *Lexer
	tokenBuf Token
	strBuf   string
}

func NewParser(in *bufio.Reader) *Parser {
	return &Parser{
		l: NewLexer(in),
	}
}

func (p *Parser) parseStringList() ([]string, error) {
	_, _, err := p.l.ExpectIgnoreSpace(POPEN, "bracket opening list of strings")
	if err != nil {
		return nil, err
	}

	var list []string
	for {
		t, s, err := p.l.SafeScanIgnoreSpace()
		if err != nil {
			return nil, err
		}
		if t == PCLOSE {
			break
		}
		if len(list) == 0 {
			if t == IDENT {
				list = append(list, s)
			} else {
				return nil, fmt.Errorf("expected identifier or closing bracket, got %s at %s", s, p.l.PositionStr())
			}
		} else {
			if t == COMMA {
				_, s, err := p.l.ExpectIgnoreSpace(IDENT, "identifier after comma")
				if err != nil {
					return nil, err
				}
				list = append(list, s)
			} else {
				return nil, fmt.Errorf("expected comma separated identifier or closing bracket, got %s at %s", s, p.l.PositionStr())
			}
		}
	}

	return list, nil
}

func (p *Parser) parseFunctionDecl() (*FunctionDecl, error) {
	// NOTE: expects the 'function' keyword to be consumed already

	// function name
	_, name, err := p.l.ExpectIgnoreSpace(IDENT, "function name identifier")
	if err != nil {
		return nil, err
	}

	// function params
	params, err := p.parseStringList()
	if err != nil {
		return nil, err
	}

	// function returns
	_, _, err = p.l.ExpectIgnoreSpace(TO, "=>")
	if err != nil {
		return nil, err
	}
	returns, err := p.parseStringList()
	if err != nil {
		return nil, err
	}

	// closing semicolon
	_, _, err = p.l.ExpectIgnoreSpace(SEMICOLON, "semicolon closing function declaration")
	if err != nil {
		return nil, err
	}

	return &FunctionDecl{name, params, returns}, nil
}

func (p *Parser) parseInputDescriptor() (InputDescriptor, error) {
	// input name
	t, name, err := p.l.ExpectIgnoreSpace(IDENT, "input name")
	if err != nil {
		return InputDescriptor{}, err
	}

	// sharding (all, keyed, each, any none)
	_, _, err = p.l.ExpectIgnoreSpace(EQUALS, "=")
	var sharding workflow.Sharding
	t, s, err := p.l.SafeScanIgnoreSpace()
	if err != nil {
		return InputDescriptor{}, err
	}
	switch t {
	case ALL:
		sharding = workflow.ShardingAll
	case KEYED:
		sharding = workflow.ShardingKeyed
	case EACH:
		sharding = workflow.ShardingEach
	case ANY:
		sharding = workflow.ShardingAny
	default:
		return InputDescriptor{}, fmt.Errorf("expected sharding, got %s at %s", s, p.l.PositionStr())
	}

	// input source identifier
	_, ident, err := p.l.ExpectIgnoreSpace(IDENT, "input source identifier")
	if err != nil {
		return InputDescriptor{}, err
	}

	return InputDescriptor{name: name, src: ident, sharding: sharding}, nil
}

func (p *Parser) parseOutputDescriptor() (OutputDescriptor, error) {
	// destination identifier
	_, dest, err := p.l.ExpectIgnoreSpace(IDENT, "output destination identifier")
	if err != nil {
		return OutputDescriptor{}, err
	}

	// output name
	_, _, err = p.l.ExpectIgnoreSpace(EQUALS, "=")
	if err != nil {
		return OutputDescriptor{}, err
	}
	_, name, err := p.l.ExpectIgnoreSpace(IDENT, "output name")
	if err != nil {
		return OutputDescriptor{}, err
	}

	return OutputDescriptor{dest: dest, name: name}, nil
}

func (p *Parser) parseFunctionApplication() (*Statement, error) {
	fa := Statement{kind: FunctionAppl}

	// function application Name
	_, s, err := p.l.ExpectIgnoreSpace(IDENT, "function application identifier")
	if err != nil {
		return nil, err
	}
	fa.Name = s

	// input descriptors
	_, _, err = p.l.ExpectIgnoreSpace(POPEN, "bracket opening input descriptors of function application")
	if err != nil {
		return nil, err
	}
	for {
		t, s, err := p.l.SafePeekIgnoreSpace()
		if err != nil {
			return nil, err
		}

		if t == PCLOSE {
			p.l.ConsumePeek()
			break
		} else if len(fa.Args) > 0 {
			if t != COMMA {
				return nil, fmt.Errorf("expected next comma separated input descriptor or closing bracket, got %s at %s", s, p.l.PositionStr())
			}
			p.l.ConsumePeek()
		} else if t != IDENT {
			return nil, fmt.Errorf("expected input descriptor or closing bracket, got %s at %s", s, p.l.PositionStr())
		}

		inDesc, err := p.parseInputDescriptor()
		if err != nil {
			return nil, err
		}
		fa.Args = append(fa.Args, inDesc)
	}

	// output descriptors
	_, _, err = p.l.ExpectIgnoreSpace(TO, "=>")
	if err != nil {
		return nil, err
	}
	_, _, err = p.l.ExpectIgnoreSpace(POPEN, "bracket opening output descriptors of function application")
	if err != nil {
		return nil, err
	}
	for {
		t, s, err := p.l.SafePeekIgnoreSpace()
		if err != nil {
			return nil, err
		}

		if t == PCLOSE {
			p.l.ConsumePeek()
			break
		} else if len(fa.Rets) > 0 {
			if t != COMMA {
				return nil, fmt.Errorf("expected next comma separated output descriptor or closing bracket, got %s at %s", s, p.l.PositionStr())
			}
			p.l.ConsumePeek()
		} else if t != IDENT {
			return nil, fmt.Errorf("expected output descriptor or closing bracket, got %s at %s", s, p.l.PositionStr())
		}

		outDesc, err := p.parseOutputDescriptor()
		if err != nil {
			return nil, err
		}
		fa.Rets = append(fa.Rets, outDesc)
	}

	// closing semicolon
	_, _, err = p.l.ExpectIgnoreSpace(SEMICOLON, "semicolon closing function application")
	if err != nil {
		return nil, err
	}

	return &fa, nil
}

func (p *Parser) parseCompositionStatements() ([]*Statement, error) {
	_, _, err := p.l.ExpectIgnoreSpace(BOPEN, "brace opening list of composition Statements")
	if err != nil {
		return nil, err
	}

	var statements []*Statement
	for {
		t, s, err := p.l.SafePeekIgnoreSpace()
		if err != nil {
			return nil, err
		}

		if t == BCLOSE {
			p.l.ConsumePeek()
			break
		} else if t == IDENT {
			funcApp, err := p.parseFunctionApplication()
			if err != nil {
				return nil, err
			}
			statements = append(statements, funcApp)
		} else {
			return nil, fmt.Errorf("expected function or closing brace, got %s at %s", s, p.l.PositionStr())
		}
	}

	return statements, nil
}

func (p *Parser) parseComposition() (*Composition, error) {
	// NOTE: expects the 'composition' keyword to be consumed already

	// composition name
	_, name, err := p.l.ExpectIgnoreSpace(IDENT, "composition Name identifier")
	if err != nil {
		return nil, err
	}

	// composition params
	params, err := p.parseStringList()
	if err != nil {
		return nil, err
	}

	// composition returns
	_, _, err = p.l.ExpectIgnoreSpace(TO, "=>")
	if err != nil {
		return nil, err
	}
	returns, err := p.parseStringList()
	if err != nil {
		return nil, err
	}

	// composition Statements
	statements, err := p.parseCompositionStatements()
	if err != nil {
		return nil, err
	}

	return &Composition{
		Name:       name,
		params:     params,
		returns:    returns,
		Statements: statements,
	}, nil
}

func (p *Parser) Parse() (*DandelionWorkflow, error) {
	w := DandelionWorkflow{}

	logrus.Tracef("Parsing workflow")
	for {
		t, s := p.l.ScanIgnoreSpace()
		switch t {
		case EOF:
			return &w, nil
		case FUNC:
			function, err := p.parseFunctionDecl()
			if err != nil {
				return nil, err
			}
			w.FunctionDecls = append(w.FunctionDecls, function)
		case COMP:
			composition, err := p.parseComposition()
			if err != nil {
				return nil, err
			}
			w.Compositions = append(w.Compositions, composition)
		default:
			return nil, fmt.Errorf("expected 'function' or 'composition' keyword, got %s at %s", s, p.l.PositionStr())
		}
	}
}
