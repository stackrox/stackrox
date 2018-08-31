import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import ReactTablePropTypes from 'react-table/lib/propTypes';
import flattenObject from 'utils/flattenObject';

const columnHeaderClassName =
    'p-3 text-primary-500 border-b border-base-300 hover:text-primary-600 cursor-pointer truncate select-none relative text-left border-r-0 shadow-none';
const columnClassName = 'p-3 text-left border-r-0 cursor-pointer self-center';
const pageSize = 20;

class Table extends Component {
    static propTypes = {
        columns: ReactTablePropTypes.columns.isRequired,
        rows: PropTypes.arrayOf(PropTypes.object).isRequired,
        onRowClick: PropTypes.func,
        selectedRowId: PropTypes.string,
        idAttribute: PropTypes.string,
        noDataText: ReactTablePropTypes.noDataText
    };

    static defaultProps = {
        noDataText: 'No records.',
        selectedRowId: null,
        idAttribute: 'id',
        onRowClick: null
    };

    getTbodyProps = state => {
        const table = [...document.body.getElementsByClassName('rt-table')];
        const tableBody = table[0] && table[0].lastChild;
        const isTableOverflow = state.pageRows && state.pageRows.length < state.minRows;

        if (tableBody && isTableOverflow) {
            tableBody.scrollTop = 0;
        }

        return {
            className: isTableOverflow ? 'overflow-hidden' : ''
        };
    };

    getTrGroupProps = (state, rowInfo) => ({
        className: rowInfo && rowInfo.original ? '' : 'invisible'
    });

    getTrProps = (state, rowInfo) => {
        const flattenedRowInfo = rowInfo && rowInfo.original && flattenObject(rowInfo.original);
        return {
            onClick: () => {
                if (this.props.onRowClick) this.props.onRowClick(rowInfo.original);
            },
            className:
                rowInfo &&
                rowInfo.original &&
                flattenedRowInfo[this.props.idAttribute] === this.props.selectedRowId
                    ? 'bg-base-100'
                    : ''
        };
    };

    getColumnClassName = column => {
        let className = column.className || columnClassName;
        className = column.widthClassName ? `${className} ${column.widthClassName}` : className;
        return column.wrap ? `whitespace-normal overflow-visible ${className}` : className;
    };

    getHeaderClassName = column => {
        const className = column.headerClassName || columnHeaderClassName;
        return column.widthClassName ? `${className} ${column.widthClassName}` : className;
    };

    render() {
        const { rows, columns, ...rest } = this.props;
        columns.forEach(column =>
            Object.assign(column, {
                className: this.getColumnClassName(column),
                headerClassName: this.getHeaderClassName(column)
            })
        );
        return (
            <ReactTable
                ref={r => (this.reactTable = r)} // eslint-disable-line
                data={rows}
                columns={columns}
                getTbodyProps={this.getTbodyProps}
                getTrGroupProps={this.getTrGroupProps}
                getTrProps={this.getTrProps}
                defaultPageSize={pageSize}
                className={`border-0 -highlight w-full ${rows.length > pageSize && 'h-full'}`}
                showPagination={rows.length > pageSize}
                resizable={false}
                sortable
                defaultSortDesc={false}
                showPageJump={false}
                minRows={Math.min(this.props.rows.length, pageSize)}
                {...rest}
            />
        );
    }
}

export default Table;
