package handlers

import(
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"
	"sync"
	"github.com/opwire/opwire-agent/utils"
)

const DEFAULT_COMMAND string = "opwire-agent-default"

type Executor struct {
	commands map[string]CommandDescriptor
	pipeChain *PipeChain
}

type ExecutorOptions struct {
	Command CommandDescriptor
}

type CommandDescriptor struct {
	CommandString string
	subCommands []string
}

type CommandInvocation struct {
	CommandString string
	Name string
}

func NewExecutor(opts *ExecutorOptions) (*Executor, error) {
	e := &Executor{}
	if opts != nil {
		e.Register(DEFAULT_COMMAND, &opts.Command)
	}
	e.pipeChain = &PipeChain{}
	return e, nil
}

func (e *Executor) Register(name string, descriptor *CommandDescriptor) (error) {
	if e.commands == nil {
		e.commands = make(map[string]CommandDescriptor)
	}
	if descriptor != nil && len(descriptor.CommandString) > 0 {
		if cloned, err := prepareCommandDescriptor(descriptor.CommandString); err == nil {
			e.commands[name] = cloned
		}
	}
	return nil
}

func (e *Executor) Run(opts *CommandInvocation, inData []byte) ([]byte, []byte, error) {
	if descriptor, err := e.getCommandDescriptor(opts); err == nil {
		if cmds, err := buildExecCmds(descriptor); err == nil {
			count := len(cmds)
			if count > 0 {
				if count == 1 {
					return runSingleCommand(cmds[0], inData)
				}
				ib := bytes.NewBuffer(inData)
				var ob bytes.Buffer
				var eb bytes.Buffer
				err := e.pipeChain.Run(ib, &ob, &eb, cmds...)
				return ob.Bytes(), eb.Bytes(), err
			} else {
				return nil, nil, errors.New("Command not found")
			}
		} else {
			return nil, nil, err
		}
	} else {
		return nil, nil, err
	}
}

func (e *Executor) getCommandDescriptor(opts *CommandInvocation) (*CommandDescriptor, error) {
	if opts != nil && len(opts.CommandString) > 0 {
		descriptor, err := prepareCommandDescriptor(opts.CommandString)
		return &descriptor, err
	} else if len(opts.Name) > 0 {
		if descriptor, ok := e.commands[opts.Name]; ok {
			return &descriptor, nil
		}
	}
	if descriptor, ok := e.commands[DEFAULT_COMMAND]; ok {
		return &descriptor, nil
	}
	return nil, errors.New("Default command has not been provided")
}

func prepareCommandDescriptor(cmdString string) (CommandDescriptor, error) {
	descriptor := CommandDescriptor{}
	if len(cmdString) == 0 {
		return descriptor, errors.New("Command must not be empty")
	}
	descriptor.CommandString = cmdString
	descriptor.subCommands = utils.Split(descriptor.CommandString, "|")
	return descriptor, nil
}

func buildExecCmds(d *CommandDescriptor) ([]*exec.Cmd, error) {
	procs := make([]*exec.Cmd, 0)
	for _, proc := range d.subCommands {
		if cmd, err := buildExecCmd(proc); err == nil {
			procs = append(procs, cmd)
		} else {
			return nil, err
		}
	}
	return procs, nil
}

func buildExecCmd(cmdString string) (*exec.Cmd, error) {
	if len(cmdString) == 0 {
		return nil, errors.New("Sub-command must not be empty")
	}
	parts := strings.Split(cmdString, " ")
	return exec.Command(parts[0], parts[1:]...), nil
}

func runSingleCommand(cmdObject *exec.Cmd, inData []byte) ([]byte, []byte, error) {
	var outData []byte
	var errData []byte
	
	inPipe, _ := cmdObject.StdinPipe()
	outPipe, _ := cmdObject.StdoutPipe()
	errPipe, _ := cmdObject.StderrPipe()

	cmdObject.Start()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		if inData != nil {
			inPipe.Write(inData)
			inPipe.Close()
		}
	}()

	go func() {
		defer wg.Done()
		outData, _ = ioutil.ReadAll(outPipe)
		errData, _ = ioutil.ReadAll(errPipe)
		cmdObject.Wait()
	}()

	wg.Wait()

	return outData, errData, nil
}

func (e *Executor) RunWithPipes(opts *CommandInvocation, ip *io.PipeReader, op *io.PipeWriter, ep *io.PipeWriter) (error) {
	return nil
}
