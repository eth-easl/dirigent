package workflow

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"slices"
	"sync"
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

type DataType int

const (
	Bytes DataType = iota
	DandelionSets
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

type Data interface {
	GetType() DataType
	GetData() []byte
	GetDandelionData() (*InputSet, error)
}

type BytesData struct {
	Data []byte
}

func (d *BytesData) GetType() DataType {
	return Bytes
}
func (d *BytesData) GetData() []byte {
	return d.Data
}
func (d *BytesData) GetDandelionData() (*InputSet, error) {
	return nil, fmt.Errorf("not a DandelionData object")
}

type Statement struct {
	Name string
	kind StatementType

	args       []InputDescriptor
	rets       []OutputDescriptor
	statements []*Statement // is nil unless type is loop

	consumers      []*Statement
	consumerArgIdx []int
	consumerOutIdx []int
	inData         []Data
	inDataMutex    sync.Mutex
	outData        []Data

	pFlags uint8
}

type Composition struct {
	Name       string
	params     []string
	returns    []string
	statements []*Statement

	inData         []Data
	consumers      []*Statement
	consumerArgIdx []int
	consumerOutIdx []int

	outData       []Data
	outStmts      []*Statement
	outStmtRetIdx []int
}

type Workflow struct {
	FunctionDecls []*FunctionDecl
	Compositions  []*Composition

	validated bool
}

func (s *Statement) GetInData() []Data {
	return s.inData
}
func (s *Statement) GetNumOutData() int {
	return len(s.rets)
}
func (s *Statement) SetOutData(data []Data) error {
	if len(data) != len(s.rets) {
		return fmt.Errorf("got %d output data objects, statement has %d returns", len(data), len(s.rets))
	}
	s.outData = data
	return nil
}
func (s *Statement) SetDone() []*Statement {
	var stmtRunnable []*Statement
	for i, consumer := range s.consumers {
		consumer.inDataMutex.Lock()

		consumer.inData[s.consumerArgIdx[i]] = s.outData[s.consumerOutIdx[i]]

		allTrue := true
		for _, arg := range consumer.inData {
			if arg == nil {
				allTrue = false
				break
			}
		}
		if allTrue {
			stmtRunnable = append(stmtRunnable, consumer)
		}

		consumer.inDataMutex.Unlock()
	}

	return stmtRunnable
}
func (c *Composition) GetInitialRunnable(inData []Data) ([]*Statement, error) {
	if len(inData) != len(c.params) {
		return nil, fmt.Errorf("got %d input data objects, composition has %d params", len(c.inData), len(c.params))
	}
	c.inData = inData

	var stmtRunnable []*Statement
	for i, consumer := range c.consumers {
		consumer.inData[c.consumerArgIdx[i]] = c.inData[c.consumerOutIdx[i]]

		allTrue := true
		for _, arg := range consumer.inData {
			if arg == nil {
				allTrue = false
				break
			}
		}
		if allTrue {
			stmtRunnable = append(stmtRunnable, consumer)
		}
	}
	return stmtRunnable, nil
}
func (c *Composition) CollectOutData() []Data {
	if c.outData != nil {
		return c.outData
	}

	c.outData = make([]Data, len(c.outStmts))
	for i, outStmt := range c.outStmts {
		c.outData[i] = outStmt.outData[c.outStmtRetIdx[i]]
	}
	return c.outData
}
func (c *Composition) GetNumStatements() int {
	return len(c.statements)
}
func (w *Workflow) GetFunctionNames() []string {
	var names []string
	for _, decl := range w.FunctionDecls {
		names = append(names, decl.name)
	}
	return names
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
	stack := make([]*Statement, len(c.consumers))
	copy(stack, c.consumers)
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

	type outLoc struct {
		stmt    *Statement
		dataIdx int
	}

	// validate workflow semantics and set consumers
	for _, c := range w.Compositions {
		logrus.Tracef("Processing composition %s", c.Name)
		funcOutputs := make(map[string]outLoc)

		for _, stmt := range c.statements {
			// compare functions against the function declarations
			if !stmt.checkFunctionDecl(w.FunctionDecls) {
				return fmt.Errorf("function %s not declared doesn't match declaration", stmt.Name)
			}

			// collect (intermediate) outputs
			for retIdx, ret := range stmt.rets {
				_, ok := funcOutputs[ret.dest]
				if !ok {
					funcOutputs[ret.dest] = outLoc{stmt, retIdx}
				} else {
					return fmt.Errorf("output %s is defined more than once", ret.dest)
				}
			}
		}

		// check that inputs match an output or composition parameter + set consumers
		for _, stmt := range c.statements {
			stmt.inData = make([]Data, len(stmt.args))
			for argIdx, arg := range stmt.args {
				srcLoc, ok := funcOutputs[arg.src]
				if !ok {
					isInput := false
					for paramIdx, param := range c.params {
						if param == arg.src {
							isInput = true
							c.consumers = append(c.consumers, stmt)
							c.consumerArgIdx = append(c.consumerArgIdx, argIdx)
							c.consumerOutIdx = append(c.consumerOutIdx, paramIdx)
							break
						}
					}
					if !isInput {
						return fmt.Errorf("cannot find source for argument %s", arg.src)
					}
				} else {
					srcLoc.stmt.consumers = append(srcLoc.stmt.consumers, stmt)
					srcLoc.stmt.consumerArgIdx = append(srcLoc.stmt.consumerArgIdx, argIdx)
					srcLoc.stmt.consumerOutIdx = append(srcLoc.stmt.consumerOutIdx, srcLoc.dataIdx)
				}
			}
		}

		// check that composition returns match a statement output
		for _, ret := range c.returns {
			srcLoc, ok := funcOutputs[ret]
			if ok {
				c.outStmts = append(c.outStmts, srcLoc.stmt)
				c.outStmtRetIdx = append(c.outStmtRetIdx, srcLoc.dataIdx)
			} else {
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
