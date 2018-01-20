import React, { Component } from 'react';
import resolvePath from 'object-resolve-path';
import PropTypes from 'prop-types';

class Table extends Component {
    static propTypes = {
        columns: PropTypes.arrayOf(PropTypes.object).isRequired,
        rows: PropTypes.arrayOf(PropTypes.shape({
            id: PropTypes.string
        })).isRequired,
        onRowClick: PropTypes.func,
        checkboxes: PropTypes.bool
    };

    static defaultProps = {
        onRowClick: null,
        checkboxes: false
    };

    constructor(props) {
        super(props);

        this.state = {
            active: null,
            selected: new Set()
        };
    }

    getSelectedRows = () => Array.from(this.state.selected);

    clearSelectedRows = () => {
        const { selected } = this.state;
        selected.clear();
        this.setState({ selected });
    }

    rowCheckedHandler = row => (event) => {
        event.stopPropagation();
        const { selected } = this.state;
        if (!selected.has(row)) selected.add(row);
        else selected.delete(row);
        this.setState({ selected });
    };

    rowClickHandler = row => () => {
        if (this.props.onRowClick) {
            this.props.onRowClick(row);
        }
    }

    displayHeaders() {
        const tableHeaders = this.props.columns.map((column) => {
            const className = `p-3 text-primary-500 border-b border-base-300 hover:text-primary-600 ${column.align === 'right' ? 'text-right' : 'text-left'}`;
            return (
                <th className={className} key={column.label}>
                    {column.label}
                </th>);
        });
        if (this.props.checkboxes) {
            tableHeaders.unshift(<th className="p-3 text-primary-500 border-b border-base-300 hover:text-primary-600" key="checkboxTableHeader" />);
        }
        return (
            <tr>{tableHeaders}</tr>
        );
    }

    displayBody() {
        const { rows, columns } = this.props;
        const { active } = this.state;
        const rowClickable = !!this.props.onRowClick;
        return rows.map((row, i) => {
            const tableCells = columns.map((column) => {
                let value = resolvePath(row, column.key);
                if (column.keyValueFunc) value = column.keyValueFunc(value);
                const customClassName = (column.classFunc && column.classFunc(value)) || '';
                const className = `p-3 ${active === row ? 'bg-primary-300' : ''} ${column.align === 'right' ? 'text-right' : 'text-left'} ${customClassName}`;
                return <td className={className} key={`${column.key}`}>{value || column.default}</td>;
            });
            if (this.props.checkboxes) {
                tableCells.unshift((
                    <td className="text-center" key="checkboxTableCell" >
                        <input type="checkbox" className="h-4 w-4 cursor-pointer" onClick={this.rowCheckedHandler(row)} />
                    </td>
                ));
            }
            return (
                <tr
                    className={`${rowClickable ? 'cursor-pointer' : ''} border-b border-base-300 hover:bg-base-100`}
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
                <thead>{this.displayHeaders()}</thead>
                <tbody>{this.displayBody()}</tbody>
            </table>
        );
    }
}

export default Table;
