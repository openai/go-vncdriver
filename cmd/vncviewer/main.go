package main

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	"log"
	"os"
	"path"
	"sync"
	"time"

	"github.com/google/gxui"
	"github.com/google/gxui/drivers/gl"
	"github.com/google/gxui/samples/flags"
	"github.com/openai/go-vncdriver/gymvnc"
	"github.com/openai/go-vncdriver/vncclient"
	"github.com/pixiv/go-libjpeg/rgb"
)

func toImage(s *gymvnc.Screen) image.Image {
	img := rgb.NewImage(image.Rectangle{
		Max: image.Point{
			X: int(s.Width),
			Y: int(s.Height),
		},
	})
	for i, c := range s.Data {
		img.Pix[3*i] = uint8(c.R)
		img.Pix[3*i+1] = uint8(c.G)
		img.Pix[3*i+2] = uint8(c.B)
	}
	return img
}

func mapKey(in gxui.KeyboardKey) (out uint32) {
	// see keysymdef.h
	switch in {
	case gxui.KeyUnknown:
		return 0
	case gxui.KeySpace:
		return 0xff80
	case gxui.KeyApostrophe:
		return 0x0027
	case gxui.KeyComma:
		return 0x002c
	case gxui.KeyMinus:
		return 0x002d
	case gxui.KeyPeriod:
		return 0x002e
	case gxui.KeySlash:
		return 0x002f
	case gxui.Key0:
		return 0x0030
	case gxui.Key1:
		return 0x0031
	case gxui.Key2:
		return 0x0032
	case gxui.Key3:
		return 0x0033
	case gxui.Key4:
		return 0x0034
	case gxui.Key5:
		return 0x0035
	case gxui.Key6:
		return 0x0036
	case gxui.Key7:
		return 0x0037
	case gxui.Key8:
		return 0x0038
	case gxui.Key9:
		return 0x0039
	case gxui.KeySemicolon:
		return 0x003b
	case gxui.KeyEqual:
		return 0x003d
	case gxui.KeyA:
		return 0x0061
	case gxui.KeyB:
		return 0x0062
	case gxui.KeyC:
		return 0x0063
	case gxui.KeyD:
		return 0x0064
	case gxui.KeyE:
		return 0x0065
	case gxui.KeyF:
		return 0x0066
	case gxui.KeyG:
		return 0x0067
	case gxui.KeyH:
		return 0x0068
	case gxui.KeyI:
		return 0x0069
	case gxui.KeyJ:
		return 0x006a
	case gxui.KeyK:
		return 0x006b
	case gxui.KeyL:
		return 0x006c
	case gxui.KeyM:
		return 0x006d
	case gxui.KeyN:
		return 0x006e
	case gxui.KeyO:
		return 0x006f
	case gxui.KeyP:
		return 0x0070
	case gxui.KeyQ:
		return 0x0071
	case gxui.KeyR:
		return 0x0072
	case gxui.KeyS:
		return 0x0073
	case gxui.KeyT:
		return 0x0074
	case gxui.KeyU:
		return 0x0075
	case gxui.KeyV:
		return 0x0076
	case gxui.KeyW:
		return 0x0077
	case gxui.KeyX:
		return 0x0078
	case gxui.KeyY:
		return 0x0079
	case gxui.KeyZ:
		return 0x007a
	case gxui.KeyLeftBracket:
		return 0x005b
	case gxui.KeyBackslash:
		return 0x005c
	case gxui.KeyRightBracket:
		return 0x005d
	case gxui.KeyGraveAccent:
		return 0x0060
	case gxui.KeyWorld1:
	case gxui.KeyWorld2:
	case gxui.KeyEscape:
		return 0xff1b
	case gxui.KeyEnter:
		return 0xff8d
	case gxui.KeyTab:
		return 0xff89
	case gxui.KeyBackspace:
		return 0xff08
	case gxui.KeyInsert:
	case gxui.KeyDelete:
	case gxui.KeyRight:
		return 0xff53
	case gxui.KeyLeft:
		return 0xff51
	case gxui.KeyDown:
		return 0xff54
	case gxui.KeyUp:
		return 0xff52
	case gxui.KeyPageUp:
	case gxui.KeyPageDown:
	case gxui.KeyHome:
	case gxui.KeyEnd:
	case gxui.KeyCapsLock:
	case gxui.KeyScrollLock:
	case gxui.KeyNumLock:
	case gxui.KeyPrintScreen:
	case gxui.KeyPause:
	case gxui.KeyF1:
	case gxui.KeyF2:
	case gxui.KeyF3:
	case gxui.KeyF4:
	case gxui.KeyF5:
	case gxui.KeyF6:
	case gxui.KeyF7:
	case gxui.KeyF8:
	case gxui.KeyF9:
	case gxui.KeyF10:
	case gxui.KeyF11:
	case gxui.KeyF12:
	case gxui.KeyKp0:
	case gxui.KeyKp1:
	case gxui.KeyKp2:
	case gxui.KeyKp3:
	case gxui.KeyKp4:
	case gxui.KeyKp5:
	case gxui.KeyKp6:
	case gxui.KeyKp7:
	case gxui.KeyKp8:
	case gxui.KeyKp9:
	case gxui.KeyKpDecimal:
	case gxui.KeyKpDivide:
	case gxui.KeyKpMultiply:
	case gxui.KeyKpSubtract:
	case gxui.KeyKpAdd:
	case gxui.KeyKpEnter:
	case gxui.KeyKpEqual:
	case gxui.KeyLeftShift:
	case gxui.KeyLeftControl:
	case gxui.KeyLeftAlt:
	case gxui.KeyLeftSuper:
	case gxui.KeyRightShift:
	case gxui.KeyRightControl:
	case gxui.KeyRightAlt:
	case gxui.KeyRightSuper:
	case gxui.KeyMenu:
	case gxui.KeyLast:
	}
	return 0
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %v <vnc address>", path.Base(os.Args[0]))
	}

	fmt.Println("creating session")
	s := gymvnc.NewVNCSession("", gymvnc.VNCSessionConfig{
		Address:  os.Args[1],
		Password: "openai",
		Encoding: "tight",
	})

	gl.StartDriver(func(driver gxui.Driver) {
		theme := flags.CreateTheme(driver)
		img := theme.CreateImage()
		window := theme.CreateWindow(1000, 1000, "go-vncdriver vnc viewer")
		window.AddChild(img)
		window.OnClose(driver.Terminate)
		var events struct {
			sync.Mutex
			slice []gymvnc.VNCEvent
		}
		onEvent := func(e gymvnc.VNCEvent) {
			events.Lock()
			defer events.Unlock()
			events.slice = append(events.slice, e)
		}
		window.OnKeyDown(func(e gxui.KeyboardEvent) {
			onEvent(gymvnc.KeyEvent{
				Keysym: mapKey(e.Key),
				Down:   true,
			})
		})
		window.OnKeyUp(func(e gxui.KeyboardEvent) {
			onEvent(gymvnc.KeyEvent{
				Keysym: mapKey(e.Key),
				Down:   false,
			})
		})
		handleMouseEvent := func(e gxui.MouseEvent) {
			var btnMask vncclient.ButtonMask
			if e.State.IsDown(gxui.MouseButtonLeft) {
				btnMask |= vncclient.ButtonLeft
			}
			if e.State.IsDown(gxui.MouseButtonMiddle) {
				btnMask |= vncclient.ButtonMiddle
			}
			if e.State.IsDown(gxui.MouseButtonRight) {
				btnMask |= vncclient.ButtonRight
			}
			onEvent(gymvnc.PointerEvent{
				X:    uint16(e.Point.X),
				Y:    uint16(e.Point.Y),
				Mask: btnMask,
			})
		}
		img.OnMouseDown(handleMouseEvent)
		img.OnMouseUp(handleMouseEvent)
		img.OnMouseMove(handleMouseEvent)

		go func() {
			for {
				fmt.Println("tick")
				time.Sleep(time.Second / 60)
				events.Lock()
				screen, updates, err := s.Step(events.slice)
				events.slice = events.slice[:0]
				events.Unlock()
				if err != nil {
					log.Fatal(err)
				}
				if len(updates) == 0 {
					continue
				}
				source := toImage(screen)
				rgba := image.NewRGBA(source.Bounds())
				draw.Draw(rgba, source.Bounds(), source, image.ZP, draw.Src)
				driver.Call(func() {
					img.SetTexture(driver.CreateTexture(rgba, 1))
				})
			}
		}()
	})
}
