package dandelion_workflow

import (
	"cluster_manager/internal/control_plane/workflow"
	"fmt"
	"github.com/sirupsen/logrus"
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

	SrcStmt       *Statement
	SrcStmtOutIdx int
	SrcTask       *workflow.Task
	SrcTaskOutIdx int
}

type OutputDescriptor struct {
	dest     string
	name     string
	feedback bool

	SrcStmt       *Statement
	SrcStmtOutIdx int

	DestStmt      []*Statement
	DestStmtInIdx []int
	isCompOutput  bool
}

type Statement struct {
	Name string
	kind StatementType

	Args       []InputDescriptor
	Rets       []OutputDescriptor
	statements []*Statement // is nil unless type is loop

	procFlags       uint8  // used for processing
	parentProcessed []bool // used for partitioning
}

type Composition struct {
	Name       string
	params     []string
	returns    []string
	Statements []*Statement

	Consumers      []*Statement
	ConsumerArgIdx []int
	ConsumerOutIdx []int

	outStmts      []*Statement
	outStmtRetIdx []int
}

type DandelionWorkflow struct {
	Name          string
	FunctionDecls []*FunctionDecl
	Compositions  []*Composition

	validated bool
}

func (s *Statement) hasOneConsumer() bool {
	if len(s.Rets) == 0 {
		return false
	}
	var consumer *Statement
	for _, ret := range s.Rets {
		if ret.isCompOutput {
			return false
		}
		for _, dest := range ret.DestStmt {
			if consumer == nil {
				consumer = dest
			} else if consumer != dest {
				return false
			}
		}
	}
	return true
}
func (s *Statement) hasOneParent() bool {
	if len(s.Args) == 0 {
		return false
	}
	var parent *Statement
	for _, arg := range s.Args {
		if parent == nil {
			parent = arg.SrcStmt
		} else if parent != arg.SrcStmt {
			return false
		}
	}
	return true
}

func (s *Statement) checkFunctionDecl(functionDecls []*FunctionDecl) bool {
	if s.kind == FunctionAppl {
		for _, fd := range functionDecls {
			if fd.name == s.Name {
				if len(fd.params) != len(s.Args) {
					return false
				}
				if len(fd.returns) != len(s.Rets) {
					return false
				}
				for _, arg := range s.Args {
					if !slices.Contains(fd.params, arg.name) {
						return false
					}
				}
				for _, ret := range s.Rets {
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
	stack := make([]*Statement, len(c.Consumers))
	copy(stack, c.Consumers)
	stackSize := len(stack)

	for stackSize > 0 {
		stmt := stack[stackSize-1]

		if stmt.procFlags == 0 {
			stmt.procFlags = 1

			for _, ret := range stmt.Rets {
				for _, dep := range ret.DestStmt {
					if dep.procFlags == 0 {
						stack = append(stack, dep)
						stackSize++
					} else if dep.procFlags == 1 { // consumer is on DFS path
						return true
					}
				}
			}
		} else {
			stmt.procFlags = 2
			stackSize--
			stack = stack[:stackSize]
		}
	}

	return false
}
func (dwf *DandelionWorkflow) Process() error {
	if dwf.validated {
		return nil
	}

	// validate workflow semantics and set Consumers
	for _, c := range dwf.Compositions {
		logrus.Tracef("Processing composition %s", c.Name)
		funcOutputs := make(map[string]*OutputDescriptor)

		for _, stmt := range c.Statements {
			// compare functions against the function declarations
			if !stmt.checkFunctionDecl(dwf.FunctionDecls) {
				return fmt.Errorf("function %s not declared doesn't match declaration", stmt.Name)
			}

			// collect (intermediate) outputs
			for retIdx := range stmt.Rets {
				stmt.Rets[retIdx].SrcStmt = stmt
				stmt.Rets[retIdx].SrcStmtOutIdx = retIdx

				_, ok := funcOutputs[stmt.Rets[retIdx].dest]
				if !ok {
					funcOutputs[stmt.Rets[retIdx].dest] = &stmt.Rets[retIdx]
				} else {
					return fmt.Errorf("output %s is defined more than once", stmt.Rets[retIdx].dest)
				}
			}
		}

		// check that inputs match an output or composition parameter + set Consumers
		for _, stmt := range c.Statements {
			stmt.parentProcessed = make([]bool, len(stmt.Args))
			for argIdx := range stmt.Args {
				srcDescriptor, ok := funcOutputs[stmt.Args[argIdx].src]
				if ok {
					srcDescriptor.DestStmt = append(srcDescriptor.DestStmt, stmt)
					srcDescriptor.DestStmtInIdx = append(srcDescriptor.DestStmtInIdx, argIdx)
					stmt.Args[argIdx].SrcStmt = srcDescriptor.SrcStmt
					stmt.Args[argIdx].SrcStmtOutIdx = srcDescriptor.SrcStmtOutIdx
				} else {
					isInput := false
					for paramIdx, param := range c.params {
						if param == stmt.Args[argIdx].src {
							isInput = true
							c.Consumers = append(c.Consumers, stmt)
							c.ConsumerArgIdx = append(c.ConsumerArgIdx, argIdx)
							c.ConsumerOutIdx = append(c.ConsumerOutIdx, paramIdx)
							stmt.Args[argIdx].SrcStmtOutIdx = paramIdx
							break
						}
					}
					if !isInput {
						return fmt.Errorf("cannot find source for argument %s", stmt.Args[argIdx].src)
					}
				}
			}
		}

		// check that composition returns match a statement output
		for _, ret := range c.returns {
			srcDescriptor, ok := funcOutputs[ret]
			if ok {
				srcDescriptor.isCompOutput = true
				c.outStmts = append(c.outStmts, srcDescriptor.SrcStmt)
				c.outStmtRetIdx = append(c.outStmtRetIdx, srcDescriptor.SrcStmtOutIdx)
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

	dwf.validated = true
	return nil
}

func (dwf *DandelionWorkflow) CheckFunctionDeclarations(serviceList []string) bool {
	for _, decl := range dwf.FunctionDecls {
		if !slices.Contains(serviceList, decl.name) {
			return false
		}
	}
	return true
}

func (dwf *DandelionWorkflow) ExportWorkflow(method PartitionMethod) ([]*workflow.Workflow, [][]*workflow.Task, error) {
	err := dwf.Process()
	if err != nil {
		return nil, nil, err
	}

	var partitionFunc partitionFunction
	var methodStr string
	switch method {
	case FullPartition:
		partitionFunc = fullPartition
		methodStr = "fullPartition"
	case NoPartition:
		partitionFunc = noPartition
		methodStr = "noPartition"
	case ConsumerBased:
		partitionFunc = consumerBased
		methodStr = "consumerBased"
	default:
		return nil, nil, fmt.Errorf("unknown partition method %v", method)
	}

	logrus.Tracef("Exporting workflow and task objects from dandelion workflow %s using partition method %s", dwf.Name, methodStr)
	var wfs []*workflow.Workflow
	var tasks [][]*workflow.Task
	tasksTotal := 0
	for i, c := range dwf.Compositions {
		wf := &workflow.Workflow{
			Name:   fmt.Sprintf("%s_%s", dwf.Name, c.Name),
			NumIn:  uint32(len(c.params)),
			NumOut: uint32(len(c.returns)),
		}
		tasks = append(tasks, partitionFunc(c, wf))
		wfs = append(wfs, wf)
		tasksTotal += len(tasks[i])
	}
	logrus.Tracef("Exported %d workflows and %d tasks.", len(wfs), tasksTotal)

	return wfs, tasks, nil
}
