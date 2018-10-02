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

    toggleRowHandler = ({ id }) => () => {
        this.props.toggleRow(id);
    };

    stopPropagationOnClick = e => e.stopPropagation();

    toggleSelectAllHandler = () => {
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
                        onChange={this.toggleRowHandler(original)}
                        onClick={this.stopPropagationOnClick} // don't want checkbox click to select the row
                    />
                ),
                Header: () => (
                    <input
                        type="checkbox"
                        checked={this.allSelected()}
                        ref={input => {
                            if (input) {
                                input.indeterminate = this.someSelected(); // eslint-disable-line no-param-reassign
                            }
                        }}
                        onChange={this.toggleSelectAllHandler}
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
