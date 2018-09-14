/**
 * Utility functions to help with checkbox actions
 */

/**
 *  Toggles a row selected/unselected in checkbox tables
 *
 *  @param {!String} id the id of the row to toggle
 *  @param {!Object[]} selection the current selection
 *  @returns {!Object[]} the modified selection
 */
export function toggleRow(id, selection) {
    const modifiedSelection = [...selection];
    const keyIndex = modifiedSelection.indexOf(id);
    // check to see if the key exists
    if (keyIndex >= 0) modifiedSelection.splice(keyIndex, 1);
    else modifiedSelection.push(id);
    return modifiedSelection;
}

/**
 *  Toggles all selected/unselected rows in checkbox tables.
 *  If some or none are selected, all become selected on that page,
 *  else if all are selected in the entire table, all become unselected in the table
 *
 *  @param {!String} rowsLength the length of the table
 *  @param {!Object[]} selection the current selection
 *  @returns {!Object[]} the modified selection
 */
export function toggleSelectAll(rowsLength, selection, tableRef) {
    const selectedAll = selection.length !== 0 && selection.length === rowsLength;
    let modifiedSelection = [];
    // we need to get at the internals of ReactTable, passed through by ref
    // the 'sortedData' property contains the currently accessible records based on the filter and sort
    const { sortedData, page, pageSize } = tableRef.getResolvedState();
    const startIndex = page * pageSize;
    const nextPageIndex = (page + 1) * pageSize;

    if (!selectedAll) {
        modifiedSelection = [...selection];
        let previouslySelected = 0;
        // we just push all the IDs onto the selection array of the currently selected page
        for (let i = startIndex; i < nextPageIndex; i += 1) {
            if (!sortedData[i]) break;
            const { id } = sortedData[i].checkbox;
            const keyIndex = modifiedSelection.indexOf(id);
            // if already selected, don't add again, else add to the selection
            if (keyIndex >= 0) previouslySelected += 1;
            else modifiedSelection.push(id);
        }
        // if all were previously selected on the current page, unselect all on page
        if (
            previouslySelected === pageSize ||
            previouslySelected === sortedData.length % pageSize
        ) {
            for (let i = startIndex; i < nextPageIndex; i += 1) {
                if (!sortedData[i]) break;
                const { id } = sortedData[i].checkbox;
                const keyIndex = modifiedSelection.indexOf(id);
                modifiedSelection.splice(keyIndex, 1);
            }
        }
    }
    return modifiedSelection;
}
