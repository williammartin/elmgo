package elmgo_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/williammartin/elmgo"
)

type TextRendererSpy struct {
	renderLastCalledWith string
}

func (s *TextRendererSpy) Render(text string) {
	s.renderLastCalledWith = text
}

func (s *TextRendererSpy) LastCalledWith() string {
	return s.renderLastCalledWith
}

// Model
type CounterModel struct {
	Count int
}

// Messages
type CounterMsg interface {
	isCounterMsg()
}

//go-sumtype:decl Msg
type Increment struct{}

func (*Increment) isCounterMsg() {}

type Decrement struct{}

func (*Decrement) isCounterMsg() {}

// CounterApp
type CounterApp struct {
	dispatchHandle elmgo.Dispatcher[CounterMsg]
}

func (a *CounterApp) Init() CounterModel {
	return CounterModel{
		Count: 0,
	}
}

func (a *CounterApp) Update(msg CounterMsg, model CounterModel) (CounterModel, elmgo.Cmd) {
	switch CounterMsg(msg).(type) {
	case *Increment:
		return CounterModel{Count: model.Count + 1}, nil
	case *Decrement:
		return CounterModel{Count: model.Count - 1}, nil
	default:
		panic(fmt.Sprintf("unexpected msg type: %v", msg))
	}
}

func (a *CounterApp) View(model CounterModel, dispatcher elmgo.Dispatcher[CounterMsg]) string {
	a.dispatchHandle = dispatcher
	return fmt.Sprintf("Count is: %d", model.Count)
}

func TestAllTheThings(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	counterApp := &CounterApp{}
	renderer := &TextRendererSpy{}
	elmer := elmgo.NewApp[CounterModel, CounterMsg, string](counterApp, renderer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	elmer.Run(ctx)

	is.Equal(renderer.LastCalledWith(), "Count is: 0")

	counterApp.dispatchHandle.Dispatch(&Increment{})

	eventually(renderer.LastCalledWith).returns("Count is: 1")

	counterApp.dispatchHandle.Dispatch(&Decrement{})

	eventually(renderer.LastCalledWith).returns("Count is: 0")
}

// TODO: Turn this into an expressive matcher for elmgo e.g.
// eventuallyRenders()
type AsyncAssertion struct {
	fn func() string
}

func eventually(fn func() string) *AsyncAssertion {
	return &AsyncAssertion{
		fn: fn,
	}
}

func (m *AsyncAssertion) returns(text string) {
	for {
		<-time.After(time.Millisecond * 10)
		if m.fn() == text {
			return
		}
	}
}
