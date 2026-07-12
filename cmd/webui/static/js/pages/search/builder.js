/* ═══════════════════════════════════════════
   TagFile — Search Query Builder Component
   ═══════════════════════════════════════════ */

var CONDITION_TYPES = {
    filename:  {label: '文件名',   ops: [{v:'contains',l:'包含'},{v:'not_contains',l:'不包含'},{v:'equals',l:'等于'}]},
    tag:       {label: '标签',     ops: [{v:'has',l:'包含'},{v:'not_has',l:'不包含'}]},
    tag_value: {label: '标签值',   ops: [{v:'has',l:'包含'},{v:'not_has',l:'不包含'}]},
    size:      {label: '文件大小', ops: [{v:'gt',l:'大于'},{v:'gte',l:'大于等于'},{v:'lt',l:'小于'},{v:'lte',l:'小于等于'}]},
    ext:       {label: '文件类型', ops: [{v:'in',l:'属于'},{v:'not_in',l:'不属于'}]},
    date:      {label: '修改日期', ops: [{v:'gt',l:'晚于'},{v:'gte',l:'不早于'},{v:'lt',l:'早于'},{v:'lte',l:'不晚于'}]}
};

var SIZE_UNITS = [
    {v:1, l:'B'}, {v:1024, l:'KB'}, {v:1048576, l:'MB'}, {v:1073741824, l:'GB'}
];

/* ── Build query object from DOM ── */
function buildQueryFromDOM(container) {
    if (!container) container = document.getElementById('query-builder');
    return parseGroupDOM(container);
}

function parseGroupDOM(groupEl) {
    var logic = 'and';
    var logicBtn = groupEl.querySelector(':scope > .qb-header .qb-logic-toggle, :scope > .qb-logic-toggle');
    if (logicBtn) logic = logicBtn.getAttribute('data-logic') || 'and';

    var conditions = [];
    var children = groupEl.children;
    for (var i = 0; i < children.length; i++) {
        var child = children[i];
        if (child.classList.contains('qb-group')) {
            var sub = parseGroupDOM(child);
            if (sub.conditions.length > 0) conditions.push(sub);
        } else if (child.classList.contains('qb-row')) {
            var cond = parseRowDOM(child);
            if (cond) conditions.push(cond);
        }
    }

    return {logic: logic, conditions: conditions};
}

function parseRowDOM(row) {
    var type = row.querySelector('.qb-type');
    if (!type) return null;
    var typeVal = type.value;
    var op = row.querySelector('.qb-op');
    var opVal = op ? op.value : '';

    var cond = {type: typeVal, op: opVal};

    switch (typeVal) {
        case 'filename':
        case 'tag':
            var input = row.querySelector('.qb-val-input');
            cond.value = input ? input.value.trim() : '';
            break;
        case 'tag_value':
            var tagInput = row.querySelector('.qb-tag-input');
            var valInput = row.querySelector('.qb-val-input');
            cond.tag = tagInput ? tagInput.value.trim() : '';
            cond.value = valInput ? valInput.value.trim() : '';
            break;
        case 'size':
            var numInput = row.querySelector('.qb-val-number');
            var unitSelect = row.querySelector('.qb-val-unit');
            var num = parseFloat(numInput ? numInput.value : '0') || 0;
            var unit = parseInt(unitSelect ? unitSelect.value : '1') || 1;
            cond.value = num * unit;
            break;
        case 'ext':
            var input = row.querySelector('.qb-val-input');
            var raw = input ? input.value.trim() : '';
            cond.value = raw ? raw.split(',').map(function(s){return s.trim();}).filter(Boolean) : [];
            break;
        case 'date':
            var input = row.querySelector('.qb-val-date');
            cond.value = input ? input.value : '';
            break;
    }

    if (!cond.value || (Array.isArray(cond.value) && cond.value.length === 0)) {
        return null;
    }
    return cond;
}

/* ── Restore DOM from query object ── */
function restoreQueryToDOM(query, container) {
    if (!container) container = document.getElementById('query-builder');
    while (container.firstChild) container.removeChild(container.firstChild);
    if (!query || !query.conditions || query.conditions.length === 0) {
        addConditionRow(container);
        return;
    }
    restoreGroupDOM(query, container);
}

function restoreGroupDOM(group, container) {
    for (var i = 0; i < group.conditions.length; i++) {
        var c = group.conditions[i];
        if (c.conditions) {
            var groupEl = createGroupElement(c.logic || 'and');
            container.appendChild(groupEl);
            restoreGroupDOM(c, groupEl);
        } else {
            container.appendChild(createConditionRow(c));
        }
    }
}

/* ── UI Builders ── */
function addConditionRow(container) {
    if (!container) container = document.getElementById('query-builder');
    container.appendChild(createConditionRow(null));
}

function addGroup(container) {
    if (!container) container = document.getElementById('query-builder');
    var groupEl = createGroupElement('and');
    container.appendChild(groupEl);
    addConditionRow(groupEl);
}

function createGroupElement(logic) {
    var group = createEl('div', {className: 'qb-group', style: 'margin:8px 0;padding:12px;border:1px dashed var(--border);border-radius:6px;background:var(--bg-subtle,rgba(0,0,0,0.02))'});

    var header = createEl('div', {className: 'qb-header', style: 'display:flex;justify-content:space-between;align-items:center;margin-bottom:8px'});
    var logicToggle = createEl('button', {
        className: 'btn btn-xs btn-ghost qb-logic-toggle',
        'data-logic': logic,
        onclick: function(e) {
            e.preventDefault();
            var cur = logicToggle.getAttribute('data-logic');
            var next = cur === 'and' ? 'or' : 'and';
            logicToggle.setAttribute('data-logic', next);
            logicToggle.textContent = next.toUpperCase();
        }
    }, logic.toUpperCase());

    var actions = createEl('div', {style: 'display:flex;gap:4px'});
    var addBtn = createEl('button', {className: 'btn btn-xs btn-default', onclick: function(e) { e.preventDefault(); addConditionRow(group); }}, '+ 条件');
    var delBtn = createEl('button', {className: 'btn btn-xs btn-ghost', onclick: function(e) {
        e.preventDefault();
        group.remove();
    }}, '×');
    actions.appendChild(addBtn);
    actions.appendChild(delBtn);

    header.appendChild(logicToggle);
    header.appendChild(actions);
    group.appendChild(header);
    return group;
}

function createConditionRow(cond) {
    var row = createEl('div', {className: 'qb-row', style: 'display:flex;align-items:center;gap:6px;margin:4px 0;flex-wrap:wrap'});

    // Type select
    var typeSelect = createEl('select', {className: 'qb-type', style: 'width:90px'});
    for (var key in CONDITION_TYPES) {
        typeSelect.appendChild(createEl('option', {value: key}, CONDITION_TYPES[key].label));
    }
    row.appendChild(typeSelect);

    // Op select
    var opSelect = createEl('select', {className: 'qb-op', style: 'width:90px'});
    row.appendChild(opSelect);

    // Value area
    var valArea = createEl('div', {className: 'qb-val-area', style: 'display:flex;gap:4px;align-items:center'});
    row.appendChild(valArea);

    // Delete button
    var delBtn = createEl('button', {className: 'btn btn-xs btn-ghost', onclick: function(e) {
        e.preventDefault();
        row.remove();
    }}, '×');
    row.appendChild(delBtn);

    function updateOpsAndValue() {
        var type = typeSelect.value;
        var typeDef = CONDITION_TYPES[type];

        while (opSelect.firstChild) opSelect.removeChild(opSelect.firstChild);
        for (var i = 0; i < typeDef.ops.length; i++) {
            opSelect.appendChild(createEl('option', {value: typeDef.ops[i].v}, typeDef.ops[i].l));
        }

        while (valArea.firstChild) valArea.removeChild(valArea.firstChild);

        switch (type) {
            case 'filename':
            case 'tag':
                valArea.appendChild(createEl('input', {className: 'qb-val-input', type: 'text', placeholder: type === 'tag' ? '标签名...' : '文件名...', style: 'width:160px', autocomplete: 'off'}));
                break;
            case 'tag_value':
                var tagInput = createEl('input', {className: 'qb-tag-input', type: 'text', placeholder: '标签名...', style: 'width:100px', autocomplete: 'off'});
                var valInput = createEl('input', {className: 'qb-val-input', type: 'text', placeholder: '值...', style: 'width:100px'});
                valArea.appendChild(tagInput);
                valArea.appendChild(createEl('span', {style: 'color:var(--ink-muted,#888)'}, ':'));
                valArea.appendChild(valInput);
                break;
            case 'size':
                valArea.appendChild(createEl('input', {className: 'qb-val-number', type: 'number', min: '0', value: '1', style: 'width:80px'}));
                var unitSel = createEl('select', {className: 'qb-val-unit', style: 'width:60px'});
                for (var i = 0; i < SIZE_UNITS.length; i++) {
                    unitSel.appendChild(createEl('option', {value: SIZE_UNITS[i].v}, SIZE_UNITS[i].l));
                }
                unitSel.value = '1048576';
                valArea.appendChild(unitSel);
                break;
            case 'ext':
                valArea.appendChild(createEl('input', {className: 'qb-val-input', type: 'text', placeholder: 'jpg, png, gif...', style: 'width:180px'}));
                break;
            case 'date':
                valArea.appendChild(createEl('input', {className: 'qb-val-date', type: 'date', style: 'width:140px'}));
                break;
        }
    }

    typeSelect.addEventListener('change', updateOpsAndValue);
    updateOpsAndValue();

    // Restore values if cond provided
    if (cond) {
        typeSelect.value = cond.type;
        updateOpsAndValue();
        opSelect.value = cond.op;

        switch (cond.type) {
            case 'filename':
            case 'tag':
                var inp = valArea.querySelector('.qb-val-input');
                if (inp) inp.value = cond.value || '';
                break;
            case 'tag_value':
                var tagInp = valArea.querySelector('.qb-tag-input');
                var valInp = valArea.querySelector('.qb-val-input');
                if (tagInp) tagInp.value = cond.tag || '';
                if (valInp) valInp.value = cond.value || '';
                break;
            case 'size':
                var numInp = valArea.querySelector('.qb-val-number');
                var unitSel = valArea.querySelector('.qb-val-unit');
                if (numInp && unitSel) {
                    var bytes = typeof cond.value === 'number' ? cond.value : parseFloat(cond.value) || 0;
                    var bestUnit = 1;
                    for (var i = SIZE_UNITS.length - 1; i >= 0; i--) {
                        if (bytes >= SIZE_UNITS[i].v && bytes % SIZE_UNITS[i].v === 0) {
                            bestUnit = SIZE_UNITS[i].v;
                            break;
                        }
                    }
                    numInp.value = bytes / bestUnit;
                    unitSel.value = String(bestUnit);
                }
                break;
            case 'ext':
                var inp = valArea.querySelector('.qb-val-input');
                if (inp && Array.isArray(cond.value)) inp.value = cond.value.join(', ');
                break;
            case 'date':
                var inp = valArea.querySelector('.qb-val-date');
                if (inp) inp.value = cond.value || '';
                break;
        }
    }

    return row;
}

/* ── Summary text for history ── */
function queryToSummary(query) {
    if (!query || !query.conditions || query.conditions.length === 0) return '';
    return conditionsToSummary(query.conditions);
}

function conditionsToSummary(conditions) {
    var parts = [];
    for (var i = 0; i < conditions.length; i++) {
        var c = conditions[i];
        if (c.conditions) {
            parts.push('(' + conditionsToSummary(c.conditions) + ')');
        } else {
            var typeName = CONDITION_TYPES[c.type] ? CONDITION_TYPES[c.type].label : c.type;
            var opName = '';
            var typeOps = CONDITION_TYPES[c.type] ? CONDITION_TYPES[c.type].ops : [];
            for (var j = 0; j < typeOps.length; j++) {
                if (typeOps[j].v === c.op) { opName = typeOps[j].l; break; }
            }
            var val = c.value;
            if (c.type === 'size' && typeof val === 'number') {
                if (val >= 1073741824) val = (val/1073741824) + 'GB';
                else if (val >= 1048576) val = (val/1048576) + 'MB';
                else if (val >= 1024) val = (val/1024) + 'KB';
                else val = val + 'B';
            }
            if (Array.isArray(val)) val = val.join(',');
            parts.push(typeName + ':' + opName + ':' + val);
        }
    }
    return parts.join(' AND ');
}
