// videoPlayer.js - libmpv embedded player with HTML5 fallback
var vpInterval = null;
var vpDuration = 0;
var vpSeeking = false;
var vpFilePath = '';
var overlayObserver = null;

function initVideoPlayer(containerID, filePath, fileID, videoPlayable) {
    var container = document.getElementById(containerID);
    if (!container) return;

    if (videoPlayable) {
        renderHTML5Player(container, filePath);
        return;
    }

    // Use libmpv native overlay for non-browser formats
    renderMpvPlayer(container, filePath);
}

function renderHTML5Fallback(container, filePath) {
    window.go.main.App.GetMediaURL(filePath).then(function(url) {
        if (!url) {
            container.innerHTML =
                '<div class="card" style="text-align:center;color:#888;padding:24px">' +
                '<p>无法播放此视频格式</p>' +
                '<p style="font-size:13px;margin-top:8px">请安装 <a href="https://mpv.io/installation/" target="_blank">mpv</a> 以获得完整的视频格式支持。</p>' +
                '</div>';
            return;
        }
        container.innerHTML =
            '<video controls preload="metadata" style="max-width:100%;max-height:500px;border-radius:6px;background:#000">' +
            '<source src="' + url + '"></video>';
    });
}

function renderMpvPlayer(container, filePath) {
    vpFilePath = filePath;
    // Render a placeholder container for the native overlay,
    // plus a control bar identical to the old VLC UI
    container.innerHTML =
        '<div class="mpv-player" style="background:#000;border-radius:6px;overflow:hidden;position:relative">' +
            '<div id="mpv-overlay-area" style="width:100%;height:400px;background:#000;position:relative">' +
                '<span style="position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);color:#888;font-size:14px">mpv 播放器</span>' +
            '</div>' +
            '<div class="mpv-controls" style="background:#1a1a1a;padding:12px 16px;display:flex;align-items:center;gap:10px;color:#fff;flex-wrap:wrap">' +
                '<button id="vp-play-btn" class="mpv-btn" onclick="vpTogglePlay()" title="播放/暂停" style="background:#e94560;border:none;color:#fff;cursor:pointer;padding:8px 20px;border-radius:4px;font-size:14px;min-width:90px">▶ 播放</button>' +
                '<button id="vp-stop-btn" class="mpv-btn" onclick="vpStop()" title="停止" style="background:none;border:1px solid #555;color:#ccc;cursor:pointer;padding:8px 16px;border-radius:4px;font-size:14px">⏹ 停止</button>' +
                '<span id="vp-time" style="font-size:12px;min-width:110px;text-align:center;white-space:nowrap">00:00 / 00:00</span>' +
                '<input id="vp-seek" type="range" min="0" max="1000" value="0" ' +
                    'onmousedown="vpSeeking=true" onmouseup="vpSeekEnd()" ' +
                    'ontouchstart="vpSeeking=true" ontouchend="vpSeekEnd()" ' +
                    'oninput="vpSeekDrag(this.value)" ' +
                    'style="flex:1;min-width:120px;cursor:pointer;accent-color:#e94560;height:4px">' +
                '<span style="font-size:12px;white-space:nowrap">🔊</span>' +
                '<input id="vp-volume" type="range" min="0" max="200" value="80" ' +
                    'onchange="vpSetVolume(this.value)" ' +
                    'style="width:80px;cursor:pointer;accent-color:#e94560">' +
            '</div>' +
        '</div>';

    // Start tracking overlay area position for the native mpv window
    startOverlayTracking('mpv-overlay-area');
}

function startOverlayTracking(targetID) {
    stopOverlayTracking();
    var target = document.getElementById(targetID);
    if (!target) return;

    function sendRect() {
        var rect = target.getBoundingClientRect();
        window.go.main.App.VideoSetOverlayRect(
            Math.round(rect.left),
            Math.round(rect.top),
            Math.round(rect.width),
            Math.round(rect.height)
        );
    }

    overlayObserver = new ResizeObserver(function() { sendRect(); });
    overlayObserver.observe(target);
    document.addEventListener('scroll', sendRect, true);
    sendRect();
}

function stopOverlayTracking() {
    if (overlayObserver) {
        overlayObserver.disconnect();
        overlayObserver = null;
    }
    document.removeEventListener('scroll', function(){}, true);
}

function vpTogglePlay() {
    var btn = document.getElementById('vp-play-btn');
    if (!btn) return;

    if (btn.textContent.indexOf('播放') >= 0 || btn.textContent.indexOf('▶') >= 0) {
        window.go.main.App.VideoOpen(vpFilePath).then(function(r) {
            if (r.error) {
                showFlash(r.error, 'error');
                return;
            }
            btn.textContent = '⏸ 暂停';
            btn.style.background = '#e94560';
            vpStartPolling();
        });
    } else {
        window.go.main.App.VideoPause().then(function() {
            btn.textContent = '▶ 播放';
            btn.style.background = '#2d8c3c';
        });
    }
}

function vpStop() {
    vpStopPolling();
    stopOverlayTracking();
    window.go.main.App.VideoStop();
    resetControls();
}

function resetControls() {
    var btn = document.getElementById('vp-play-btn');
    if (btn) {
        btn.textContent = '▶ 播放';
        btn.style.background = '#e94560';
    }
    var timeEl = document.getElementById('vp-time');
    if (timeEl) timeEl.textContent = '00:00 / 00:00';
    var seek = document.getElementById('vp-seek');
    if (seek) seek.value = 0;
    vpDuration = 0;
}

function vpSeekDrag(val) {
    var dur = vpDuration || 0;
    if (dur <= 0) return;
    var sec = (parseInt(val) / 1000) * dur;
    var timeEl = document.getElementById('vp-time');
    if (timeEl) {
        timeEl.textContent = formatTime(sec) + ' / ' + formatTime(dur);
    }
}

function vpSeekEnd() {
    vpSeeking = false;
    var seek = document.getElementById('vp-seek');
    if (!seek) return;
    var dur = vpDuration || 0;
    if (dur > 0) {
        var sec = (parseInt(seek.value) / 1000) * dur;
        window.go.main.App.VideoSeek(sec);
    }
}

function vpSetVolume(val) {
    window.go.main.App.VideoSetVolume(parseInt(val));
}

function vpStartPolling() {
    vpStopPolling();
    vpInterval = setInterval(function() {
        window.go.main.App.VideoGetState().then(function(state) {
            if (!state || !state.playing) {
                if (!state || !state.filePath) {
                    vpOnPlaybackEnded();
                }
                return;
            }

            vpDuration = state.duration || 0;

            var timeEl = document.getElementById('vp-time');
            if (timeEl) {
                timeEl.textContent = formatTime(state.position) + ' / ' + formatTime(state.duration);
            }

            if (!vpSeeking) {
                var seek = document.getElementById('vp-seek');
                if (seek && state.duration > 0) {
                    seek.value = Math.min((state.position / state.duration) * 1000, 1000);
                }
            }
        });
    }, 500);
}

function vpStopPolling() {
    if (vpInterval) { clearInterval(vpInterval); vpInterval = null; }
}

function vpOnPlaybackEnded() {
    vpStopPolling();
    resetControls();
    var timeEl = document.getElementById('vp-time');
    var seek = document.getElementById('vp-seek');
    if (timeEl && vpDuration > 0) {
        timeEl.textContent = formatTime(vpDuration) + ' / ' + formatTime(vpDuration);
    }
    if (seek && vpDuration > 0) {
        seek.value = 1000;
    }
}

function formatTime(sec) {
    if (!sec || sec < 0 || isNaN(sec)) return '00:00';
    var m = Math.floor(sec / 60);
    var s = Math.floor(sec % 60);
    return (m < 10 ? '0' : '') + m + ':' + (s < 10 ? '0' : '') + s;
}

function renderHTML5Player(container, filePath) {
    window.go.main.App.GetMediaURL(filePath).then(function(url) {
        if (!url) {
            // Media server unavailable, use mpv overlay instead
            renderMpvPlayer(container, filePath);
            return;
        }
        container.innerHTML =
            '<video controls preload="metadata" style="max-width:100%;max-height:500px;border-radius:6px;background:#000">' +
            '<source src="' + url + '"></video>';
    });
}

var _origNavigate = App.navigate;
App.navigate = function(page, data) {
    vpStopPolling();
    stopOverlayTracking();
    window.go.main.App.VideoStop();
    _origNavigate.call(App, page, data);
};
