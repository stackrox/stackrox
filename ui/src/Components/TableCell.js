import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTooltip from 'react-tooltip';
import resolvePath from 'object-resolve-path';

class TableCell extends Component {
    static propTypes = {
        column: PropTypes.shape({}).isRequired,
        row: PropTypes.shape({}).isRequired
    };

    renderToolTip = (column, row) => (
        <ReactTooltip id={`tooltip-${row.id}`} type="dark" effect="solid">
            {column.tooltip(resolvePath(row, column.key))}
        </ReactTooltip>
    );

    renderTableCell = (column, row) => {
        let result = '';
        let value;
        let customClassName;
        if (column.keys) {
            if (column.keyValueFunc) {
                value = column.keyValueFunc(...column.keys.map(key => resolvePath(row, key)));
            } else value = column.keys.map(key => resolvePath(row, key));
            customClassName =
                (column.classFunc &&
                    column.classFunc(...column.keys.map(key => resolvePath(row, key)))) ||
                '';
        } else {
            if (column.keyValueFunc) {
                value = column.keyValueFunc(resolvePath(row, column.key));
            } else value = resolvePath(row, column.key);
            customClassName = (column.classFunc && column.classFunc(value)) || '';
        }
        const className = `p-3 ${
            column.align === 'right' ? 'text-right' : 'text-left'
        } ${customClassName}`;
        if (column.tooltip) {
            result = (
                <td className={className} key={`${column.key}`}>
                    <div className="inline-block" data-tip data-for={`tooltip-${row.id}`}>
                        {value || column.default}
                    </div>
                    {this.renderToolTip(column, row)}
                </td>
            );
        } else
            result = (
                <td className={className} key={`${column.key}`}>
                    {value || column.default}
                </td>
            );
        return result;
    };

    render() {
        return this.renderTableCell(this.props.column, this.props.row);
    }
}

export default TableCell;
