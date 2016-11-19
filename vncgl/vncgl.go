package vncgl

import (
	"image"
	_ "image/png"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/juju/errors"
	"github.com/op/go-logging"
	"github.com/openai/go-vncdriver/vncclient"
)

var (
	log       = logging.MustGetLogger("vncgl")
	renderers int
	glfwState sync.Mutex
)

func colorsToImage(x, y, width, height uint16, colors []vncclient.Color) *image.RGBA {
	imgRect := image.Rect(int(x), int(y), int(x+width), int(y+height))
	rgba := image.NewRGBA(imgRect)

	for i, color := range colors {
		rgba.Pix[4*i] = color.R
		rgba.Pix[4*i+1] = color.G
		rgba.Pix[4*i+2] = color.B
		rgba.Pix[4*i+3] = 1.0
	}

	return rgba
}

// Only interact with this from the main thread
type VNCGL struct {
	rootTexture               uint32
	window                    *glfw.Window
	windowWidth, windowHeight uint16
	closed                    bool
}

func (v *VNCGL) Init(width, height uint16, name string, screen []vncclient.Color) error {
	glfwState.Lock()
	defer glfwState.Unlock()

	if renderers == 0 {
		if err := glfw.Init(); err != nil {
			log.Fatalf("failed to initialize glfw: %v", err)
		}

		glfw.WindowHint(glfw.Resizable, glfw.False)
		glfw.WindowHint(glfw.ContextVersionMajor, 2)
		glfw.WindowHint(glfw.ContextVersionMinor, 1)
	}
	renderers++

	window, err := glfw.CreateWindow(int(width), int(height), name, nil, nil)
	if err != nil {
		return errors.Annotate(err, "couldn't create window")
	}
	window.MakeContextCurrent()
	glfw.SwapInterval(0)

	if err := gl.Init(); err != nil {
		window.Destroy()
		return errors.Annotate(err, "could not init opengl")
	}

	v.window = window
	v.windowWidth = width
	v.windowHeight = height

	if screen != nil {
		image := colorsToImage(0, 0, width, height, screen)
		v.applyImage(image)
	}
	return nil
}

func (g *VNCGL) Close() error {
	if g.closed {
		return nil
	}
	g.closed = true

	g.window.Destroy()
	if g.rootTexture != 0 {
		gl.DeleteTextures(1, &g.rootTexture)
	}

	glfwState.Lock()
	defer glfwState.Unlock()
	renderers--
	if renderers == 0 {
		glfw.Terminate()
	}
	return nil
}

func (g *VNCGL) Apply(updates []*vncclient.FramebufferUpdateMessage) {
	start := time.Now().UnixNano()
	count := 0

	for _, update := range updates {
		for _, rect := range update.Rectangles {
			count += 1

			var rgba *image.RGBA
			switch enc := rect.Enc.(type) {
			case *vncclient.RawEncoding:
				rgba = colorsToImage(rect.X, rect.Y, rect.Width, rect.Height, enc.Colors)
			case *vncclient.ZRLEEncoding:
				rgba = colorsToImage(rect.X, rect.Y, rect.Width, rect.Height, enc.Colors)
			case *vncclient.TightEncoding:
				rgba = colorsToImage(rect.X, rect.Y, rect.Width, rect.Height, enc.Colors)
			default:
				panic(errors.Errorf("BUG: unrecognized encoding: %+v", enc))
			}

			g.applyImage(rgba)
		}
	}
	log.Debugf("Completed Apply: count=%d time=%v", count, time.Duration(time.Now().UnixNano()-start))
}

func (g *VNCGL) applyImage(img *image.RGBA) {
	// TODO: make sure texture can't legitimately be 0
	if g.rootTexture == 0 {
		g.rootTexture = newTexture(img)
	} else {
		gl.BindTexture(gl.TEXTURE_2D, g.rootTexture)
		gl.TexSubImage2D(
			gl.TEXTURE_2D,
			0,
			int32(img.Rect.Min.X),
			int32(img.Rect.Min.Y),
			int32(img.Rect.Size().X),
			int32(img.Rect.Size().Y),
			gl.RGBA,
			gl.UNSIGNED_BYTE,
			gl.Ptr(img.Pix))
	}
}

func newTexture(rgba *image.RGBA) uint32 {
	var texture uint32
	gl.Enable(gl.TEXTURE_2D)
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	return texture
}

func (g *VNCGL) Render() {
	if g.closed {
		return
	}

	glfw.PollEvents()
	if g.window.ShouldClose() {
		g.Close()
		return
	}

	g.window.MakeContextCurrent()
	g.drawScene()
	gl.Finish()

	start := time.Now().UnixNano()
	g.window.SwapBuffers()
	// Usually takes like 100microseconds, but sometimes takes up
	// to 10ms.
	log.Debugf("SwapBuffers: time=%v", time.Duration(time.Now().UnixNano()-start))
}

func (g *VNCGL) drawScene() {
	gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()
	gl.Ortho(0, float64(g.windowWidth), 0, float64(g.windowHeight), -1, 1)
	gl.Enable(gl.TEXTURE_2D)

	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.BindTexture(gl.TEXTURE_2D, g.rootTexture)
	gl.Begin(gl.QUADS)
	gl.TexCoord2d(0.0, 0.0)
	gl.Vertex3f(0, float32(g.windowHeight), 0.0)

	gl.TexCoord2d(1.0, 0.0)
	gl.Vertex3f(float32(g.windowWidth), float32(g.windowHeight), 0.0)

	gl.TexCoord2d(1.0, 1.0)
	gl.Vertex3f(float32(g.windowWidth), 0.0, 0.0)

	gl.TexCoord2d(0.0, 1.0)
	gl.Vertex3f(0, 0.0, 0.0)

	gl.End()
	gl.PopMatrix()
	// gl.Flush()
}

func init() {
	// GLFW event handling must run on the main OS thread
	runtime.LockOSThread()
}

func connect(imgs chan *image.RGBA) (*vncclient.ClientConn, error) {
	tcp, err := net.Dial("tcp", "172.16.163.128:5900")
	if err != nil {
		return nil, errors.Annotate(err, "could not connect to server")
	}

	errCh := make(chan error)
	serverMessageCh := make(chan vncclient.ServerMessage)
	conn, err, _ := vncclient.Client(tcp, &vncclient.ClientConfig{
		Auth: []vncclient.ClientAuth{
			&vncclient.PasswordAuth{
				Password: "openai",
			},
		},
		ServerMessageCh: serverMessageCh,
		ErrorCh:         errCh,
	})
	if err != nil {
		return nil, errors.Annotate(err, "could not establish VNC connection to server")
	}

	go func() {
		err := <-errCh
		panic(errors.ErrorStack(err))
	}()

	updates := make(chan *vncclient.FramebufferUpdateMessage)

	// Message thread
	go func() {
		// Ensure standard format
		conn.SetPixelFormat(&vncclient.PixelFormat{
			BPP:        32,
			Depth:      24,
			BigEndian:  false,
			TrueColor:  true,
			RedMax:     255,
			GreenMax:   255,
			BlueMax:    255,
			RedShift:   0,
			GreenShift: 8,
			BlueShift:  16,
		})

		conn.SetEncodings([]vncclient.Encoding{
			&vncclient.ZRLEEncoding{},
			&vncclient.TightEncoding{},
			&vncclient.RawEncoding{},
		})

		conn.FramebufferUpdateRequest(true, 0, 0, conn.FramebufferWidth, conn.FramebufferHeight)

		for {
			select {
			case msg := <-serverMessageCh:
				switch msg := msg.(type) {
				case *vncclient.FramebufferUpdateMessage:
					updates <- msg
					// Keep re-requesting!
					conn.FramebufferUpdateRequest(true, 0, 0, conn.FramebufferWidth, conn.FramebufferHeight)
				}
			}
		}
	}()

	vncgl := &VNCGL{}
	if err := vncgl.Init(conn.FramebufferWidth, conn.FramebufferHeight, conn.DesktopName, nil); err != nil {
		return nil, err
	}

	// Apply those updates
	var frames int
	target := time.Now().UnixNano()
	printTarget := time.Now().UnixNano()
	for {
		target += int64(1000 * 1000 * 1000 / 60)
		frames += 1

		if time.Now().UnixNano() > printTarget {
			printTarget = time.Now().UnixNano() + 1000*1000*1000
			log.Infof("effective framerate: %v", frames)
			frames = 0
		}

		var done bool
		for !done {
			select {
			case msg := <-updates:
				msgs := []*vncclient.FramebufferUpdateMessage{msg}
				vncgl.Apply(msgs)
			default:
				done = true
			}
		}

		vncgl.Render()

		delta := time.Duration(target-time.Now().UnixNano()) * time.Nanosecond
		if delta > 0 {
			log.Infof("sleeping: %v", delta)
			time.Sleep(delta)
		} else {
			log.Infof("full behind by: %v", delta)
		}
	}
}
