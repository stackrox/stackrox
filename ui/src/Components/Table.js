import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import ReactTablePropTypes from 'react-table/lib/propTypes';
import flattenObject from 'utils/flattenObject';

export const nonSortableHeaderClassName =
    'px-2 py-4 pb-3 font-700 text-base-600 select-none relative text-left border-r-0 leading-normal';
export const defaultHeaderClassName = `${nonSortableHeaderClassName} hover:bg-primary-200 hover:z-1 hover:text-primary-700`;
export const defaultColumnClassName =
    'p-2 flex items-center font-600 text-base-600 text-left border-r-0 leading-normal';
export const wrapClassName = 'whitespace-normal overflow-visible';
export const rtTrActionsClassName =
    'rt-tr-actions hidden right-0 p-0 mr-2 w-auto text-left self-center';
export const DEFAULT_PAGE_SIZE = 50;

class Table extends Component {
    static propTypes = {
        columns: ReactTablePropTypes.columns.isRequired,
        rows: PropTypes.arrayOf(PropTypes.object).isRequired,
        onRowClick: PropTypes.func,
        selectedRowId: PropTypes.string,
        manual: PropTypes.bool,
        idAttribute: PropTypes.string,
        noDataText: ReactTablePropTypes.noDataText,
        setTableRef: PropTypes.func,
        page: PropTypes.number,
        trClassName: PropTypes.string,
        showThead: PropTypes.bool,
        defaultSorted: PropTypes.arrayOf(PropTypes.object)
    };

    static defaultProps = {
        noDataText: 'No records.',
        selectedRowId: null,
        manual: false,
        idAttribute: 'id',
        onRowClick: null,
        setTableRef: null,
        page: 0,
        trClassName: '',
        showThead: true,
        defaultSorted: []
    };

    getTheadProps = () => {
        if (!this.props.showThead) {
            return {
                style: { display: 'none' }
            };
        }
        // returns an object, if there are no styles to override
        return {};
    };

    getTrGroupProps = (state, rowInfo) => ({
        className: rowInfo && rowInfo.original ? this.props.trClassName : 'hidden'
    });

    getTrProps = (state, rowInfo) => {
        const flattenedRowInfo = rowInfo && rowInfo.original && flattenObject(rowInfo.original);

        const classes = [];
        if (rowInfo && rowInfo.original) {
            classes.push(
                flattenedRowInfo[this.props.idAttribute] === this.props.selectedRowId
                    ? 'row-active'
                    : ''
            );
            classes.push(rowInfo.original.disabled ? 'data-test-disabled' : '');
        }
        classes.push(this.props.onRowClick ? '' : 'cursor-default');
        return {
            onClick: () => {
                if (this.props.onRowClick) {
                    this.props.onRowClick(rowInfo.original);
                }
            },
            className: classes.join(' ')
        };
    };

    getColumnClassName = column => column.className || defaultColumnClassName;

    getHeaderClassName = column => column.headerClassName || defaultHeaderClassName;

    render() {
        const { rows, columns, defaultSorted, manual, ...rest } = this.props;
        if (!columns || !columns.length) return null;
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
                getTheadProps={this.getTheadProps}
                defaultPageSize={DEFAULT_PAGE_SIZE}
                defaultSorted={defaultSorted}
                className={`flex flex-1 overflow-auto border-0 w-full h-full z-0 ${
                    rest.expanded ? 'expanded' : ''
                } `}
                resizable={false}
                sortable
                defaultSortDesc={false}
                showPageJump={false}
                minRows={Math.min(this.props.rows.length, DEFAULT_PAGE_SIZE)}
                page={this.props.page}
                showPagination={false}
                manual={manual}
                {...rest}
            />
        );
    }
}

export default Table;
