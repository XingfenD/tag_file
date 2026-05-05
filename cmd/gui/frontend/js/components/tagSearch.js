// tagSearch.js - Tag autocomplete dropdown (adapted from shared_js.html)
function setupTagSearch(inputId, dropdownId, onSelect) {
    var input = document.getElementById(inputId);
    var dropdown = document.getElementById(dropdownId);
    var tags = [];
    var selectedIndex = -1;

    function filter(val) {
        var v = val.toLowerCase();
        return tags.filter(function(t) {
            return t.name.toLowerCase().indexOf(v) >= 0 || t.path.toLowerCase().indexOf(v) >= 0;
        });
    }

    function showDropdown(items) {
        dropdown.innerHTML = '';
        if (items.length === 0) { dropdown.style.display = 'none'; return; }
        items.forEach(function(item, i) {
            var div = document.createElement('div');
            div.className = 'tag-search-item' + (i === selectedIndex ? ' active' : '');
            div.setAttribute('data-id', item.id);
            div.setAttribute('data-name', item.name);
            div.textContent = item.path;
            div.addEventListener('click', function() {
                input.value = item.path;
                input.setAttribute('data-tag-id', item.id);
                dropdown.style.display = 'none';
                if (onSelect) onSelect(item);
            });
            dropdown.appendChild(div);
        });
        dropdown.style.display = 'block';
    }

    input.addEventListener('input', function() {
        var items = filter(this.value);
        selectedIndex = -1;
        showDropdown(items);
    });

    input.addEventListener('keydown', function(e) {
        var items = dropdown.querySelectorAll('.tag-search-item');
        if (e.key === 'ArrowDown') {
            e.preventDefault();
            selectedIndex = Math.min(selectedIndex + 1, items.length - 1);
            showDropdown(filter(input.value));
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            selectedIndex = Math.max(selectedIndex - 1, -1);
            showDropdown(filter(input.value));
        } else if (e.key === 'Enter') {
            e.preventDefault();
            if (selectedIndex >= 0 && items[selectedIndex]) {
                items[selectedIndex].click();
            }
        } else if (e.key === 'Escape') {
            dropdown.style.display = 'none';
        }
    });

    document.addEventListener('click', function(e) {
        if (e.target !== input && e.target !== dropdown && !dropdown.contains(e.target)) {
            dropdown.style.display = 'none';
        }
    });

    return {
        setTags: function(t) { tags = t; }
    };
}
