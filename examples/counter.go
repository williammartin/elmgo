package main

import (
	"context"
	"fmt"

	"github.com/jroimartin/gocui"
	"github.com/williammartin/elmgo"
)

// Model
type CounterModel struct {
	Count int32
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

func (a *CounterApp) View(model CounterModel, dispatcher elmgo.Dispatcher[CounterMsg]) *GoCUIView {
	return &GoCUIView{
		Title:    "Counter",
		Contents: fmt.Sprintf("Enter to Increment - Backspace to Decrement\n\nCount is: %d", model.Count),
		Keybindings: []Keybinding{{
			Key: gocui.KeyEnter,
			Fn: func() {
				dispatcher.Dispatch(&Increment{})
			},
		}, {
			Key: gocui.KeyBackspace2,
			Fn: func() {
				dispatcher.Dispatch(&Decrement{})
			},
		}},
	}
}

func main() {
	g, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		panic(err)
	}

	counterApp := &CounterApp{}
	renderer := NewGoCUIRenderer()
	elmer := elmgo.NewApp[CounterModel, CounterMsg, *GoCUIView](counterApp, renderer)

	ctx, cancel := context.WithCancel(context.Background())
	renderer.Run(ctx, g)
	elmerDone := elmer.Run(ctx)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		cancel()
		return gocui.ErrQuit
	}); err != nil {
		panic(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		panic(err)
	}
	defer g.Close()
	<-elmerDone
}

// Silly GoCUI Types and Behaviour
// There's a lot of work to do to figure out what the rendering seam should be, because this is super leaky.
type Keybinding struct {
	Key gocui.Key
	Fn  func()
}
type GoCUIView struct {
	Title       string
	Contents    string
	Keybindings []Keybinding
}

type GoCUIRenderer struct {
	viewCh chan *GoCUIView
}

func NewGoCUIRenderer() *GoCUIRenderer {
	return &GoCUIRenderer{
		viewCh: make(chan *GoCUIView),
	}
}

func (r *GoCUIRenderer) Run(ctx context.Context, g *gocui.Gui) {
	maxX, maxY := g.Size()

	// Initially no view...maybe use an option?
	var v *GoCUIView
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case view := <-r.viewCh:
				g.Update(func(g *gocui.Gui) error {
					if v != nil {
						g.DeleteKeybindings("root")
						g.DeleteView("root")
					}
					v = view
					gV, err := g.SetView("root", 0, 0, maxX, maxY)
					if err != nil && err != gocui.ErrUnknownView {
						return err
					}
					gV.Title = v.Title

					for _, kb := range v.Keybindings {
						f := kb.Fn
						if err := g.SetKeybinding("root", kb.Key, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
							f()
							return nil
						}); err != nil {
							return err
						}
					}

					if _, err := g.SetCurrentView("root"); err != nil {
						return err
					}

					fmt.Fprintln(gV, v.Contents)

					return nil
				})
			}
		}
	}()
}

func (r *GoCUIRenderer) Render(view *GoCUIView) {
	r.viewCh <- view
}
