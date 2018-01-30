import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTooltip from 'react-tooltip';

import resolvePath from 'object-resolve-path';

class TableCell extends Component {
    static propTypes = {
        column: PropTypes.shape({}).isRequired,
        row: PropTypes.shape({}).isRequired
    }

    renderToolTip = (column, row) => (
        <ReactTooltip id={`tooltip-${row.id}`} type="dark" effect="solid">
            {column.tooltip(resolvePath(row, column.key))}
        </ReactTooltip>
    );

    renderTableCell = (column, row) => {
        let value = resolvePath(row, column.key);
        if (column.keyValueFunc) value = column.keyValueFunc(value);
        const customClassName = (column.classFunc && column.classFunc(value)) || '';
        const className = `p-3 ${column.align === 'right' ? 'text-right' : 'text-left'} ${customClassName}`;
        if (column.tooltip) {
            return (
                <td className={className} key={`${column.key}`}>
                    <div className="inline-block" data-tip data-for={`tooltip-${row.id}`}>{value || column.default}</div>
                    {this.renderToolTip(column, row)}
                </td>
            );
        }
        return (
            <td className={className} key={`${column.key}`}>
                {value || column.default}
            </td>
        );
    }

    render() {
        return this.renderTableCell(this.props.column, this.props.row);
    }
}

export default TableCell;
