import React, { Component } from 'react';
import resolvePath from 'object-resolve-path';
import PropTypes from 'prop-types';

class Table extends Component {
    static propTypes = {
        columns: PropTypes.arrayOf(PropTypes.object).isRequired,
        rows: PropTypes.arrayOf(PropTypes.object).isRequired,
        onRowClick: PropTypes.func
    };

    static defaultProps = {
        onRowClick: null
    };

    constructor(props) {
        super(props);

        this.state = {
            active: null
        };
    }

    displayHeaders() {
        return (
            <tr>{this.props.columns.map((column) => {
                const className = `p-3 text-primary-500 border-b border-base-300 hover:text-primary-600 ${column.align === 'right' ? 'text-right' : 'text-left'}`;
                return (
                    <th className={className} key={column.label}>
                        {column.label}
                    </th>);
            })}
            </tr>
        );
    }

    rowClickHandler = row => () => {
        if (this.props.onRowClick) {
            this.props.onRowClick(row);
        }
    }

    displayBody() {
        const { rows, columns } = this.props;
        const { active } = this.state;
        const rowClickable = !!this.props.onRowClick;
        return rows.map((row, i) => {
            const cols = columns.map((column) => {
                let value = resolvePath(row, column.key);
                if (column.keyValueFunc) value = column.keyValueFunc(value);
                const customClassName = (column.classFunc && column.classFunc(value)) || '';
                const className = `p-3 ${active === row ? 'bg-primary-300' : ''} ${column.align === 'right' ? 'text-right' : 'text-left'} ${customClassName}`;
                return <td className={className} key={`${column.key}`}>{value || column.default}</td>;
            });
            return (
                <tr
                    className={`${rowClickable ? 'cursor-pointer' : ''} border-b border-base-300 hover:bg-base-100`}
                    key={`row-${i}`}
                    onClick={rowClickable ? this.rowClickHandler(row) : null}
                >
                    {cols}
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
