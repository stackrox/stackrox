import React, { Component } from 'react';
import resolvePath from 'object-resolve-path';

class Table extends Component {
    constructor(props) {
        super(props);

        this.state = {
            active: null
        }

        this.rowClick = this.rowClick.bind(this);
    }

    displayHeaders() {
       return <tr>{this.props.columns.map(function(column, i) {
           return <th className="p-3 text-left border-b border-t border-base-300 text-primary-500" key={column.label + i}>
               {column.label}
             </th>;
       })}</tr>
    }

    displayBody() {
        var rows = this.props.rows;
        var columns = this.props.columns;
        var active = this.state.active;
        var rowClick = this.rowClick;
        return rows.map(function (row, i) {
            var cols = columns.map(function (column, i) {
                var className = `p-3 ${active === row ? 'bg-primary-300' : ''}`;
                var value = resolvePath(row, column.key);
                return <td className={className} key={column.key + '-' + i}>{value}</td>;
            });
            return <tr className="cursor-pointer border-b border-base-300" key={i} onClick={() => rowClick(row)}>
                {cols}
              </tr>;
        });
    }

    rowClick(row) {
        this.props.onRowClick(row);
    }

    render() {
        return (
            <table className='w-full border-collapse text-sm'>
                <thead>{this.displayHeaders()}</thead>
                <tbody>{this.displayBody()}</tbody>
            </table>
        );
    }

}

export default Table;
