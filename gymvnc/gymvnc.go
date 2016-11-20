package gymvnc

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/op/go-logging"
	"github.com/openai/go-vncdriver/vncclient"
)

var (
	log = logging.MustGetLogger("gymvnc")
	id  = 0
)

type sessionMgr struct {
	Ready      sync.WaitGroup
	Done       chan bool
	Error      chan error
	Terminated sync.WaitGroup
}

func NewSessionMgr() *sessionMgr {
	return &sessionMgr{
		Done:  make(chan bool),
		Error: make(chan error, 1),
	}
}

func (s *sessionMgr) Close() error {
	close(s.Done)
	return nil
}

type Region struct {
	X, Y, Width, Height uint16
}

type VNCSessionConfig struct {
	Address  string
	Password string
	Encoding string

	QualityLevel     int // 0-9, 9 being top quality. Not orthogonal to FineQualityLevel/SubsampleLevel, see https://github.com/TurboVNC/turbovnc/blob/master/unix/Xvnc/programs/Xserver/hw/vnc/rfbserver.c#L1103-L1112
	CompressLevel    int // 0-9, 9 being highest compression
	FineQualityLevel int // 0-100, 100 being top quality
	SubsampleLevel   int // 0-3, 3 being grayscale; 0 being full color

	StartTimeout time.Duration
	Subscription []Region
}

type VNCSession struct {
	id    int
	label string

	mgr  *sessionMgr
	conn *vncclient.ClientConn

	frontScreen        *Screen
	backScreen         *Screen
	backUpdated        bool
	deferredUpdates    []*vncclient.FramebufferUpdateMessage
	deferredUpdatesMax int
	pauseUpdates       bool
	updated            *sync.Cond

	renderer       Renderer
	rendererActive bool

	name   string
	config VNCSessionConfig

	lock   sync.Mutex
	err    error
	closed bool
}

func NewVNCSession(name string, c VNCSessionConfig) *VNCSession {
	if c.QualityLevel == -1 {
	} else if c.QualityLevel < 0 {
		log.Warningf("[%s] Quality level %d requested, but valid values are betweeen 0 (worst) and 9 (best). Using 0 intead.", c.Address, c.QualityLevel)
		c.QualityLevel = 0
	} else if c.QualityLevel > 9 {
		log.Warningf("[%s] Quality level %d requested, but valid values are betweeen 0 (worst) and 9 (best). Using 9 instead.", c.Address, c.QualityLevel)
		c.QualityLevel = 9
	}

	if c.CompressLevel == -1 {
	} else if c.CompressLevel < 0 {
		log.Warningf("[%s] Compress level %d requested, but valid values are betweeen 0 (least compression) and 9 (most compression). Using 0 intead.", c.Address, c.CompressLevel)
		c.CompressLevel = 0
	} else if c.CompressLevel > 9 {
		log.Warningf("[%s] Compress level %d requested, but valid values are betweeen 0 (least compression) and 9 (most compression). Using 9 instead.", c.Address, c.CompressLevel)
		c.CompressLevel = 9
	}

	if c.FineQualityLevel == -1 {
	} else if c.FineQualityLevel < 0 {
		log.Warningf("[%s] Fine quality level %d requested, but valid values are betweeen 1 (lowest quality) and 100 (highest quality). Using 0 instead.", c.Address, c.FineQualityLevel)
		c.FineQualityLevel = 0
	} else if c.FineQualityLevel > 100 {
		log.Warningf("[%s] Fine quality level %d requested, but valid values are betweeen 0 (lowest quality) and 100 (highest quality). Using 100 instead.", c.Address, c.FineQualityLevel)
		c.FineQualityLevel = 100
	}

	if c.SubsampleLevel == -1 {
	} else if c.SubsampleLevel < 0 {
		log.Warningf("[%s] Subsample level %d requested, but valid values are betweeen 0 (full color) and 3 (grayscale). Using 0 instead.", c.Address, c.SubsampleLevel)
		c.SubsampleLevel = 0
	} else if c.SubsampleLevel > 3 {
		log.Warningf("[%s] Subsample level %d requested, but valid values are betweeen 0 (full color) and 3 (grayscale). Using 3 instead.", c.Address, c.SubsampleLevel)
		c.SubsampleLevel = 3
	}

	if c.Encoding == "" {
		c.Encoding = "tight"
	}

	lock := &sync.Mutex{}
	session := &VNCSession{
		name:               name,
		mgr:                NewSessionMgr(),
		backUpdated:        true,
		deferredUpdatesMax: 60,
		updated:            sync.NewCond(lock),
		config:             c,

		id:    id,
		label: fmt.Sprintf("%d:%s:%s", id, name, c.Address),
	}
	id++
	session.start()
	return session
}

func (c *VNCSession) start() {
	updates := make(chan *vncclient.FramebufferUpdateMessage, 4096)

	// Called from main thread. Will call Done once this session
	// is fully setup.
	c.mgr.Ready.Add(1)

	// Maintains the connection to the remote
	go func() {
		err := c.connect(updates)
		if err != nil {
			// Try reporting the error
			select {
			case c.mgr.Error <- err:
			default:
			}
		}
	}()

	go func() {
		if err := <-c.mgr.Error; err != nil {
			c.lock.Lock()
			c.err = err
			c.lock.Unlock()
		}
	}()
}

func (c *VNCSession) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	c.mgr.Close()

	if c.conn != nil {
		log.Debugf("[%s] closing connection to VNC server", c.label)
		c.conn.Close()
	}
	if c.rendererActive {
		// This can *only* be called from the main thread, so
		// we can't auto-clean it up on error.
		c.renderer.Close()
	}
	return nil
}

func (c *VNCSession) Step(events []VNCEvent) (*Screen, []*vncclient.FramebufferUpdateMessage, error) {
	c.lock.Lock()
	conn := c.conn
	err := c.err
	c.lock.Unlock()

	if err != nil {
		// this VNCSession is broken
		return nil, nil, err
	} else if conn == nil {
		// not yet connected
		return nil, nil, nil
	}

	for _, event := range events {
		err := event.Execute(conn)
		if err != nil {
			return nil, nil, errors.Annotatef(err, "could not step %s", c.config.Address)
		}
	}

	screen, updates := c.Flip()
	return screen, updates, nil
}

func (c *VNCSession) SetRenderer(renderer Renderer) error {
	if c.rendererActive {
		return errors.New("cannot change renderer while active")
	}
	c.renderer = renderer
	return nil
}

// Must call from main thread
func (c *VNCSession) Render(close bool) error {
	if c.renderer == nil {
		return errors.Errorf("[%s] VNCSession has no renderer. This likely means your go_vncdriver was installed without the OpenGL viewer. See https://github.com/openai/tree/master/go-vncdriver for instructions on how to install with the OpenGL viewer.", c.label)
	}

	c.lock.Lock()
	if c.closed {
		c.lock.Unlock()
		return errors.Errorf("[%s] VNCSession is already closed; can't render", c.label)
	}
	conn := c.conn
	c.lock.Unlock()

	if conn == nil {
		// Not connected yet; nothing to render!
		return nil
	}

	if !c.rendererActive {
		if err := c.renderer.Init(c.conn.FramebufferWidth, c.conn.FramebufferHeight, "go-vncdriver: "+c.conn.DesktopName, c.frontScreen.Data); err != nil {
			return errors.Annotate(err, "could not render")
		}
		c.rendererActive = true
	}

	if close {
		c.renderer.Close()
	} else {
		c.renderer.Render()
	}
	return nil
}

// If rendering, must call from main thread
func (c *VNCSession) Flip() (*Screen, []*vncclient.FramebufferUpdateMessage) {
	c.updated.L.Lock()
	defer c.updated.L.Unlock()

	var updates []*vncclient.FramebufferUpdateMessage

	if c.backUpdated {
		c.frontScreen, c.backScreen = c.backScreen, c.frontScreen
		c.backUpdated = false
		updates = c.deferredUpdates
		go func() {
			c.updated.L.Lock()
			defer c.updated.L.Unlock()

			c.applyDeferred()

			if c.pauseUpdates {
				// Restart the framebuffer request cycle
				c.pauseUpdates = false
				log.Infof("[%s] resuming updates", c.label)
				err := c.requestUpdate()
				if err != nil {
					select {
					case c.mgr.Error <- err:
					default:
					}
				}
			}
		}()

		if c.rendererActive {
			// Keep the GL screen fed
			c.renderer.Apply(updates)
		}
	}

	return c.frontScreen, updates
}

// Apply any deferred updates *while holding the lock*
func (c *VNCSession) applyDeferred() error {
	if c.backUpdated {
		return nil
	}
	c.backUpdated = true

	for _, update := range c.deferredUpdates {
		c.applyUpdate(update)
	}
	c.deferredUpdates = nil
	return nil
}

// Apply an update *while holding the lock*
func (c *VNCSession) applyUpdate(update *vncclient.FramebufferUpdateMessage) error {
	var bytes uint32
	start := time.Now().UnixNano()
	for _, rect := range update.Rectangles {
		switch enc := rect.Enc.(type) {
		case *vncclient.RawEncoding:
			bytes += c.applyRect(c.conn, rect, enc.Colors)
		case *vncclient.ZRLEEncoding:
			bytes += c.applyRect(c.conn, rect, enc.Colors)
		case *vncclient.TightEncoding:
			bytes += c.applyRect(c.conn, rect, enc.Colors)
		default:
			return errors.Errorf("unsupported encoding: %T", enc)
		}
	}
	delta := time.Now().UnixNano() - start
	log.Debugf("[%s] Update complete: time=%dus type=%T rectangles=%+v bytes=%d", c.label, delta/1000, update, len(update.Rectangles), bytes)
	return nil
}

func (c *VNCSession) applyRect(conn *vncclient.ClientConn, rect vncclient.Rectangle, colors []vncclient.Color) uint32 {
	var bytes uint32
	// var wg sync.WaitGroup
	// wg.Add(int(rect.Height))
	for y := uint32(0); y < uint32(rect.Height); y++ {
		// go func(y uint32) {
		encStart := uint32(rect.Width) * y
		encEnd := encStart + uint32(rect.Width)

		screenStart := uint32(conn.FramebufferWidth)*(uint32(rect.Y)+y) + uint32(rect.X)
		screenEnd := screenStart + uint32(rect.Width)

		bytes += encEnd - encStart

		copy(c.backScreen.Data[screenStart:screenEnd], colors[encStart:encEnd])
		// wg.Done()
		// }(y)
	}
	// wg.Wait()
	return bytes
}

func (c *VNCSession) maintainFrameBuffer(updates chan *vncclient.FramebufferUpdateMessage) error {
	done := false

	for {
		select {
		case update := <-updates:
			c.updated.L.Lock()
			if err := c.applyDeferred(); err != nil {
				c.updated.L.Unlock()
				return errors.Annotate(err, "when applying deferred updates")
			}

			if err := c.applyUpdate(update); err != nil {
				c.updated.L.Unlock()
				return errors.Annotate(err, "when applying new update")
			}
			c.deferredUpdates = append(c.deferredUpdates, update)

			if len(c.deferredUpdates) >= c.deferredUpdatesMax && !c.pauseUpdates {
				log.Infof("[%s] update queue max of %d reached; pausing further updates", c.label, c.deferredUpdatesMax)
				c.pauseUpdates = true
			}

			// Update complete!
			c.updated.Broadcast()
			c.updated.L.Unlock()
		case <-c.mgr.Done:
			log.Debugf("[%s] shutting down frame buffer thread", c.label)
			return nil
		}

		if !done {
			c.mgr.Ready.Done()
			done = true
		}
	}
}

func (c *VNCSession) SetSubscription(subs []Region) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.config.Subscription = subs
}

func (c *VNCSession) requestUpdate() error {
	if c.config.Subscription != nil {
		for _, sub := range c.config.Subscription {
			err := c.conn.FramebufferUpdateRequest(true, sub.X, sub.Y, sub.Width, sub.Height)
			if err != nil {
				return err
			}
		}
	} else {
		err := c.conn.FramebufferUpdateRequest(true, 0, 0, c.conn.FramebufferWidth, c.conn.FramebufferHeight)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *VNCSession) connect(updates chan *vncclient.FramebufferUpdateMessage) error {
	log.Infof("[%s] opening connection to VNC server", c.label)

	errorCh := make(chan error, 1)
	serverMessageCh := make(chan vncclient.ServerMessage)

	var conn *vncclient.ClientConn
	totalSleep := 0 * time.Second
	for i := 0; ; i++ {
		soft := true
		target, err := net.Dial("tcp", c.config.Address)
		if err == nil {
			conn, err, soft = vncclient.Client(target, &vncclient.ClientConfig{
				Auth: []vncclient.ClientAuth{
					&vncclient.PasswordAuth{
						Password: c.config.Password,
					},
				},
				ServerMessageCh: serverMessageCh,
				ErrorCh:         errorCh,
			})
		}
		if soft == true && c.config.StartTimeout > 0 {
			if totalSleep >= c.config.StartTimeout {
				return errors.Annotatef(err, "could not establish VNC connection to server, and exceeded start timeout of %s by sleeping for %s", c.config.StartTimeout, totalSleep)
			}

			// Choose how long to sleep
			sleepTarget := time.Duration(2*(i+1)) * time.Second
			if sleepTarget > time.Duration(30)*time.Second {
				sleepTarget = time.Duration(30) * time.Second
			}

			log.Infof("VNC server %s is not yet connectable: %s. Sleeping for %s and will try again (%s/%s)", c.config.Address, err, sleepTarget, totalSleep, c.config.StartTimeout)

			totalSleep += sleepTarget
			time.Sleep(sleepTarget)
		} else if err != nil {
			return errors.Annotate(err, "could not establish VNC connection to server")
		} else {
			break
		}
	}

	defer func() {
		c.lock.Lock()
		if c.conn == nil {
			// We never stored the conn object, so we need
			// to close it ourselves.
			log.Infof("[%s] autoclosing connection to VNC server", c.label)
			conn.Close()
		}
		c.lock.Unlock()
	}()

	go func() {
		select {
		case err := <-errorCh:
			c.mgr.Error <- errors.Annotatef(err, "[%s] vnc error", c.label)
		case <-c.mgr.Done:
		}
	}()

	// While the VNC protocol supports more exotic formats, we
	// only want straight RGB with 1 byte per color.
	c.frontScreen = NewScreen(conn.FramebufferWidth, conn.FramebufferHeight)
	c.backScreen = NewScreen(conn.FramebufferWidth, conn.FramebufferHeight)

	err := conn.SetPixelFormat(&vncclient.PixelFormat{
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
	if err != nil {
		return errors.Annotate(err, "could not set pixel format")
	}

	var encoding vncclient.Encoding
	switch c.config.Encoding {
	case "tight":
		encoding = &vncclient.TightEncoding{}
	case "zrle":
		encoding = &vncclient.ZRLEEncoding{}
	case "raw":
		encoding = &vncclient.RawEncoding{}
	default:
		return errors.Errorf("invalid encoding: %s", c.config.Encoding)
	}

	encodings := []vncclient.Encoding{
		encoding,
	}
	if c.config.QualityLevel != -1 {
		encodings = append(encodings, vncclient.QualityLevel(c.config.QualityLevel))
	}
	if c.config.CompressLevel != -1 {
		encodings = append(encodings, vncclient.CompressLevel(c.config.CompressLevel))
	}
	if c.config.FineQualityLevel != -1 {
		encodings = append(encodings, vncclient.FineQualityLevel(c.config.FineQualityLevel))
	}
	if c.config.SubsampleLevel != -1 {
		encodings = append(encodings, vncclient.SubsampleLevel(c.config.SubsampleLevel))
	}

	err = conn.SetEncodings(encodings)
	if err != nil {
		return errors.Annotate(err, "could not set encodings")
	}

	c.lock.Lock()
	// Make the connection visible so it can be used in requestUpdate
	c.conn = conn

	err = c.requestUpdate()
	if err != nil {
		c.conn = nil
		c.lock.Unlock()
		return errors.Annotate(err, "could not send framebuffer update request")
	}

	if c.closed {
		c.conn = nil
		c.lock.Unlock()
		// Conn will be closed by our earlier defer
		return errors.Errorf("[%s] VNCSession object was closed before connection was established", c.label)
	}
	log.Infof("[%s] connection established", c.label)
	c.lock.Unlock()

	// Spin up a screenbuffer thread
	go func() {
		err := c.maintainFrameBuffer(updates)
		if err != nil {
			// Report the error, if any
			select {
			case c.mgr.Error <- err:
			default:
			}
		}
	}()

	for {
		select {
		case msg := <-serverMessageCh:
			log.Debugf("[%s] Just received: %T %+v", c.label, msg, msg)
			switch msg := msg.(type) {
			case *vncclient.FramebufferUpdateMessage:
				updates <- msg
				// Keep re-requesting!
				c.updated.L.Lock()
				if !c.pauseUpdates {
					err := c.requestUpdate()
					if err != nil {
						select {
						case c.mgr.Error <- err:
						default:
						}
					}
				}
				c.updated.L.Unlock()
			}
		case <-c.mgr.Done:
			log.Debugf("[%s] server message goroutine exiting", c.label)
		}
	}
}

type VNCBatch struct {
	sessions map[string]*VNCSession
}

func NewVNCBatch() *VNCBatch {
	return &VNCBatch{
		sessions: map[string]*VNCSession{},
	}
}

func (v *VNCBatch) Close(name string) error {
	if session, ok := v.sessions[name]; ok {
		session.Close()
		delete(v.sessions, name)
	}
	return nil
}

func (v *VNCBatch) Open(name string, config VNCSessionConfig) error {
	if evicted, ok := v.sessions[name]; ok {
		evicted.Close()
	}

	session := NewVNCSession(name, config)
	v.sessions[name] = session
	return nil
}

func (v *VNCBatch) Step(actions map[string][]VNCEvent) (map[string]*Screen, map[string][]*vncclient.FramebufferUpdateMessage, map[string]error) {
	observationN := map[string]*Screen{}
	updatesN := map[string][]*vncclient.FramebufferUpdateMessage{}
	errN := map[string]error{}

	for name, action := range actions {
		session := v.sessions[name]

		observation, updates, err := session.Step(action)
		observationN[name] = observation
		updatesN[name] = updates
		errN[name] = err
	}
	return observationN, updatesN, errN
}

func (v *VNCBatch) SetSubscription(name string, subs []Region) error {
	if session, ok := v.sessions[name]; ok {
		session.SetSubscription(subs)
		return nil
	} else {
		return errors.Errorf("no such session: %s", name)
	}
}

func (v *VNCBatch) SetRenderer(name string, renderer Renderer) error {
	if session, ok := v.sessions[name]; ok {
		return session.SetRenderer(renderer)
	} else {
		return errors.Errorf("no such session: %s", name)
	}
}

func (v *VNCBatch) Render(name string, close bool) error {
	if session, ok := v.sessions[name]; ok {
		return session.Render(close)
	} else {
		return errors.Errorf("no such session: %s", name)
	}
}

func (v *VNCBatch) Flip() (screens []*Screen, updates [][]*vncclient.FramebufferUpdateMessage) {
	for _, session := range v.sessions {
		screen, update := session.Flip()
		screens = append(screens, screen)
		updates = append(updates, update)
	}
	return
}

func (v *VNCBatch) Peek() (screens []*Screen) {
	for _, session := range v.sessions {
		screens = append(screens, session.frontScreen)
	}
	return
}

func (v *VNCBatch) PeekBack() (screens []*Screen) {
	for _, session := range v.sessions {
		screens = append(screens, session.backScreen)
	}
	return
}

// func NewVNCBatch(addresses []string, passwords []string, compressLevel, fineQualityLevel, subsampleLevel int, encoding string, done chan bool, errCh chan error) (*VNCBatch, error) {
// 	batch := &VNCBatch{}
// 	mgr := NewSessionMgr()

// 	for i := range addresses {
// 		address := addresses[i]
// 		password := passwords[i]

// 		batch.sessions = append(batch.sessions, NewVNCSession(address, password, compressLevel, fineQualityLevel, subsampleLevel, encoding, mgr))
// 	}

// 	allReady := make(chan bool)
// 	go func() {
// 		mgr.Ready.Wait()
// 		allReady <- true
// 	}()

// 	select {
// 	case <-allReady:
// 		go func() {
// 			select {
// 			case <-done:
// 				log.Debugf("Closing VNC batch due to user request")
// 				// Translate 'done' closing into closing down
// 				// this pipeline.
// 				close(mgr.Done)
// 			case err := <-mgr.Error:
// 				log.Debugf("Closing VNCBatch due to error: %s", err)

// 				// Capture the relevant error, and let
// 				// the user know.
// 				batch.setError(err)
// 				errCh <- err
// 				close(mgr.Done)
// 			}
// 		}()

// 		return batch, nil
// 	case err := <-mgr.Error:
// 		return nil, err
// 	case <-done:
// 		// upstream requested a cancelation
// 		mgr.Close()
// 		return nil, nil
// 	}
// }
