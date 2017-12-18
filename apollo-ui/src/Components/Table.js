import React, { Component } from 'react';
import resolvePath from 'object-resolve-path';

class Table extends Component {
    constructor(props) {
        super(props);

        this.state = {
            active: null
        };
    }

    displayHeaders() {
        return (
            <tr>{this.props.columns.map((column, i) => {
                const className = `p-3 text-primary-500 border-b border-base-300 hover:text-primary-600 ${column.align === 'right' ? 'text-right' : 'text-left'}`;
                return (
                    <th className={className} key={column.label + i}>
                        {column.label}
                    </th>);
            })}
            </tr>
        );
    }

    rowClickHandler = row => () => {
        this.props.onRowClick(row);
    }

    displayBody() {
        const { rows, columns } = this.props;
        const { active } = this.state;
        return rows.map((row, i) => {
            const cols = columns.map((column) => {
                const value = resolvePath(row, column.key);
                const classFunc = column.classFunc || (() => '');
                const className = `p-3 ${active === row ? 'bg-primary-300' : ''} ${column.align === 'right' ? 'text-right' : 'text-left'} ${classFunc(value)}`;
                return <td className={className} key={`${column.key}-${i}`}>{value || column.default}</td>;
            });
            return <tr className="cursor-pointer border-b border-base-300 hover:bg-base-100" key={`row-${i}`} onClick={this.rowClickHandler(row)}>{cols}</tr>;
        });
    }

    render() {
        return (
            <table className="w-full border-collapse">
                <thead>{this.displayHeaders()}</thead>
                <tbody>{this.displayBody()}</tbody>
            </table>
        );
    }
}

export default Table;
