import React, { Component } from 'react';
import { EventEmitter } from 'fbemitter';

class Table extends Component {
    constructor(props) {
        super(props);
    
        this.state = {
            columns: this.props.columns,
            rows: this.props.rows,
            active: null
        }

        this.rowClick = this.rowClick.bind(this);
    }

    componentWillMount() {
        // create a new instance of an event emitter
        this.emitter = new EventEmitter();

        // set up event listeners for this componenet
        this.rowClickListener = this.emitter.addListener('Table:row-clicked', (data) => {
            console.log('Row Clicked', data);
        });
    }

    displayHeaders() {
       return <tr>{this.state.columns.map(function(column, i) {
           return <th className='p-2 text-left border border-grey-light' key={column.label + i}>{column.label}</th>;
       })}</tr>
    }

    displayBody() {
        var rows = this.state.rows;
        var columns = this.state.columns;
        var active = this.state.active;
        var rowClick = this.rowClick;
        return rows.map(function (row, i) {
            var cols = columns.map(function (column, i) {
                var className = `p-2 text-left border border-grey-light ${active === row.id ? 'bg-blue-lightest' : ''}`
                return <td className={className} key={row[column.key]}>{row[column.key]}</td>;
            });
            return <tr className='cursor-pointer' key={row.id} onClick={() => rowClick(row)}>{cols}</tr>
        });
    }

    rowClick(row) {
        (this.state.active === row.id) ? this.setState({ active: null }) : this.setState({ active: row.id });
        this.emitter.emit('Table:row-clicked', row);
    }

    render() {
        return (
            <table className='w-full border-collapse'>
                <thead>{this.displayHeaders()}</thead>
                <tbody>{this.displayBody()}</tbody>
            </table>
        );
    }

    componentWillUnmount() {
        // remove event listeners
        this.rowClickListener.remove();
    }

}

export default Table;
