import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import ReactTablePropTypes from 'react-table/lib/propTypes';
import flattenObject from 'utils/flattenObject';

export const defaultHeaderClassName =
    'px-2 py-4 pb-3 font-700 text-base-600 border-b border-base-300 hover:bg-primary-200 hover:text-primary-700 truncate select-none relative text-left border-r-0 shadow-none leading-normal';
export const defaultColumnClassName =
    'px-2 py-2 font-600 text-base-600 text-left border-r-0 self-center leading-normal';
export const wrapClassName = 'whitespace-normal overflow-visible';
export const rtTrActionsClassName =
    'rt-tr-actions hidden pin-r p-0 mr-2 w-auto text-left self-center shadow';
export const pageSize = 50;

class Table extends Component {
    static propTypes = {
        columns: ReactTablePropTypes.columns.isRequired,
        rows: PropTypes.arrayOf(PropTypes.object).isRequired,
        onRowClick: PropTypes.func,
        selectedRowId: PropTypes.string,
        idAttribute: PropTypes.string,
        noDataText: ReactTablePropTypes.noDataText,
        setTableRef: PropTypes.func,
        page: PropTypes.number
    };

    static defaultProps = {
        noDataText: 'No records.',
        selectedRowId: null,
        idAttribute: 'id',
        onRowClick: null,
        setTableRef: null,
        page: 0
    };

    getTrGroupProps = (state, rowInfo) => ({
        className: rowInfo && rowInfo.original ? '' : 'hidden'
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
                    ? 'row-active'
                    : ''
        };
    };

    getColumnClassName = column => column.className || defaultColumnClassName;

    getHeaderClassName = column => column.headerClassName || defaultHeaderClassName;

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
                ref={this.props.setTableRef}
                data={rows}
                columns={columns}
                getTrGroupProps={this.getTrGroupProps}
                getTrProps={this.getTrProps}
                defaultPageSize={pageSize}
                className="flex flex-1 overflow-auto border-0 w-full h-full"
                resizable={false}
                sortable
                defaultSortDesc={false}
                showPageJump={false}
                minRows={Math.min(this.props.rows.length, pageSize)}
                page={this.props.page}
                showPagination={false}
                {...rest}
            />
        );
    }
}

export default Table;
