/* ═══════════════════════════════════════════
   TagFile — Shared Sort Controls
   ═══════════════════════════════════════════ */

/**
 * Initialize sort controls with sessionStorage persistence.
 * @param {Object} opts
 * @param {string} opts.fieldId       - ID of the <select> for sort field
 * @param {string} opts.orderId       - ID of the button for sort order
 * @param {string} opts.storagePrefix - sessionStorage key prefix (e.g. 'tagfile_tree')
 * @param {string} opts.defaultField  - default field value (e.g. 'name')
 * @param {string} opts.defaultOrder  - default order ('asc' or 'desc')
 * @param {Function} opts.onChange     - callback(sortField, sortOrder) when sort changes
 * @returns {{ field: string, order: string }} current sort state
 */
function initSortControls(opts) {
    var fieldSel = document.getElementById(opts.fieldId);
    var orderBtn = document.getElementById(opts.orderId);
    var field = sessionStorage.getItem(opts.storagePrefix + '_sort_field') || opts.defaultField || 'name';
    var order = sessionStorage.getItem(opts.storagePrefix + '_sort_order') || opts.defaultOrder || 'asc';

    function updateOrderBtn() {
        if (!orderBtn) return;
        if (order === 'desc') {
            orderBtn.classList.add('desc');
            orderBtn.innerHTML = '降序 ▾';
        } else {
            orderBtn.classList.remove('desc');
            orderBtn.innerHTML = '升序 ▴';
        }
    }

    if (fieldSel) {
        fieldSel.value = field;
        fieldSel.addEventListener('change', function() {
            field = fieldSel.value;
            sessionStorage.setItem(opts.storagePrefix + '_sort_field', field);
            if (opts.onChange) opts.onChange(field, order);
        });
    }

    if (orderBtn) {
        updateOrderBtn();
        orderBtn.addEventListener('click', function() {
            order = order === 'asc' ? 'desc' : 'asc';
            sessionStorage.setItem(opts.storagePrefix + '_sort_order', order);
            updateOrderBtn();
            if (opts.onChange) opts.onChange(field, order);
        });
    }

    return { field: field, order: order };
}
