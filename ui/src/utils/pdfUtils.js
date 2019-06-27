import toLower from 'lodash/toLower';
import startCase from 'lodash/startCase';
import entityToColumns from 'constants/tableColumns';
import ReactDOMServer from 'react-dom/server';
import flattenObject from 'utils/flattenObject';

const createPDFTable = (tableData, entityType, query, pdfId, tableColumns) => {
    const { standardId } = query;
    const table = document.getElementById('pdf-table');
    const parent = document.getElementById(pdfId);
    if (table) {
        parent.removeChild(table);
    }
    let type = null;
    if (query.groupBy) {
        type = startCase(toLower(query.groupBy));
    } else if (standardId) {
        type = 'Standard';
    }
    const columns = tableColumns || entityToColumns[standardId || entityType];
    if (tableData.length) {
        const headers = columns
            .map(col => col.Header)
            .filter(header => header !== 'id' && header !== 'Id');
        const headerKeys = columns.map(col => col.accessor).filter(header => header !== 'id');
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
                    colValue = 'test';
                    ReactDOMServer.renderToString(
                        columns[index + 1].Cell({ original: val, pdf: true })
                    );
                } else {
                    const flattenedObj = flattenObject(val);
                    colValue =
                        (flattenedObj[key] && flattenedObj[key].replace(/\s+/g, ' ').trim()) ||
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
