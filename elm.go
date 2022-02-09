package elmgo

import (
	"context"
)

// Elmgo         __   __
//        .'  '.'  `.
//     _.-|  o | o  |-._
//   .~   `.__.'.__.'^  ~.
// .~     ^  /   \  ^     ~.
// \-._^   ^|     |    ^_.-/
// `\  `-._  \___/ ^_.-' /'
//   `\_   `--...--'   /'
//      `-.._______..-'      /\  /\
//         __/   \__         | |/ /_
//       .'^   ^    `.      .'   `__\
//     .'    ^     ^  `.__.'^ .\ \
//    .' ^ .    ^   .    ^  .'  \/
//   /    /        ^ \'.__.'
//  |  ^ /|   ^      |
//   \   \|^      ^  |
//    `\^ |        ^ |
//      `~|    ^     |
//        |  ^     ^ |
//        \^         /
//         `.    ^ .'
//    jgs   : ^    ;
//  .-~~~~~~   |  ^ ~~~~~~-.
// /   ^     ^ |    ^       \
// \^     ^   / \  ^     ^  /

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . Dispatcher
type Dispatcher[Msg any] interface {
	Dispatch(Msg)
}

type ChannelDispatcher[Msg any] struct {
	dispatchCh chan<- Msg
}

func (d *ChannelDispatcher[Msg]) Dispatch(msg Msg) {
	d.dispatchCh <- msg
}

//counterfeiter:generate . Renderer
type Renderer[A any] interface {
	Render(A)
}

type Cmd interface {
	Run() error
}

//counterfeiter:generate . Elmable
type Elmable[Model any, Msg any, Renderable any] interface {
	Init() Model
	Update(Msg, Model) (Model, Cmd)
	View(Model, Dispatcher[Msg]) Renderable
}

type App[Model any, Msg any, Renderable any] struct {
	elmable  Elmable[Model, Msg, Renderable]
	renderer Renderer[Renderable]
}

// Wow these generics are gross
func NewApp[Model any, Msg any, Renderable any](elmable Elmable[Model, Msg, Renderable], renderer Renderer[Renderable]) *App[Model, Msg, Renderable] {
	return &App[Model, Msg, Renderable]{
		elmable:  elmable,
		renderer: renderer,
	}
}

func (a *App[Model, Msg, Renderable]) Run(ctx context.Context) chan struct{} {
	doneCh := make(chan struct{})
	dispatchCh := make(chan Msg)
	// Get initial model and render the view
	model := a.elmable.Init()
	a.renderer.Render(a.elmable.View(model, &ChannelDispatcher[Msg]{
		dispatchCh: dispatchCh,
	}))

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(doneCh)
				return
			case msg := <-dispatchCh:
				// Note: this is mutating the model in place
				model, _ = a.elmable.Update(msg, model)
				a.renderer.Render(a.elmable.View(model, &ChannelDispatcher[Msg]{
					dispatchCh: dispatchCh,
				}))
			}
		}
	}()

	return doneCh
}
