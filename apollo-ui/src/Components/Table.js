import React, { Component } from 'react';
import emitter from 'emitter';

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
           return <th className='p-2 text-left border border-grey-light' key={column.label + i}>{column.label}</th>;
       })}</tr>
    }

    displayBody() {
        var rows = this.props.rows;
        var columns = this.props.columns;
        var active = this.state.active;
        var rowClick = this.rowClick;
        return rows.map(function (row, i) {
            var cols = columns.map(function (column, i) {
                var className = `p-2 text-left border border-grey-light ${active === row.id ? 'bg-blue-lightest' : ''}`
                return <td className={className} key={row[column.key]}>{row[column.key]}</td>;
            });
            return <tr className='cursor-pointer' key={i} onClick={() => rowClick(row)}>{cols}</tr>
        });
    }

    rowClick(row) {
        if (this.state.active === row.id) {
            this.setState({ active: null });
            emitter.emit('Table:row-selected', null);
        }
        else {
            this.setState({ active: row.id });
            emitter.emit('Table:row-selected', row);
        }
    }

    render() {
        return (
            <table className='w-full border-collapse'>
                <thead>{this.displayHeaders()}</thead>
                <tbody>{this.displayBody()}</tbody>
            </table>
        );
    }

}

export default Table;
