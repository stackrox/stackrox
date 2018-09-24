import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTablePropTypes from 'react-table/lib/propTypes';
import Table from 'Components/Table';

class CheckboxTable extends Component {
    static propTypes = {
        columns: ReactTablePropTypes.columns.isRequired,
        rows: PropTypes.arrayOf(PropTypes.object).isRequired,
        onRowClick: PropTypes.func,
        selectedRowId: PropTypes.string,
        toggleRow: PropTypes.func.isRequired,
        toggleSelectAll: PropTypes.func.isRequired,
        selection: PropTypes.arrayOf(PropTypes.string),
        page: PropTypes.number.isRequired
    };

    static defaultProps = {
        selectedRowId: null,
        onRowClick: null,
        selection: []
    };

    setTableRef = table => {
        this.reactTable = table;
    };

    toggleRow = ({ id }) => e => {
        e.stopPropagation();
        this.props.toggleRow(id);
    };

    toggleSelectAll = () => () => {
        this.props.toggleSelectAll();
    };

    someSelected = () => {
        const { selection, rows } = this.props;
        return selection.length !== 0 && selection.length < rows.length;
    };

    allSelected = () => {
        const { selection, rows } = this.props;
        return selection.length !== 0 && selection.length === rows.length;
    };

    addCheckboxColumns = () => {
        const { columns, selection } = this.props;
        return [
            {
                id: 'checkbox',
                accessor: '',
                Cell: ({ original }) => (
                    <input
                        type="checkbox"
                        checked={selection.includes(original.id)}
                        onClick={this.toggleRow(original)}
                    />
                ),
                Header: () => (
                    <input
                        type="checkbox"
                        checked={this.allSelected()}
                        ref={input => {
                            if (input) {
                                input.indeterminate = this.someSelected(); // eslint-disable-line
                            }
                        }}
                        onChange={this.toggleSelectAll()}
                    />
                ),
                sortable: false,
                width: 40
            },
            ...columns
        ];
    };

    render() {
        const { ...rest } = this.props;
        const columns = this.addCheckboxColumns();
        return <Table {...rest} columns={columns} setTableRef={this.setTableRef} />;
    }
}

export default CheckboxTable;
