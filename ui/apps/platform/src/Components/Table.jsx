import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table-6';
import ReactTablePropTypes from 'react-table-6/lib/propTypes';
import flattenObject from 'utils/flattenObject';

export const nonSortableHeaderClassName =
    'px-2 py-4 pb-3 font-700 text-base-600 select-none relative text-left border-r-0 leading-normal';
export const defaultHeaderClassName = `${nonSortableHeaderClassName} hover:bg-primary-200 hover:z-1 hover:text-primary-700`;
export const defaultColumnClassName =
    'p-2 flex items-center text-base-600 text-left border-r-0 leading-normal';
export const wrapClassName = 'whitespace-normal overflow-visible';
export const rtTrActionsClassName =
    'rt-tr-actions hidden right-0 p-0 mr-2 w-auto text-left self-center';
export const DEFAULT_PAGE_SIZE = 50;

export const Expander = ({ isExpanded }) => {
    return (
        <div className={`rt-expander w-1 pt-2 pointer-events-auto ${isExpanded ? '-open' : ''}`} />
    );
};

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
        defaultSorted: PropTypes.arrayOf(PropTypes.object),
        pageSize: PropTypes.number,
        noHorizontalPadding: PropTypes.bool,
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
        defaultSorted: [],
        pageSize: DEFAULT_PAGE_SIZE,
        noHorizontalPadding: false,
    };

    getTheadProps = () => {
        if (!this.props.showThead) {
            return {
                style: { display: 'none' },
            };
        }
        // returns an object, if there are no styles to override
        return {};
    };

    getTrGroupProps = (state, rowInfo) => ({
        className: rowInfo && rowInfo.original ? this.props.trClassName : 'hidden',
    });

    getTrProps = (state, rowInfo) => {
        const flattenedRowInfo = rowInfo && rowInfo.original && flattenObject(rowInfo.original);

        const classes = [];
        if (rowInfo && rowInfo.original) {
            if (flattenedRowInfo[this.props.idAttribute] === this.props.selectedRowId) {
                classes.push('row-active');
            }
            if (rowInfo.original.disabled) {
                classes.push('data-test-disabled');
            }
        }
        if (!this.props.onRowClick) {
            classes.push('cursor-default');
        }
        return {
            onClick: () => {
                if (this.props.onRowClick) {
                    this.props.onRowClick(rowInfo.original);
                }
            },
            className: classes.join(' '),
        };
    };

    getHorizontalPaddingClass = () => {
        return this.props.noHorizontalPadding ? 'px-0' : 'px-3';
    };

    getTheadTrProps = () => {
        return {
            className: this.getHorizontalPaddingClass(),
        };
    };

    getTbodyProps = () => {
        return {
            className: this.getHorizontalPaddingClass(),
        };
    };

    getColumnClassName = (column) => column.className || defaultColumnClassName;

    getHeaderClassName = (column) => column.headerClassName || defaultHeaderClassName;

    render() {
        const { rows, columns, defaultSorted, manual, pageSize, ...rest } = this.props;
        if (!columns || !columns.length) {
            return null;
        }
        columns.forEach((column) =>
            Object.assign(column, {
                className: this.getColumnClassName(column),
                headerClassName: this.getHeaderClassName(column),
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
                getTheadTrProps={this.getTheadTrProps}
                getTbodyProps={this.getTbodyProps}
                defaultPageSize={pageSize}
                defaultSorted={defaultSorted}
                className={`flex flex-1 overflow-auto border-0 w-full h-full z-0 ${
                    rest.expanded ? 'expanded' : ''
                } `}
                resizable={false}
                sortable
                defaultSortDesc={false}
                showPageJump={false}
                minRows={Math.min(this.props.rows.length, pageSize)}
                page={this.props.page}
                pageSize={pageSize}
                showPagination={false}
                manual={manual}
                {...rest}
            />
        );
    }
}

export default Table;
