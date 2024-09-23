package workflow

import (
	"fmt"
	"slices"
)

type Sharding int

const (
	ShardingAll Sharding = iota
	ShardingKeyed
	ShardingEach
)

type LoopCond int

const (
	LoopCondNone LoopCond = iota
	LoopCondUntilEmpty
	LoopCondUntilItemEmpty
)

type StatementType int

const (
	FunctionAppl StatementType = iota
	LoopStmt
)

type FunctionDecl struct {
	name    string
	params  []string
	returns []string
}

type InputDescriptor struct {
	name     string
	src      string
	sharding Sharding
	loopCond LoopCond
}

type OutputDescriptor struct {
	dest     string
	name     string
	feedback bool
}

type Statement struct {
	Name string
	kind StatementType

	args       []InputDescriptor
	rets       []OutputDescriptor
	statements []*Statement

	consumers      []*Statement
	consumerArgIdx []int
	argAvail       []bool

	pFlags uint8
}

type Composition struct {
	name       string
	params     []string
	returns    []string
	Statements []*Statement

	consumers      []*Statement
	consumerArgIdx []int
}

type Workflow struct {
	FunctionDecls []*FunctionDecl
	Compositions  []*Composition

	validated bool
}

// TODO: SetDone() is not thread safe
func (s *Statement) SetDone() []*Statement {
	var stmtRunnable []*Statement
	for i, consumer := range s.consumers {
		consumer.argAvail[s.consumerArgIdx[i]] = true

		allTrue := true
		for _, arg := range consumer.argAvail {
			if !arg {
				allTrue = false
				break
			}
		}
		if allTrue {
			stmtRunnable = append(stmtRunnable, consumer)
		}
	}
	return stmtRunnable
}
func (c *Composition) GetInitialRunnable() []*Statement {
	var stmtRunnable []*Statement
	for i, consumer := range c.consumers {
		consumer.argAvail[c.consumerArgIdx[i]] = true

		allTrue := true
		for _, arg := range consumer.argAvail {
			if !arg {
				allTrue = false
				break
			}
		}
		if allTrue {
			stmtRunnable = append(stmtRunnable, consumer)
		}
	}
	return stmtRunnable
}

func (s *Statement) checkFunctionDecl(functionDecls []*FunctionDecl) bool {
	if s.kind == FunctionAppl {
		for _, fd := range functionDecls {
			if fd.name == s.Name {
				if len(fd.params) != len(s.args) {
					return false
				}
				if len(fd.returns) != len(s.rets) {
					return false
				}
				for _, arg := range s.args {
					if !slices.Contains(fd.params, arg.name) {
						return false
					}
				}
				for _, ret := range s.rets {
					if !slices.Contains(fd.returns, ret.name) {
						return false
					}
				}
				return true
			}
		}
	} else {
		for _, loopStmt := range s.statements {
			if loopStmt.checkFunctionDecl(functionDecls) {
				return false
			}
		}
	}

	return false
}
func (c *Composition) checkCircularDependency() bool {
	stack := c.consumers
	stackSize := len(stack)

	for stackSize > 0 {
		stmt := stack[stackSize-1]

		if stmt.pFlags == 0 {
			stmt.pFlags = 1

			for _, dep := range stmt.consumers {
				if dep.pFlags == 0 {
					stack = append(stack, dep)
					stackSize++
				} else if dep.pFlags == 1 { // consumer is on DFS path
					return true
				}
			}
		} else {
			stmt.pFlags = 2
			stackSize--
			stack = stack[:stackSize]
		}
	}

	return false
}
func (w *Workflow) Process() error {
	if w.validated {
		return nil
	}

	// validate workflow semantics and set consumers
	for _, c := range w.Compositions {
		funcOutputs := make(map[string]*Statement)

		for _, stmt := range c.Statements {
			// compare functions against the function declarations
			if !stmt.checkFunctionDecl(w.FunctionDecls) {
				return fmt.Errorf("function %s not declared doesn't match declaration", stmt.Name)
			}

			// collect (intermediate) outputs
			for _, ret := range stmt.rets {
				_, ok := funcOutputs[ret.dest]
				if !ok {
					funcOutputs[ret.dest] = stmt
				} else {
					return fmt.Errorf("output %s is defined more than once", ret.dest)
				}
			}
		}

		// check that inputs match an output or composition parameter + set consumers
		for _, stmt := range c.Statements {
			stmt.argAvail = make([]bool, len(stmt.args))
			for argIdx, arg := range stmt.args {
				srcStmt, ok := funcOutputs[arg.src]
				if !ok {
					if slices.Contains(c.params, arg.src) {
						c.consumers = append(c.consumers, stmt)
						c.consumerArgIdx = append(c.consumerArgIdx, argIdx)
					} else {
						return fmt.Errorf("cannot find source for argument %s", arg.src)
					}
				} else {
					srcStmt.consumers = append(srcStmt.consumers, stmt)
					srcStmt.consumerArgIdx = append(srcStmt.consumerArgIdx, argIdx)
				}
			}
		}

		// check that composition returns match a statement output
		for _, ret := range c.returns {
			_, ok := funcOutputs[ret]
			if !ok {
				return fmt.Errorf("cannot find source for composition output %s", ret)
			}
		}

		// check for absence circular dependencies
		if c.checkCircularDependency() {
			return fmt.Errorf("circular dependency detected")
		}

		// TODO: functions without input/output
	}

	w.validated = true
	return nil
}

func (w *Workflow) GetFunctionNames() []string {
	var names []string
	for _, decl := range w.FunctionDecls {
		names = append(names, decl.name)
	}
	return names
}

func (w *Workflow) getFunctionDecl(name string) *FunctionDecl {
	for _, decl := range w.FunctionDecls {
		if decl.name == name {
			return decl
		}
	}
	return nil
}
func (fd *FunctionDecl) export() string {
	str := fmt.Sprintf("(:function %s(", fd.name)
	for i, param := range fd.params {
		str += param
		if i != len(fd.params)-1 {
			str += " "
		}
	}
	str += ")->("
	for i, ret := range fd.returns {
		str += ret
		if i != len(fd.returns)-1 {
			str += " "
		}
	}
	str += "))"
	return str
}
func (s *Statement) export() (string, []string) {
	if s.kind == FunctionAppl {
		str := fmt.Sprintf("(%s(", s.Name)
		for _, arg := range s.args {
			str += "("
			switch arg.sharding {
			case ShardingKeyed:
				str += ":keyed "
			case ShardingEach:
				str += ":each "
			}
			str += fmt.Sprintf("%s<-%s)", arg.name, arg.src)
		}
		str += ")=>("
		for _, ret := range s.rets {
			str += fmt.Sprintf("(%s:=%s)", ret.dest, ret.name)
		}
		str += "))"

		return str, []string{s.Name}
	} else {
		var funcUsed []string
		str := "(:loop ("
		for _, arg := range s.args {
			str += "("
			switch arg.loopCond {
			case LoopCondUntilEmpty:
				str += ":until_empty "
			case LoopCondUntilItemEmpty:
				str += ":until_empty_item "
			}
			str += fmt.Sprintf("%s <- %s)", arg.name, arg.src)
		}
		str += ")=>("
		for _, stmt := range s.statements {
			lStr, lFuncUsed := stmt.export()
			for _, f := range lFuncUsed {
				funcUsed = append(funcUsed, f)
			}
			str += lStr
		}
		str += ")=>("
		for _, ret := range s.rets {
			str += "("
			if ret.feedback {
				str += ":feedback "
			}
			str += fmt.Sprintf("%s := %s)", ret.dest, ret.name)
		}
		str += "))"
		return str, funcUsed
	}

}
func (w *Workflow) ExportStatements(exportName string, exportStmts []*Statement) (string, error) {
	// make sure workflow is validated
	err := w.Process()
	if err != nil {
		return "", err
	}

	funcDecls := make(map[string]string)
	compositionContent := ""
	stmtArgs := make(map[string]int)
	stmtRets := make(map[string]int)

	// export Statements + collect statement arguments and returns
	for _, stmt := range exportStmts {
		stmtStr, stmtFuncUsed := stmt.export()
		for _, f := range stmtFuncUsed {
			_, ok := funcDecls[f]
			if !ok {
				funcDecls[f] = w.getFunctionDecl(f).export()
			}
		}
		compositionContent += stmtStr

		for _, arg := range stmt.args {
			stmtArgs[arg.src]++
		}
		for _, ret := range stmt.rets {
			stmtRets[ret.dest]++
		}
	}

	// determine which inputs and outputs come from the composition, i.e. are not intermediate between functions
	compositionInput := ""
	compositionOutput := ""
	for arg := range stmtArgs {
		_, ok := stmtRets[arg]
		if ok {
			delete(stmtRets, arg)
		} else {
			compositionInput += arg + " "
		}
	}
	if len(compositionInput) > 0 {
		compositionInput = compositionInput[:len(compositionInput)-1]
	}
	for ret := range stmtRets {
		compositionOutput += ret + " "
	}
	if len(compositionOutput) > 0 {
		compositionOutput = compositionOutput[:len(compositionOutput)-1]
	}

	// build export string
	exportStr := ""
	for _, str := range funcDecls {
		exportStr += str
	}
	exportStr += fmt.Sprintf(" (:composition %s(%s)->(%s)(%s))", exportName, compositionInput, compositionOutput, compositionContent)

	return exportStr, nil
}
