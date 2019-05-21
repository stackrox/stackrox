import toLower from 'lodash/toLower';
import startCase from 'lodash/startCase';
import isEmpty from 'lodash/isEmpty';
import entityToColumns from 'constants/tableColumns';
import flattenObject from 'utils/flattenObject';
import {
    controlsTableColumns,
    nodesTableColumns,
    deploymentsTableColumns,
    clustersTableColumns
} from 'Containers/Compliance/List/evidenceTableColumns';
import ReactDOM from 'react-dom';

const createPDFTable = (tableData, entityType, query, pdfId, resourceType) => {
    let standardId = null;
    if (query && !isEmpty(query)) {
        standardId = query;
    }

    const table = document.getElementById('pdf-table');
    const parent = document.getElementById(pdfId);
    if (table) {
        parent.removeChild(table);
    }
    let type = null;
    if (query && query.groupBy && query.groupBy !== '') {
        type = startCase(toLower(query.groupBy));
    } else if (standardId) {
        type = 'Standard';
    }
    let columns = entityToColumns[entityType];
    if (entityType === 'CONTROL' || resourceType === 'CONTROL') {
        switch (resourceType) {
            case 'NODE':
                columns = nodesTableColumns;
                break;
            case 'DEPLOYMENT':
                columns = deploymentsTableColumns;
                break;
            case 'CLUSTER':
                columns = clustersTableColumns;
                break;
            default:
                columns = controlsTableColumns;
                break;
        }
    }
    if (tableData && columns) {
        const headers = columns.filter(col => col.HeaderText !== 'id').map(col => col.HeaderText);

        const headerKeys = columns.filter(col => col.HeaderText !== 'id').map(col => col.accessor);
        const cells = columns.filter(col => col.HeaderText !== 'id').map(col => col.Cell);

        if (tableData[0] && tableData[0].rows && type) {
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
                const trimmedStr = document.createTextNode(val[key] || 'N/A');
                td.appendChild(trimmedStr);
                if (cells[index]) {
                    const str = cells[index](val);
                    if (typeof str === 'object') {
                        // eslint-disable-next-line
                        ReactDOM.unmountComponentAtNode(td);
                        // eslint-disable-next-line
                        ReactDOM.render(str, td);
                    }
                }
                tr.appendChild(td);
            });
            tbdy.appendChild(tr);
        };
        let rowsData = null;
        if (tableData.rows) {
            rowsData = tableData.rows;
        } else {
            rowsData = tableData;
        }

        rowsData = rowsData.map(row => ({ ...flattenObject(row), original: row }));
        rowsData.forEach(val => {
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
