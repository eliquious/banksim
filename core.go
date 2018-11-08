package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MessageType is the message type
type MessageType string

// Common message types
const (
	MessageTypeStart MessageType = "START"
	MessageTypeStop  MessageType = "STOP"
)

// Message is the value type exchanged between data processors
type Message struct {
	Timestamp time.Time
	Type      MessageType
	Forward   bool
	Value     interface{}
}

// State manages the processor state
type State string

// States
const (
	StateRunning State = "RUNNING"
	StateWaiting       = "WAITING"
	StateKilled        = "KILLED"
)

// HandleFunc processes messages
type HandleFunc func(context.Context, Process, Message)

// Handler is a simple interface for anything with a HandleFunc
type Handler interface {
	Handle(ctx context.Context, proc Process, msg Message)
}

// Process is the base data processing element.
type Process interface {
	Name() string
	SetState(State)
	Start(*sync.WaitGroup)
	Send(Message)
	Inbox() <-chan Message
	Children() ProcessList
}

// ProcessList is a list type for Process
type ProcessList []Process

// Dispatch dispatches a message to a list of processes
func (p ProcessList) Dispatch(m Message) {
	for i := 0; i < len(p); i++ {
		p[i].Send(m)
	}
}

// NewDefaultProcess creates a new DefaultProcess with the given properties.
func NewDefaultProcess(ctx context.Context, name string, h Handler, ps ProcessList) Process {
	p := &DefaultProcess{
		ctx:      ctx,
		name:     name,
		handler:  h,
		state:    StateWaiting,
		stateCh:  make(chan State, 2),
		inbox:    make(chan Message, 2),
		children: ps,
	}
	// go p.Start()
	return p
}

// DefaultProcess is the default process
type DefaultProcess struct {
	handler  Handler
	ctx      context.Context
	state    State
	inbox    chan Message
	stateCh  chan State
	children ProcessList
	name     string
}

// Start runs the process
func (p *DefaultProcess) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	for _, c := range p.children {
		c.Start(wg)
	}

	go func() {
		defer wg.Done()
		for p.state != StateKilled {

			select {
			case <-p.ctx.Done():
				p.SetState(StateKilled)
			case s := <-p.stateCh:
				p.state = s
			case msg := <-p.inbox:
				switch msg.Type {
				case MessageTypeStart:
					p.state = StateRunning
					p.Children().Dispatch(msg)
				case MessageTypeStop:
					p.state = StateKilled
					p.Children().Dispatch(msg)
				}

				// Process message
				if p.state == StateRunning {
					p.handler.Handle(p.ctx, p, msg)
				}

				// Forward message
				if msg.Forward && msg.Type != MessageTypeStart {
					p.Children().Dispatch(msg)
				}
			}
		}
		p.Children().Dispatch(Message{Timestamp: time.Now(), Type: MessageTypeStop, Forward: true})
	}()
}

// SetState returns the process state
func (p *DefaultProcess) SetState(s State) {
	p.stateCh <- s
}

// Name returns the process name
func (p *DefaultProcess) Name() string {
	return p.name
}

// Send enqueues incoming messages and updates account information
func (p *DefaultProcess) Send(msg Message) {
	p.inbox <- msg
}

// Inbox returns the process queue
func (p *DefaultProcess) Inbox() <-chan Message {
	return p.inbox
}

// Children returns the process children
func (p *DefaultProcess) Children() ProcessList {
	return p.children
}

// Engine manages the pipeline
type Engine interface {
	Start(*sync.WaitGroup)
	Stop()
}

// Service allows for global processors.
type Service interface {
	Name() string
	Process(Message)
}

// ServiceList is a value type for a list of services
type ServiceList []Service

// SendTo sends a message to a particular service
func (s ServiceList) SendTo(name string, m Message) {
	for i := 0; i < len(s); i++ {
		if s[i].Name() == name {
			s[i].Process(m)
		}
	}
}

type contextKey string

var serviceKey = contextKey("svc")
var nameKey = contextKey("name")
var waitGroupKey = contextKey("waitgroup")

// SendTo allows for sending messages to services
func SendTo(ctx context.Context, svc string, msg Message) {
	if v := ctx.Value(serviceKey); v != nil {
		if svcs, ok := v.(ServiceList); ok {
			svcs.SendTo(svc, msg)
		}
		return
	}
}

// WithService adds a service to a context.Context.
func WithService(ctx context.Context, svc Service) context.Context {
	if v := ctx.Value(serviceKey); v != nil {
		if svcs, ok := v.(ServiceList); ok {
			svcs = append(svcs, svc)
			return context.WithValue(ctx, serviceKey, svcs)
		}
		return ctx
	}
	return context.WithValue(ctx, serviceKey, ServiceList{svc})
}

// WithName adds a name to a context.Context.
func WithName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, nameKey, name)
}

// Name gets a name from a context.Context.
func Name(ctx context.Context) string {
	if v := ctx.Value(nameKey); v != nil {
		if name, ok := v.(string); ok {
			return name
		}
		return ""
	}
	return ""
}

// WithWaitGroup adds a sync.WaitGroup to a context.Context.
func WithWaitGroup(ctx context.Context, wg *sync.WaitGroup) context.Context {
	return context.WithValue(ctx, waitGroupKey, wg)
}

// Done decrements a sync.WaitGroup from a context.Context.
func Done(ctx context.Context) {
	if v := ctx.Value(waitGroupKey); v != nil {
		if wg, ok := v.(*sync.WaitGroup); ok {
			fmt.Println("Done 1")
			wg.Done()
		}
	}
}

// Add increments a sync.WaitGroup from a context.Context.
func Add(ctx context.Context) {
	if v := ctx.Value(waitGroupKey); v != nil {
		if wg, ok := v.(*sync.WaitGroup); ok {
			fmt.Println("Add 1")
			wg.Add(1)
		}
	}
}

// NewEngine creates a new engine
func NewEngine(ctx context.Context, cancel context.CancelFunc, ps ProcessList) Engine {
	return &engine{ctx, cancel, ps}
}

type engine struct {
	ctx      context.Context
	cancel   context.CancelFunc
	children ProcessList
}

func (e *engine) Start(wg *sync.WaitGroup) {
	for _, p := range e.children {
		p.Start(wg)
	}
	e.children.Dispatch(Message{Timestamp: time.Now(), Type: MessageTypeStart, Forward: true})
}

func (e *engine) Stop() {
	e.children.Dispatch(Message{Timestamp: time.Now(), Type: MessageTypeStop, Forward: true})
	e.cancel()
}
