package workflow

import (
	"bufio"
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
	_, _, err := p.l.ExpectIgnoreSpace(BOPEN, "bracket opening list of strings")
	if err != nil {
		return nil, err
	}

	var list []string
	for {
		t, s, err := p.l.SafeScanIgnoreSpace()
		if err != nil {
			return nil, err
		}
		if t == BCLOSE {
			break
		}
		if t == IDENT {
			list = append(list, s)
		} else {
			return nil, fmt.Errorf("expected identifier or closing bracket, got %s", s)
		}
	}

	return list, nil
}

func (p *Parser) parseFunctionDecl() (*FunctionDecl, error) {
	// NOTE: expects the ( and :function tokens to be consumed already

	// function Name
	_, name, err := p.l.ExpectIgnoreSpace(IDENT, "function Name identifier")
	if err != nil {
		return nil, err
	}

	// function params
	params, err := p.parseStringList()
	if err != nil {
		return nil, err
	}

	// function returns
	_, _, err = p.l.ExpectIgnoreSpace(TOTHIN, "->")
	if err != nil {
		return nil, err
	}
	returns, err := p.parseStringList()
	if err != nil {
		return nil, err
	}

	return &FunctionDecl{name, params, returns}, nil
}

func (p *Parser) parseInputDescriptor() (InputDescriptor, error) {
	// NOTE: expects the ( token to be consumed already

	// loop condition (:until_empty, :until_empty_item, none)
	var loopCond LoopCond
	t, s, err := p.l.SafeScanIgnoreSpace()
	if err != nil {
		return InputDescriptor{}, err
	}
	switch t {
	case UNTIL_EMPTY:
		loopCond = LoopCondUntilEmpty
	case UNTIL_EMPTY_ITEM:
		loopCond = LoopCondUntilItemEmpty
	default:
		loopCond = LoopCondNone
	}

	// sharding (:all, :keyed, :each, none)
	var sharding Sharding
	shardingProvided := true
	if loopCond != LoopCondNone {
		t, s, err = p.l.SafeScanIgnoreSpace()
		if err != nil {
			return InputDescriptor{}, err
		}
	}
	switch t {
	case ALL:
		sharding = ShardingAll
	case KEYED:
		sharding = ShardingKeyed
	case EACH:
		sharding = ShardingEach
	default: // use :all when none is provided
		sharding = ShardingAll
		shardingProvided = false
	}

	// input descriptor Name
	if shardingProvided {
		t, s, err = p.l.SafeScanIgnoreSpace()
		if err != nil {
			return InputDescriptor{}, err
		}
	}
	if t != IDENT {
		return InputDescriptor{}, fmt.Errorf("expected input descriptor identifier, got %s", s)
	}
	name := s

	// input descriptor identifier
	_, _, err = p.l.ExpectIgnoreSpace(FROM, "<-")
	if err != nil {
		return InputDescriptor{}, err
	}
	_, ident, err := p.l.ExpectIgnoreSpace(IDENT, "input descriptor identifier")
	if err != nil {
		return InputDescriptor{}, err
	}

	// closing bracket
	_, _, err = p.l.ExpectIgnoreSpace(BCLOSE, "bracket closing input descriptor")
	if err != nil {
		return InputDescriptor{}, err
	}

	return InputDescriptor{name, ident, sharding, loopCond}, nil
}

func (p *Parser) parseOutputDescriptor() (OutputDescriptor, error) {
	// NOTE: expects the ( token to be consumed already

	// optional :feedback keyword
	feedback := false
	t, s, err := p.l.SafeScanIgnoreSpace()
	if err != nil {
		return OutputDescriptor{}, err
	}
	if t == FEEDBACK {
		feedback = true
		t, s, err = p.l.SafeScanIgnoreSpace()
		if err != nil {
			return OutputDescriptor{}, err
		}
	}

	// output descriptor identifier
	if t != IDENT {
		return OutputDescriptor{}, fmt.Errorf("expected output descriptor identifier, got %s", s)
	}
	ident := s

	// output descriptor Name
	_, _, err = p.l.ExpectIgnoreSpace(GETS, ":=")
	if err != nil {
		return OutputDescriptor{}, err
	}
	_, name, err := p.l.ExpectIgnoreSpace(IDENT, "output descriptor identifier")
	if err != nil {
		return OutputDescriptor{}, err
	}

	// closing bracket
	_, _, err = p.l.ExpectIgnoreSpace(BCLOSE, "bracket closing input descriptor")
	if err != nil {
		return OutputDescriptor{}, err
	}

	return OutputDescriptor{ident, name, feedback}, nil
}

func (p *Parser) parseFunctionApplication() (*Statement, error) {
	// NOTE: expects the ( token to be consumed and if the following IDENT token has also been consumed
	//		 already it is expected to be buffered
	fa := Statement{kind: FunctionAppl}

	// function application Name
	if p.strBuf == "" { // -> empty buffer -> consume token
		_, s, err := p.l.ExpectIgnoreSpace(IDENT, "function application identifier")
		if err != nil {
			return nil, err
		}
		fa.Name = s
	} else { // -> use buffered token
		if p.tokenBuf != IDENT {
			return nil, fmt.Errorf("invalid token buffer")
		}
		fa.Name = p.strBuf
		p.strBuf = ""
	}

	// input descriptors
	_, _, err := p.l.ExpectIgnoreSpace(BOPEN, "bracket opening input descriptors of function application")
	if err != nil {
		return nil, err
	}
	for {
		t, s, err := p.l.SafeScanIgnoreSpace()
		if err != nil {
			return nil, err
		}

		if t == BOPEN {
			inDesc, err := p.parseInputDescriptor()
			if err != nil {
				return nil, err
			}
			fa.args = append(fa.args, inDesc)
		} else if t == BCLOSE {
			break
		} else {
			return nil, fmt.Errorf("expected bracket, got %s", s)
		}
	}

	// output descriptors
	_, _, err = p.l.ExpectIgnoreSpace(TOFAT, "=>")
	if err != nil {
		return nil, err
	}
	_, _, err = p.l.ExpectIgnoreSpace(BOPEN, "bracket opening output descriptors of function application")
	if err != nil {
		return nil, err
	}
	for {
		t, s, err := p.l.SafeScanIgnoreSpace()
		if err != nil {
			return nil, err
		}

		if t == BOPEN {
			outDesc, err := p.parseOutputDescriptor()
			if err != nil {
				return nil, err
			}
			fa.rets = append(fa.rets, outDesc)
		} else if t == BCLOSE {
			break
		} else {
			return nil, fmt.Errorf("expected bracket, got %s", s)
		}
	}

	// closing bracket
	_, _, err = p.l.ExpectIgnoreSpace(BCLOSE, "bracket closing function application")

	return &fa, nil
}

func (p *Parser) parseLoop() (*Statement, error) {
	// NOTE: expects the ( and :loop tokens to be consumed already
	l := Statement{kind: LoopStmt}

	// loop input descriptors
	_, _, err := p.l.ExpectIgnoreSpace(BOPEN, "bracket opening input descriptors of loop")
	if err != nil {
		return nil, err
	}
	for {
		t, s, err := p.l.SafeScanIgnoreSpace()
		if err != nil {
			return nil, err
		}

		if t == BOPEN {
			inDesc, err := p.parseInputDescriptor()
			if err != nil {
				return nil, err
			}
			l.args = append(l.args, inDesc)
		} else if t == BCLOSE {
			break
		} else {
			return nil, fmt.Errorf("expected bracket, got %s", s)
		}
	}

	// loop function applications
	_, _, err = p.l.ExpectIgnoreSpace(TOFAT, "=>")
	if err != nil {
		return nil, err
	}
	_, _, err = p.l.ExpectIgnoreSpace(BOPEN, "bracket opening function applications of loop")
	if err != nil {
		return nil, err
	}
	for {
		t, s, err := p.l.SafeScanIgnoreSpace()
		if err != nil {
			return nil, err
		}

		if t == BOPEN {
			funcApp, err := p.parseFunctionApplication()
			if err != nil {
				return nil, err
			}
			l.statements = append(l.statements, funcApp)
		} else if t == BCLOSE {
			break
		} else {
			return nil, fmt.Errorf("expected bracket, got %s", s)
		}
	}

	// loop output descriptors
	_, _, err = p.l.ExpectIgnoreSpace(TOFAT, "=>")
	if err != nil {
		return nil, err
	}
	_, _, err = p.l.ExpectIgnoreSpace(BOPEN, "bracket opening output descriptors of loop")
	if err != nil {
		return nil, err
	}
	for {
		t, s, err := p.l.SafeScanIgnoreSpace()
		if err != nil {
			return nil, err
		}

		if t == BOPEN {
			outDesc, err := p.parseOutputDescriptor()
			if err != nil {
				return nil, err
			}
			l.rets = append(l.rets, outDesc)
		} else if t == BCLOSE {
			break
		} else {
			return nil, fmt.Errorf("expected bracket, got %s", s)
		}
	}

	// closing bracket
	_, _, err = p.l.ExpectIgnoreSpace(BCLOSE, "bracket closing loop")
	if err != nil {
		return nil, err
	}

	return &l, nil
}

func (p *Parser) parseCompositionStatements() ([]*Statement, error) {
	_, _, err := p.l.ExpectIgnoreSpace(BOPEN, "bracket opening list of composition Statements")
	if err != nil {
		return nil, err
	}

	var statements []*Statement
	for {
		t, s, err := p.l.SafeScanIgnoreSpace()
		if err != nil {
			return nil, err
		}

		if t == BOPEN {
			t, s, err = p.l.SafeScanIgnoreSpace()
			if err != nil {
				return nil, err
			}

			if t == IDENT { // -> function application
				p.tokenBuf = IDENT
				p.strBuf = s
				funcApp, err := p.parseFunctionApplication()
				if err != nil {
					return nil, err
				}
				statements = append(statements, funcApp)
			} else if t == LOOP { // -> loop statement
				loop, err := p.parseLoop()
				if err != nil {
					return nil, err
				}
				statements = append(statements, loop)
			} else {
				return nil, fmt.Errorf("expected function application or loop, got %s", s)
			}
		} else if t == BCLOSE {
			break
		} else {
			return nil, fmt.Errorf("expected bracket, got %s", s)
		}
	}

	return statements, nil
}

func (p *Parser) parseComposition() (*Composition, error) {
	// NOTE: expects the ( and :composition tokens to be consumed already

	// composition Name
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
	_, _, err = p.l.ExpectIgnoreSpace(TOTHIN, "->")
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
		statements: statements,
	}, nil
}

func (p *Parser) Parse() (*Workflow, error) {
	w := Workflow{}

	logrus.Tracef("Parsing workflow")
	for {
		// expect ( or EOF
		t, s := p.l.ScanIgnoreSpace()
		if t == EOF {
			break
		} else if t != BOPEN {
			return nil, fmt.Errorf("expected bracket denoting start of function declaration or composition, got %s", s)
		}

		// parse function declaration or composition
		t, s, err := p.l.SafeScanIgnoreSpace()
		if err != nil {
			return nil, err
		}
		if t == FUNC {
			function, err := p.parseFunctionDecl()
			if err != nil {
				return nil, err
			}
			w.FunctionDecls = append(w.FunctionDecls, function)
		} else if t == COMP {
			composition, err := p.parseComposition()
			if err != nil {
				return nil, err
			}
			w.Compositions = append(w.Compositions, composition)
		} else {
			return nil, fmt.Errorf("expected :function or :composition, got %s", s)
		}

		// expect )
		_, _, err = p.l.ExpectIgnoreSpace(BCLOSE, "closing bracket denoting end of function declaration or composition")
		if err != nil {
			return nil, err
		}
	}

	return &w, nil
}
