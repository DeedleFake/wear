package wear

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log"

	"deedles.dev/ea"
	wl "deedles.dev/wl/client"
)

type Msg = ea.Msg

type Cmd = ea.Cmd

type Model interface {
	Update(Msg) (Model, Cmd)
	Render(r Renderer)
}

type Renderer interface {
	FillRect(image.Rectangle, color.Color)
}

type model struct {
	client     *wl.Client
	compositor *wl.Compositor
	shm        *wl.Shm

	m Model
}

func (m model) Update(msg Msg) (model, Cmd) {
	switch msg := msg.(type) {
	case funcMsg:
		return msg(m)

	default:
		next, cmd := m.m.Update(msg)
		m.m = next
		return m, cmd
	}
}

func Run(ctx context.Context, m Model) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	client, err := wl.Dial()
	if err != nil {
		return fmt.Errorf("dial display: %w", err)
	}
	defer client.Close()
	client.Display().Listener = &displayListener{
		client: client,
		cancel: cancel,
	}

	registry := client.Display().GetRegistry()
	g := globalsInitializer{
		client:   client,
		registry: registry,
	}
	registry.Listener = &g

	err = client.RoundTrip()
	if err != nil {
		return fmt.Errorf("round trip for globals: %w", err)
	}

	loop := ea.New(model{
		client:     client,
		compositor: g.compositor,
		shm:        g.shm,

		m: m,
	})

	registry.Listener = &registryListener{
		loop: loop,
	}

	for _, global := range g.extra {
		loop.Enqueue(global)
	}

	for {
		select {
		case <-ctx.Done():
			return nil

		case update, ok := <-loop.Updates():
			if !ok {
				return nil
			}

			update()

		case ev, ok := <-client.Events():
			if !ok {
				return nil
			}

			loop.Enqueue(funcMsg(func(m model) (model, Cmd) {
				err := ev()
				if err != nil {
					log.Printf("flush events: %v", err)
				}
				return m, nil
			}))
		}
	}
}

type funcMsg func(model) (model, Cmd)

type GlobalMsg struct {
	Name      uint32
	Interface string
	Version   uint32
}

type GlobalRemoveMsg struct {
	Name uint32
}

type globalsInitializer struct {
	client   *wl.Client
	registry *wl.Registry

	compositor *wl.Compositor
	shm        *wl.Shm

	extra []GlobalMsg
}

func (g *globalsInitializer) Global(name uint32, inter string, version uint32) {
	switch inter {
	case wl.CompositorInterface:
		g.compositor = wl.BindCompositor(g.client, g.registry, name, version)
	case wl.ShmInterface:
		g.shm = wl.BindShm(g.client, g.registry, name, version)
	default:
		g.extra = append(g.extra, GlobalMsg{Name: name, Interface: inter, Version: version})
	}
}

func (g *globalsInitializer) GlobalRemove(name uint32) {}

type displayListener struct {
	client *wl.Client
	cancel context.CancelFunc
}

func (lis *displayListener) Error(id, code uint32, msg string) {
	log.Printf("display error: id: %v, code: %v, msg: %q", id, code, msg)
	lis.cancel()
}

func (lis *displayListener) DeleteId(id uint32) {
	lis.client.Delete(id)
}

type registryListener struct {
	loop *ea.Loop[model]
}

func (lis *registryListener) Global(name uint32, inter string, version uint32) {
	lis.loop.Enqueue(GlobalMsg{
		Name:      name,
		Interface: inter,
		Version:   version,
	})
}

func (lis *registryListener) GlobalRemove(name uint32) {
	lis.loop.Enqueue(GlobalRemoveMsg{
		Name: name,
	})
}
