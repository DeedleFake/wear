wear
====

wear is a Go module for writing Wayland clients and compositors using the Elm architecture. It is _highly_ experimental and is being developed primarily for use in [kawa](https://github.com/DeedleFake/kawa).

Design Example
--------------

This partial example uses the planned design. In other words, it's how I would _like_ it to work, but is not necessarily realistic. It is not only incomplete but also likely to change a _lot_ during development.

```go
package main

import (
	"color"
	"log"

	"deedles.dev/ea"
	"deedles.dev/wear"
)

type model struct {
	// ...
}

func newModel() model {
	// ...
}

func (m model) Update(msg ea.Msg) (wear.Model, ea.Cmd) {
	switch msg := msg.(type) {
	case wear.FrameMsg:
		// ...
	case wear.KeyMsg:
		// ...
	case wear.PointerMsg:
		// ...
	case wear.SurfaceCreationMsg:
		// ...
	}
}

func (m model) Render(r *wear.Renderer) {
	r.Clear(color.Black)
	defer r.Commit()
	
	for _, s := range m.surfaces {
		if s.On(r.Output()) {
			r.DrawSurface(s, s.Bounds(), s.Bounds())
		}
	}
}

func main() {
	err := wear.Run(context.Background(), newModel())
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
```
