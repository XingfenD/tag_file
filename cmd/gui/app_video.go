package main

import (
	"github.com/XingfenD/tag_file/cmd/gui/player"
)

func (a *App) CheckVLCAvailable() bool {
	_, err := player.NewPlayer()
	if err != nil {
		return false
	}
	return true
}

func (a *App) VideoOpen(filePath string) map[string]any {
	if a.vlcPlayer != nil {
		a.vlcPlayer.Stop()
	}
	p, err := player.NewPlayer()
	if err != nil {
		return map[string]any{"error": "VLC 未安装或初始化失败: " + err.Error()}
	}
	a.vlcPlayer = p
	if err := p.Open(filePath); err != nil {
		return map[string]any{"error": err.Error()}
	}
	if err := p.Play(); err != nil {
		return map[string]any{"error": err.Error()}
	}
	state := p.GetState()
	return map[string]any{
		"playing":  state.Playing,
		"duration": state.Duration,
		"filePath": state.FilePath,
	}
}

func (a *App) VideoPlay() map[string]any {
	if a.vlcPlayer == nil {
		return map[string]any{"error": "未加载视频"}
	}
	if err := a.vlcPlayer.Play(); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"success": true}
}

func (a *App) VideoPause() map[string]any {
	if a.vlcPlayer == nil {
		return map[string]any{}
	}
	a.vlcPlayer.Pause()
	return map[string]any{"success": true}
}

func (a *App) VideoStop() map[string]any {
	if a.vlcPlayer == nil {
		return map[string]any{}
	}
	a.vlcPlayer.Stop()
	return map[string]any{"success": true}
}

func (a *App) VideoSeek(position float64) map[string]any {
	if a.vlcPlayer == nil {
		return map[string]any{}
	}
	a.vlcPlayer.SetPosition(position)
	return map[string]any{"success": true}
}

func (a *App) VideoSetVolume(volume int) map[string]any {
	if a.vlcPlayer == nil {
		return map[string]any{}
	}
	a.vlcPlayer.SetVolume(volume)
	return map[string]any{"success": true}
}

func (a *App) VideoGetState() map[string]any {
	if a.vlcPlayer == nil {
		return map[string]any{"playing": false, "position": 0, "duration": 0, "volume": 0, "filePath": ""}
	}
	state := a.vlcPlayer.GetState()
	return map[string]any{
		"playing":  state.Playing,
		"position": state.Position,
		"duration": state.Duration,
		"volume":   state.Volume,
		"filePath": state.FilePath,
	}
}
