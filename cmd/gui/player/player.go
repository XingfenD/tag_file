package player

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type Player struct {
	mu       sync.Mutex
	cmd      *exec.Cmd
	filePath string
	started  time.Time
	duration float64
	playing  bool
}

func NewPlayer() (*Player, error) {
	_, err := findVLC()
	if err != nil {
		return nil, err
	}
	return &Player{}, nil
}

func findVLC() (string, error) {
	// Try common VLC install locations
	candidates := []string{
		"vlc",
		"C:\\Program Files\\VideoLAN\\VLC\\vlc.exe",
		"C:\\Program Files (x86)\\VideoLAN\\VLC\\vlc.exe",
	}
	for _, p := range candidates {
		if p == "vlc" {
			if _, err := exec.LookPath("vlc"); err == nil {
				return "vlc", nil
			}
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("VLC 未安装")
}

func (p *Player) Open(filePath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Stop()

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	vlcPath, err := findVLC()
	if err != nil {
		return err
	}

	p.cmd = exec.Command(vlcPath,
		"--play-and-exit",
		"--no-video-title-show",
		"--quiet",
		absPath,
	)
	p.cmd.Stdout = os.Stdout
	p.cmd.Stderr = os.Stderr

	if err := p.cmd.Start(); err != nil {
		p.cmd = nil
		return fmt.Errorf("无法启动 VLC: %w", err)
	}

	p.filePath = filePath
	p.started = time.Now()
	p.playing = true
	return nil
}

func (p *Player) Play() error {
	// VLC process is already running; no-op for external player
	return nil
}

func (p *Player) Pause() error {
	// Not supported via command line; user controls VLC directly
	return nil
}

func (p *Player) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
		p.cmd.Wait()
		p.cmd = nil
	}
	p.playing = false
	p.filePath = ""
	return nil
}

func (p *Player) SetPosition(pos float64) error {
	// Not supported via command line; user controls VLC directly
	return nil
}

func (p *Player) SetVolume(vol int) error {
	return nil
}

type State struct {
	Playing  bool    `json:"playing"`
	Position float64 `json:"position"`
	Duration float64 `json:"duration"`
	Volume   int     `json:"volume"`
	FilePath string  `json:"filePath"`
}

func (p *Player) GetState() *State {
	p.mu.Lock()
	defer p.mu.Unlock()

	running := p.cmd != nil && p.cmd.Process != nil
	if !running {
		if p.playing {
			p.playing = false
		}
		return &State{Playing: false, FilePath: p.filePath}
	}

	// Check if process is still running
	if p.cmd.ProcessState != nil && p.cmd.ProcessState.Exited() {
		p.playing = false
		return &State{Playing: false, FilePath: p.filePath}
	}

	return &State{
		Playing:  true,
		Position: 0,
		Duration: time.Since(p.started).Seconds(),
		Volume:   80,
		FilePath: p.filePath,
	}
}

func (p *Player) Close() {
	p.Stop()
}
