package main

import (
	"github.com/XingfenD/tag_file/cmd/gui/player"
)

func (a *App) CheckVLCAvailable() bool {
	return player.Available()
}

func (a *App) VideoOpen(filePath string) map[string]any {
	if a.mpvPlayer != nil {
		a.mpvPlayer.Stop()
	}
	p, err := player.NewPlayer(filePath)
	if err != nil {
		return map[string]any{"error": "mpv 初始化失败: " + err.Error()}
	}
	a.mpvPlayer = p
	state := p.GetState()
	return map[string]any{
		"playing":  state.Playing,
		"duration": state.Duration,
		"filePath": state.FilePath,
	}
}

func (a *App) VideoPlay() map[string]any {
	if a.mpvPlayer == nil {
		return map[string]any{"error": "未加载视频"}
	}
	if err := a.mpvPlayer.Play(); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"success": true}
}

func (a *App) VideoPause() map[string]any {
	if a.mpvPlayer == nil {
		return map[string]any{}
	}
	a.mpvPlayer.Pause()
	return map[string]any{"success": true}
}

func (a *App) VideoStop() map[string]any {
	if a.mpvPlayer == nil {
		return map[string]any{}
	}
	a.mpvPlayer.Stop()
	return map[string]any{"success": true}
}

func (a *App) VideoSeek(position float64) map[string]any {
	if a.mpvPlayer == nil {
		return map[string]any{}
	}
	a.mpvPlayer.SetPosition(position)
	return map[string]any{"success": true}
}

func (a *App) VideoSetVolume(volume int) map[string]any {
	if a.mpvPlayer == nil {
		return map[string]any{}
	}
	a.mpvPlayer.SetVolume(volume)
	return map[string]any{"success": true}
}

func (a *App) VideoSetOverlayRect(x, y, w, h int) map[string]any {
	if a.mpvPlayer == nil {
		return map[string]any{}
	}
	a.mpvPlayer.UpdateOverlay(x, y, w, h)
	return map[string]any{"success": true}
}

func (a *App) VideoGetState() map[string]any {
	if a.mpvPlayer == nil {
		return map[string]any{"playing": false, "position": 0, "duration": 0, "volume": 0, "filePath": ""}
	}
	state := a.mpvPlayer.GetState()
	return map[string]any{
		"playing":  state.Playing,
		"position": state.Position,
		"duration": state.Duration,
		"volume":   state.Volume,
		"filePath": state.FilePath,
	}
}
