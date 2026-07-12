/* ═══════════════════════════════════════════
   TagFile — File Detail: Info, Sidebar, Media, Rename
   ═══════════════════════════════════════════ */

function initFileDetailPage() {
    currentGroupID = 0;
    currentPage = 1;
    allTags = [];
    lastDetailData = null;

    var sort = initSortControls({
        fieldId: 'sort-field',
        orderId: 'sort-order-btn',
        storagePrefix: 'tagfile_detail',
        defaultField: 'name',
        defaultOrder: 'asc',
        onChange: function(field, order) {
            currentSort = field;
            currentOrder = order;
            loadFileDetailData(true);
        }
    });
    currentSort = sort.field;
    currentOrder = sort.order;

    /* Navigate between files without full page reload */
    initSidebarNavigation();

    API.loadAllTags().then(function(tags) {
        allTags = tags;
        resolveFilePathFromURL();
        loadFileDetailData(false);
    }).catch(function(err) {
        console.error('Failed to fetch tags:', err);
        allTags = [];
        resolveFilePathFromURL();
        loadFileDetailData(false);
    });
}

function resolveFilePathFromURL() {
    var rawPath = window.location.pathname.substring('/files/detail/'.length);
    try { filePath = decodeURI(rawPath); } catch(e) { filePath = rawPath; }
}

function initSidebarNavigation() {
    /* Intercept clicks on sidebar indexed file links */
    var indexedList = document.getElementById('dir-indexed');
    if (indexedList) {
        indexedList.addEventListener('click', function(e) {
            var link = e.target.closest('a');
            if (!link) return;
            e.preventDefault();
            var href = link.getAttribute('href');
            var rawPath = href.substring('/files/detail/'.length);
            var newPath;
            try { newPath = decodeURI(rawPath); } catch(_) { newPath = rawPath; }
            if (newPath !== filePath) {
                navigateToFile(newPath);
            }
        });
    }

    /* Handle browser back/forward */
    window.addEventListener('popstate', function(e) {
        resolveFilePathFromURL();
        currentGroupID = 0;
        currentPage = 1;
        loadFileDetailData(false);
    });
}

function navigateToFile(newPath) {
    filePath = newPath;
    currentGroupID = 0;
    currentPage = 1;
    history.pushState({filePath: newPath}, '', '/files/detail/' + encodePath(newPath));
    loadFileDetailData(false, true);
}

function updateSidebarCurrent(d) {
    var indexedList = document.getElementById('dir-indexed');
    if (!indexedList) return;
    var items = indexedList.querySelectorAll('li');
    for (var i = 0; i < items.length; i++) {
        var link = items[i].querySelector('a');
        if (!link) continue;
        var href = link.getAttribute('href');
        var match = href === '/files/detail/' + encodePath(d.file.path);
        items[i].classList.toggle('current', match);
    }
}

function loadFileDetailData(sortOnly, skipSidebar) {
    var url = '/api/files/detail/' + encodePath(filePath);
    url += '?group_id=' + (currentGroupID || 0) + '&page=' + (currentPage || 1);
    url += '&sort=' + (currentSort || 'name') + '&order=' + (currentOrder || 'asc');

    API.get(url).then(function(data) {
        var content = document.getElementById('file-detail-content');
        if (!content) return;
        if (!data.ok) {
            clearChildren(content);
            content.appendChild(makeEmptyState('&#128196;', data.msg || '文件不存在', '文件可能已被移动或删除，请返回文件列表查看'));
            return;
        }
        var d = data.detail;

        if (currentGroupID === 0 && d.groups && d.groups.length > 0) {
            currentGroupID = d.groups[0].id;
        }

        lastDetailData = d;
        if (!sortOnly) {
            renderFileInfoCard(d);
            renderMediaPreview(d);
            renderGroupsAndTags(d);
        }
        if (!skipSidebar) {
            renderDirSidebar(d);
        } else {
            updateSidebarCurrent(d);
        }
    }).catch(function(err) {
        var content = document.getElementById('file-detail-content');
        if (!content) return;
        clearChildren(content);
        /* API.get throws on 4xx/5xx — show file-not-found for 400/404, generic error otherwise */
        var isHTTPErr = err && err.message && /^HTTP /.test(err.message);
        if (isHTTPErr) {
            content.appendChild(makeEmptyState('&#128196;', '文件不存在', '文件可能已被移动或删除，请返回文件列表查看'));
        } else {
            content.appendChild(makeEmptyState('&#9888;', '加载失败', '请检查网络连接后重试'));
        }
    });
}

function renderFileInfoCard(d) {
    var content = document.getElementById('file-detail-content');
    var f = d.file;

    /* Tag preview chips */
    var tagsCell;
    var previewTags = (d.tags || []).slice(0, 5);
    if (previewTags.length > 0) {
        tagsCell = createEl('div', {className: 'tag-preview'});
        for (var i = 0; i < previewTags.length; i++) {
            var t = previewTags[i];
            tagsCell.appendChild(createEl('span', {className: 'tag-chip', 'data-tag-name': t.tag_name}, t.tag_path + (t.value ? ':' + t.value : '')));
        }
    } else {
        tagsCell = createEl('span', {className: 'text-muted'}, '无标签');
    }

    /* Rename form */
    var renameInput = createEl('input', {name: 'new_name', required: '', placeholder: '新文件名', value: f.name, style: 'width:280px'});
    var renameForm = createEl('form', {id: 'rename-form', className: 'form-row'},
        createEl('input', {type: 'hidden', name: 'file_path', value: f.path}),
        createEl('div', {className: 'form-group'}, renameInput),
        createEl('button', {type: 'submit', className: 'btn btn-primary'}, '重命名')
    );

    /* Tag list header */
    var refreshBtn = createEl('button', {className: 'btn btn-ghost btn-sm', onclick: function() { loadPage(currentPage); }, style: 'font-size:11px'}, '刷新');
    var groupsLink = createEl('a', {href: '/files/groups/' + encodePath(f.path), className: 'btn btn-default btn-sm'}, '管理标签组');

    var card = createEl('div', {className: 'card'},
        createEl('h3', {className: 'card-title'}, f.name),
        createEl('table', {style: 'margin:14px 0'},
            createEl('tr', null, createEl('th', {style: 'width:80px'}, 'ID'), createEl('td', {className: 'text-mono'}, String(f.id))),
            createEl('tr', null, createEl('th', null, '路径'), createEl('td', {className: 'text-mono'}, f.path)),
            createEl('tr', null, createEl('th', null, '大小'), createEl('td', null, formatSize(f.size))),
            createEl('tr', null, createEl('th', null, '类型'), createEl('td', null, f.type)),
            createEl('tr', null, createEl('th', {id: 'tag-count-th'}, '标签 (' + (d.pagination ? d.pagination.total : 0) + ')'), createEl('td', null, tagsCell))
        ),
        createEl('div', {style: 'margin-top:16px;padding-top:16px;border-top:1px solid var(--border)'},
            createEl('h4', {style: 'font-size:13px;color:var(--ink-light);margin-bottom:8px;font-weight:500'}, '重命名'),
            renameForm
        ),
        createEl('div', {id: 'media-preview'}),
        createEl('div', {style: 'margin-top:16px;padding-top:16px;border-top:1px solid var(--border)'},
            createEl('div', {className: 'flex-between mb-md'},
                createEl('h4', {style: 'margin:0;font-size:14px;color:var(--ink);font-weight:500;font-family:var(--font-serif)'}, '标签列表'),
                createEl('div', {style: 'display:flex;align-items:center;gap:8px'}, groupsLink, refreshBtn)
            ),
            createEl('div', {id: 'groups-and-tags'})
        )
    );

    clearChildren(content);
    content.appendChild(card);

    renameForm.addEventListener('submit', function(e) {
        e.preventDefault();
        var data = new FormData(renameForm);
        var newName = data.get('new_name');
        API.put('/api/files/' + encodePath(f.path), {new_name: newName}).then(function(resp) {
                if (resp.ok) {
                    showToast(resp.msg, 'success');
                    var oldPath = f.path;
                    var idx = oldPath.lastIndexOf('/');
                    var newPath = idx >= 0 ? oldPath.substring(0, idx) + '/' + newName : newName;
                    window.location.href = '/files/detail/' + encodePath(newPath);
                } else {
                    showToast(resp.msg, 'error');
                }
            })
            .catch(API.onError);
    });
}

function refreshFileInfoTags(d) {
    var preview = document.querySelector('.tag-preview');
    if (!preview) return;
    var previewTags = (d.tags || []).slice(0, 5);
    clearChildren(preview);
    if (previewTags.length > 0) {
        for (var i = 0; i < previewTags.length; i++) {
            var t = previewTags[i];
            preview.appendChild(createEl('span', {className: 'tag-chip', 'data-tag-name': t.tag_name}, t.tag_path + (t.value ? ':' + t.value : '')));
        }
    } else {
        preview.appendChild(createEl('span', {className: 'text-muted'}, '无标签'));
    }
    var th = document.getElementById('tag-count-th');
    if (th) th.textContent = '标签 (' + ((d.pagination && d.pagination.total) || 0) + ')';
}

function renderDirSidebar(d) {
    var dirPathEl = document.getElementById('dir-path');
    if (dirPathEl) dirPathEl.textContent = d.dir_path || '';

    var indexedList = document.getElementById('dir-indexed');
    if (indexedList) {
        clearChildren(indexedList);
        if (d.dir_files && d.dir_files.length > 0) {
            for (var i = 0; i < d.dir_files.length; i++) {
                var f = d.dir_files[i];
                var isCurrent = f.path === d.file.path;
                var li = createEl('li', isCurrent ? {className: 'current'} : null);
                var link = createEl('a', {href: '/files/detail/' + encodePath(f.path)},
                    createEl('span', {className: 'dir-file-icon'}, iconForMime(f.type)),
                    createEl('span', {className: 'dir-file-name'}, f.name)
                );
                if (f.tag_cnt > 0) link.appendChild(createEl('span', {className: 'badge'}, String(f.tag_cnt)));
                li.appendChild(link);
                li.appendChild(createEl('span', {className: 'dir-file-size'}, formatSize(f.size)));
                indexedList.appendChild(li);
            }
            staggerChildren(indexedList);
            indexedList.parentElement.style.display = '';
        } else {
            indexedList.parentElement.style.display = 'none';
        }
    }

    var unindexedList = document.getElementById('dir-unindexed');
    if (unindexedList) {
        clearChildren(unindexedList);
        if (d.dir_unindexed && d.dir_unindexed.length > 0) {
            for (var i = 0; i < d.dir_unindexed.length; i++) {
                var f = d.dir_unindexed[i];
                unindexedList.appendChild(createEl('li', null,
                    createEl('span', {className: 'dir-file-icon dimmed'}, iconForMime(f.type)),
                    createEl('span', {className: 'dir-file-name dimmed'}, f.name),
                    createEl('span', {className: 'dir-file-size dimmed'}, formatSize(f.size))
                ));
            }
            unindexedList.parentElement.style.display = '';
        } else {
            unindexedList.parentElement.style.display = 'none';
        }
    }
}

function renderMediaPreview(d) {
    var container = document.getElementById('media-preview');
    if (!container) return;
    clearChildren(container);

    if (d.media_type === 'image') {
        container.appendChild(createEl('img', {src: '/api/files/raw/' + d.file.id, style: 'max-width:100%;max-height:600px;display:block;margin-top:16px', alt: d.file.name}));
    } else if (d.media_type === 'video') {
        if (d.video_playable) {
            var video = createEl('video', {controls: '', style: 'max-width:100%;max-height:600px;margin-top:16px'});
            video.appendChild(createEl('source', {src: '/api/files/raw/' + d.file.id, type: d.file.type}));
            container.appendChild(video);
        } else {
            container.appendChild(createEl('div', {className: 'card', style: 'margin-top:16px;text-align:center'},
                makeEmptyState('&#127910;', '此视频格式（' + d.file.type + '）浏览器不支持直接播放')
            ));
        }
    } else if (d.media_type === 'audio') {
        var audio = createEl('audio', {controls: '', style: 'width:100%;margin-top:16px'});
        audio.appendChild(createEl('source', {src: '/api/files/raw/' + d.file.id, type: d.file.type}));
        container.appendChild(audio);
    }
}

