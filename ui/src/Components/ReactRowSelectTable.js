import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import ReactTablePropTypes from 'react-table/lib/propTypes';

const columnHeaderClassName =
    'p-3 text-primary-500 border-b border-base-300 hover:text-primary-600 cursor-pointer truncate select-none relative text-left border-r-0 shadow-none';
const columnClassName = 'p-3 text-left border-r-0 cursor-pointer self-center';
const pageSize = 20;

class ReactRowSelectTable extends Component {
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

    getTrProps = (state, rowInfo) => ({
        onClick: () => {
            if (this.props.onRowClick) this.props.onRowClick(rowInfo.original);
        },
        className:
            rowInfo &&
            rowInfo.original &&
            rowInfo.original[this.props.idAttribute] === this.props.selectedRowId
                ? 'bg-base-100'
                : ''
    });

    render() {
        const { rows, columns, ...rest } = this.props;
        columns.forEach(column =>
            Object.assign(column, {
                className: column.className || columnClassName,
                headerClassName: column.headerClassName || columnHeaderClassName
            })
        );
        return (
            <ReactTable
                data={rows}
                columns={columns}
                getTrProps={this.getTrProps}
                className={`border-0 -highlight ${rows.length > pageSize && 'h-full'}`}
                showPagination={rows.length > pageSize}
                defaultPageSize={rows.length > pageSize ? pageSize : rows.length}
                resizable
                sortable
                defaultSortDesc={false}
                showPageJump={false}
                {...rest}
            />
        );
    }
}

export default ReactRowSelectTable;
