package player

import (
	"fmt"
	"sync"

	mpv "github.com/gen2brain/go-mpv"
)

type Player struct {
	mu       sync.Mutex
	mpv      *mpv.Mpv
	overlay  VideoOverlay
	filePath string
	playing  bool
	position float64
	duration float64
	volume   int
	stopCh   chan struct{}
	doneCh   chan struct{}
}

func NewPlayer(filePath string) (*Player, error) {
	p := &Player{
		filePath: filePath,
		volume:   80,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}

	// Create native overlay for video rendering
	overlay, err := newOverlay()
	if err != nil {
		return nil, fmt.Errorf("覆盖层创建失败: %w", err)
	}
	p.overlay = overlay

	// Create mpv instance
	m := mpv.New()
	if m == nil {
		p.overlay.Destroy()
		return nil, fmt.Errorf("mpv 创建失败")
	}

	// Set options before initialization
	m.SetOptionString("input-default-bindings", "no")
	m.SetOptionString("osd-level", "0")
	m.SetOptionString("keep-open", "yes")

	if err := m.Initialize(); err != nil {
		m.TerminateDestroy()
		p.overlay.Destroy()
		return nil, fmt.Errorf("mpv 初始化失败: %w", err)
	}
	p.mpv = m

	// Create overlay at default size; frontend will send real position shortly
	wid, err := p.overlay.Create(0, 0, 640, 400)
	if err != nil {
		p.Close()
		return nil, fmt.Errorf("创建覆盖层失败: %w", err)
	}

	if err := m.SetProperty("wid", mpv.FormatInt64, int64(wid)); err != nil {
		p.Close()
		return nil, fmt.Errorf("设置 mpv wid 失败: %w", err)
	}

	// Load the file after wid is set so mpv renders into our overlay
	if err := m.Command([]string{"loadfile", filePath}); err != nil {
		p.Close()
		return nil, fmt.Errorf("加载文件失败: %w", err)
	}

	p.playing = true

	// Start event loop
	go p.eventLoop()

	return p, nil
}

func (p *Player) eventLoop() {
	defer close(p.doneCh)
	for {
		select {
		case <-p.stopCh:
			return
		default:
		}

		e := p.mv().WaitEvent(100)
		if e == nil {
			continue
		}

		switch e.EventID {
		case mpv.EventShutdown:
			return
		case mpv.EventEnd:
			p.mu.Lock()
			p.playing = false
			if dur, err := p.mpv.GetProperty("duration", mpv.FormatDouble); err == nil {
				if d, ok := dur.(float64); ok {
					p.position = d
					p.duration = d
				}
			}
			p.mu.Unlock()
		case mpv.EventFileLoaded:
			if dur, err := p.mpv.GetProperty("duration", mpv.FormatDouble); err == nil {
				if d, ok := dur.(float64); ok {
					p.mu.Lock()
					p.duration = d
					p.mu.Unlock()
				}
			}
		}
	}
}

func (p *Player) mv() *mpv.Mpv {
	return p.mpv
}

func (p *Player) Play() error {
	err := p.mpv.Command([]string{"set", "pause", "no"})
	if err == nil {
		p.playing = true
	}
	return err
}

func (p *Player) Pause() error {
	return p.mpv.Command([]string{"set", "pause", "yes"})
}

func (p *Player) Stop() error {
	p.mu.Lock()
	// Signal event loop to stop
	if p.stopCh != nil {
		select {
		case <-p.stopCh:
		default:
			close(p.stopCh)
		}
	}
	mpvInst := p.mpv
	p.mu.Unlock()

	if mpvInst != nil {
		mpvInst.Command([]string{"stop"})
		mpvInst.TerminateDestroy()
		<-p.doneCh
		p.mpv = nil
	}

	if p.overlay != nil {
		p.overlay.Destroy()
		p.overlay = nil
	}

	p.playing = false
	return nil
}

func (p *Player) SetPosition(pos float64) error {
	return p.mpv.Command([]string{"seek", fmt.Sprintf("%.1f", pos), "absolute"})
}

func (p *Player) SetVolume(vol int) error {
	err := p.mpv.Command([]string{"set", "volume", fmt.Sprintf("%d", vol)})
	if err == nil {
		p.volume = vol
	}
	return err
}

type State struct {
	Playing  bool    `json:"playing"`
	Position float64 `json:"position"`
	Duration float64 `json:"duration"`
	Volume   int     `json:"volume"`
	FilePath string  `json:"filePath"`
}

func (p *Player) GetState() *State {
	if p.mpv == nil {
		return &State{Playing: false}
	}

	// Get position
	if pos, err := p.mpv.GetProperty("time-pos", mpv.FormatDouble); err == nil {
		if val, ok := pos.(float64); ok {
			p.position = val
		}
	}

	// Get duration
	if dur, err := p.mpv.GetProperty("duration", mpv.FormatDouble); err == nil {
		if val, ok := dur.(float64); ok {
			p.duration = val
		}
	}

	// Get pause state
	if paused, err := p.mpv.GetProperty("pause", mpv.FormatFlag); err == nil {
		if val, ok := paused.(bool); ok {
			p.playing = !val
		}
	}

	return &State{
		Playing:  p.playing,
		Position: p.position,
		Duration: p.duration,
		Volume:   p.volume,
		FilePath: p.filePath,
	}
}

func (p *Player) UpdateOverlay(x, y, w, h int) {
	if p.overlay != nil {
		p.overlay.UpdatePosition(x, y, w, h)
	}
}

func (p *Player) Close() {
	p.Stop()
}

func Available() bool {
	_, err := newOverlay()
	return err == nil
}
