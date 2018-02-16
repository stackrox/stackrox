import React, { Component } from 'react';
import PropTypes from 'prop-types';
import isEqual from 'lodash/isEqual';

import TableCell from 'Components/TableCell';

class Table extends Component {
    static propTypes = {
        columns: PropTypes.arrayOf(
            PropTypes.shape({
                key: PropTypes.string,
                label: PropTypes.string,
                keyValueFunc: PropTypes.func,
                align: PropTypes.string,
                classFunc: PropTypes.func,
                default: PropTypes.any
            })
        ).isRequired,
        rows: PropTypes.arrayOf(
            PropTypes.shape({
                id: PropTypes.string
            })
        ).isRequired,
        onRowClick: PropTypes.func,
        checkboxes: PropTypes.bool,
        actions: PropTypes.arrayOf(
            PropTypes.shape({
                text: PropTypes.string,
                renderIcon: PropTypes.func,
                className: PropTypes.string,
                onClick: PropTypes.func,
                disabled: PropTypes.bool
            })
        )
    };

    static defaultProps = {
        onRowClick: null,
        checkboxes: false,
        actions: []
    };

    constructor(props) {
        super(props);

        this.state = {
            checked: new Set(),
            selected: null
        };
    }

    getSelectedRows = () => Array.from(this.state.checked);

    clearSelectedRows = () => {
        const { checked } = this.state;
        checked.clear();
        this.setState({ checked });
    };

    rowCheckedHandler = row => event => {
        event.stopPropagation();
        const { checked } = this.state;
        if (!checked.has(row)) checked.add(row);
        else checked.delete(row);
        this.setState({ checked });
    };

    rowClickHandler = row => () => {
        if (this.props.onRowClick) {
            this.props.onRowClick(row);
            this.setState({
                selected: row
            });
        }
    };

    actionClickHandler = (action, row) => event => {
        event.stopPropagation();
        action.onClick(row);
    };

    renderActionButtons = row =>
        this.props.actions.map((button, i) => (
            <button
                key={i}
                className={button.className}
                onClick={this.actionClickHandler(button, row)}
                disabled={button.disabled}
            >
                {button.renderIcon && (
                    <span className="flex items-center">{button.renderIcon(row)}</span>
                )}
                {button.text && (
                    <span className={`${button.renderIcon && 'ml-3'}`}>{button.text}</span>
                )}
            </button>
        ));

    renderHeaders() {
        const tableHeaders = this.props.columns.map(column => {
            const className = `p-3 text-primary-500 border-b border-base-300 hover:text-primary-600 ${
                column.align === 'right' ? 'text-right' : 'text-left'
            }`;
            return (
                <th className={className} key={column.label}>
                    {column.label}
                </th>
            );
        });
        if (this.props.checkboxes) {
            tableHeaders.unshift(
                <th
                    className="p-3 text-primary-500 border-b border-base-300 hover:text-primary-600"
                    key="checkboxTableHeader"
                />
            );
        }
        if (this.props.actions && this.props.actions.length) {
            tableHeaders.push(
                <th
                    className="p-3 text-primary-500 border-b border-base-300 hover:text-primary-600"
                    key="actionsTableHeader"
                >
                    Actions
                </th>
            );
        }
        return <tr>{tableHeaders}</tr>;
    }

    renderBody() {
        const { rows, columns } = this.props;
        const rowClickable = !!this.props.onRowClick;
        return rows.map((row, i) => {
            const tableCells = columns.map(column => (
                <TableCell column={column} row={row} key={`${column.key}`} />
            ));
            if (this.props.checkboxes) {
                tableCells.unshift(
                    <td className="p-3 text-center" key="checkboxTableCell">
                        <input
                            type="checkbox"
                            className="h-4 w-4 cursor-pointer"
                            onClick={this.rowCheckedHandler(row)}
                            checked={this.state.checked.has(row)}
                        />
                    </td>
                );
            }
            if (this.props.actions && this.props.actions.length) {
                tableCells.push(
                    <td className="flex justify-center p-3 text-center" key="actionsTableCell">
                        {this.renderActionButtons(row)}
                    </td>
                );
            }
            return (
                <tr
                    className={`${rowClickable ? 'cursor-pointer' : ''} ${
                        isEqual(this.state.selected, row) ? 'bg-base-200' : ''
                    } border-b border-base-300 hover:bg-base-100`}
                    key={i}
                    onClick={rowClickable ? this.rowClickHandler(row) : null}
                >
                    {tableCells}
                </tr>
            );
        });
    }

    render() {
        return (
            <table className="w-full border-collapse transition">
                <thead>{this.renderHeaders()}</thead>
                <tbody>{this.renderBody()}</tbody>
            </table>
        );
    }
}

export default Table;
