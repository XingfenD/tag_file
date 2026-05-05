// videoPlayer.js - VLC external player launcher with HTML5 fallback
var vpInterval = null;

function initVideoPlayer(containerID, filePath, fileID) {
    var container = document.getElementById(containerID);
    if (!container) return;

    window.go.main.App.CheckVLCAvailable().then(function(available) {
        if (!available) {
            renderHTML5Fallback(container, filePath);
            return;
        }
        renderVLCLauncher(container, filePath);
    });
}

function renderHTML5Fallback(container, filePath) {
    window.go.main.App.GetMediaURL(filePath).then(function(url) {
        if (!url) {
            container.innerHTML =
                '<div class="card" style="text-align:center;color:#888;padding:24px">' +
                '<p>无法播放此视频格式</p>' +
                '<p style="font-size:13px;margin-top:8px">请安装 <a href="https://www.videolan.org/vlc/" target="_blank">VLC Media Player</a> 以获得完整的视频格式支持。</p>' +
                '</div>';
            return;
        }
        container.innerHTML =
            '<video controls style="max-width:100%;max-height:500px;border-radius:6px;background:#000">' +
            '<source src="' + url + '"></video>';
    });
}

function renderVLCLauncher(container, filePath) {
    container.innerHTML =
        '<div class="vlc-player" style="background:#000;border-radius:6px;overflow:hidden">' +
            '<div class="vlc-screen" style="width:100%;height:400px;display:flex;align-items:center;justify-content:center;color:#fff;font-size:14px;flex-direction:column;gap:12px">' +
                '<span style="font-size:48px">🎬</span>' +
                '<span class="vlc-status">VLC Media Player</span>' +
            '</div>' +
            '<div class="vlc-controls" style="background:#1a1a1a;padding:12px 16px;display:flex;align-items:center;gap:12px;color:#fff">' +
                '<button id="vp-play-btn" class="vlc-btn" onclick="vpTogglePlay(\'' + escapeJs(filePath) + '\')" title="播放" style="background:#e94560;border:none;color:#fff;cursor:pointer;padding:8px 20px;border-radius:4px;font-size:14px">▶ 播放</button>' +
                '<button id="vp-stop-btn" class="vlc-btn" onclick="vpStop()" title="停止" style="background:none;border:1px solid #555;color:#ccc;cursor:pointer;padding:8px 16px;border-radius:4px;font-size:14px">⏹ 停止</button>' +
                '<span id="vp-time" style="font-size:12px;min-width:100px;text-align:center;margin-left:auto">已停止</span>' +
            '</div>' +
        '</div>';
}

function vpTogglePlay(filePath) {
    var btn = document.getElementById('vp-play-btn');
    var status = document.querySelector('.vlc-status');
    if (btn.textContent.indexOf('播放') >= 0) {
        window.go.main.App.VideoOpen(filePath).then(function(r) {
            if (r.error) {
                status.textContent = r.error;
                return;
            }
            btn.textContent = '⏸ 暂停';
            status.textContent = '▶ VLC 正在播放';
            document.getElementById('vp-time').textContent = '播放中...';
            vpStartPolling();
        });
    } else {
        window.go.main.App.VideoPause();
        btn.textContent = '▶ 播放';
        status.textContent = '已暂停（VLC 窗口控制）';
    }
}

function vpStop() {
    vpStopPolling();
    window.go.main.App.VideoStop();
    document.getElementById('vp-play-btn').textContent = '▶ 播放';
    document.querySelector('.vlc-status').textContent = 'VLC Media Player';
    document.getElementById('vp-time').textContent = '已停止';
}

function vpStartPolling() {
    vpStopPolling();
    vpInterval = setInterval(function() {
        window.go.main.App.VideoGetState().then(function(state) {
            if (!state || !state.playing) {
                vpStop();
            }
        });
    }, 1000);
}

function vpStopPolling() {
    if (vpInterval) { clearInterval(vpInterval); vpInterval = null; }
}

var _origNavigate = App.navigate;
App.navigate = function(page, data) {
    vpStopPolling();
    window.go.main.App.VideoStop();
    _origNavigate.call(App, page, data);
};
