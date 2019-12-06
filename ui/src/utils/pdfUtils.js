import toLower from 'lodash/toLower';
import startCase from 'lodash/startCase';
import entityToColumns from 'constants/tableColumns';
import ReactDOMServer from 'react-dom/server';
import flattenObject from 'utils/flattenObject';

/**
 * Prevent AutoTable from breaking lines in the middle of words
 * by setting the minWidth to the longest word in the column
 *
 * WARNING: this may cause the table to exceede the allowed width (just like in HTML)
 * and give a "can't fit page" console error, could be improved.
 *
 * Borrowed from https://github.com/simonbengtsson/jsPDF-AutoTable/issues/568
 */
export function enhanceWordBreak({ doc, cell, column }) {
    if (cell === undefined) {
        return;
    }

    const hasCustomWidth = typeof cell.styles.cellWidth === 'number';

    if (hasCustomWidth || cell.raw == null || cell.colSpan > 1) {
        return;
    }

    let text;

    if (cell.raw instanceof Node) {
        text = cell.raw.innerText;
    } else {
        if (typeof cell.raw === 'object') {
            // not implemented yet
            // when a cell contains other cells (colSpan)
            return;
        }
        text = cell.raw;
    }

    // split cell string by space or "-"
    const words = text.split(/\s+|(?<=-)/);

    // calculate longest word width
    const maxWordUnitWidth = words
        .map(s => Math.floor(doc.getStringUnitWidth(s) * 100) / 100)
        .reduce((a, b) => Math.max(a, b), 0);
    const maxWordWidth = maxWordUnitWidth * (cell.styles.fontSize / doc.internal.scaleFactor);

    const minWidth = cell.padding('horizontal') + maxWordWidth;

    // update minWidth for cell & column

    if (minWidth > cell.minWidth) {
        // eslint-disable-next-line no-param-reassign
        cell.minWidth = minWidth;
    }

    if (cell.minWidth > cell.wrappedWidth) {
        // eslint-disable-next-line no-param-reassign
        cell.wrappedWidth = cell.minWidth;
    }

    if (cell.minWidth > column.minWidth) {
        // eslint-disable-next-line no-param-reassign
        column.minWidth = cell.minWidth;
    }

    if (column.minWidth > column.wrappedWidth) {
        // eslint-disable-next-line no-param-reassign
        column.wrappedWidth = column.minWidth;
    }
}

const createPDFTable = (tableData, entityType, query, pdfId, tableColumns) => {
    const table = document.getElementById('pdf-table');
    const parent = document.getElementById(pdfId);
    if (table && parent.contains(table)) {
        // TODO: fix this.
        // Throwing error sometimes but not related to this PR
        try {
            parent.removeChild(table);
        } catch (err) {
            return;
        }
    }
    let type = null;
    if (query.groupBy) {
        type = startCase(toLower(query.groupBy));
    } else if (query.standardId) {
        type = 'Standard';
    }
    const columns = tableColumns || entityToColumns[query.standardId || entityType];
    if (tableData.length) {
        const filteredColumns = columns.filter(
            ({ accessor, className }) =>
                accessor && className !== 'hidden' && accessor !== 'id' && accessor !== 'checkbox'
        );
        const headers = filteredColumns.map(col => col.Header);
        const headerKeys = filteredColumns.map(col => col.accessor);
        if (tableData[0].rows && type) {
            headers.unshift(type);
            headerKeys.unshift(type);
        }
        const tbl = document.createElement('table');
        tbl.style.width = '100%';
        tbl.setAttribute('border', '1');
        const tbdy = document.createElement('tbody');
        const trh = document.createElement('tr');

        headers.forEach(val => {
            const th = document.createElement('th');
            th.appendChild(document.createTextNode(val));
            trh.appendChild(th);
        });
        tbdy.appendChild(trh);

        const addRows = val => {
            const tr = document.createElement('tr');
            headerKeys.forEach((key, index) => {
                const td = document.createElement('td');
                let colValue = '';
                if (columns[index + 1] && columns[index + 1].Cell) {
                    colValue = ReactDOMServer.renderToString(
                        columns[index + 1].Cell({ original: val, pdf: true })
                    );
                } else {
                    const flattenedObj = flattenObject(val);
                    colValue =
                        (flattenedObj[key] &&
                            String(flattenedObj[key])
                                .replace(/\s+/g, ' ')
                                .trim()) ||
                        'N/A';
                }
                td.innerHTML = colValue;
                tr.appendChild(td);
            });
            tbdy.appendChild(tr);
        };
        tableData.forEach(val => {
            if (val.rows) {
                val.rows.forEach(row => {
                    Object.assign(row, { [type]: val.name });
                    addRows(row);
                });
            } else {
                addRows(val);
            }
        });
        tbl.appendChild(tbdy);
        tbl.id = 'pdf-table';
        tbl.className = 'hidden';
        if (parent) parent.appendChild(tbl);
    }
};

export default createPDFTable;
