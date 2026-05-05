// app.js - SPA router and global state
var App = {
    currentPage: 'home',
    currentData: {},

    init: function() {
        var self = this;
        document.querySelectorAll('#sidebar .nav-link').forEach(function(link) {
            link.addEventListener('click', function(e) {
                e.preventDefault();
                var page = this.getAttribute('data-page');
                self.navigate(page);
            });
        });
        this.navigate('home');
    },

    navigate: function(page, data) {
        this.currentPage = page;
        this.currentData = data || {};
        document.querySelectorAll('#sidebar .nav-link').forEach(function(l) {
            l.classList.remove('active');
        });
        var link = document.querySelector('#sidebar [data-page="' + page + '"]');
        if (link) link.classList.add('active');
        this.clearFlash();
        this.render();
    },

    render: function() {
        var page = this.currentPage;
        var data = this.currentData;
        switch (page) {
            case 'home': renderHome(data); break;
            case 'categories': renderCategories(data); break;
            case 'tags': renderTags(data); break;
            case 'files': renderFiles(data); break;
            case 'fileDetail': renderFileDetail(data); break;
            case 'fileGroups': renderFileGroups(data); break;
            case 'search': renderSearch(data); break;
            case 'config': renderConfig(data); break;
            default: renderHome(data);
        }
    },

    flash: function(msg, type) {
        var area = document.getElementById('flash-area');
        var div = document.createElement('div');
        div.className = 'flash flash-' + (type || 'success');
        div.textContent = msg;
        area.appendChild(div);
        setTimeout(function() { div.remove(); }, 4000);
    },

    clearFlash: function() {
        document.getElementById('flash-area').innerHTML = '';
    }
};

document.addEventListener('DOMContentLoaded', function() {
    App.init();
});
