import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import ReactTablePropTypes from 'react-table/lib/propTypes';
import flattenObject from 'utils/flattenObject';
import { Tooltip } from 'react-tippy';
import * as Icon from 'react-feather';

export const defaultHeaderClassName =
    'px-2 py-4 pb-3 font-700 text-base-600 hover:bg-primary-200 hover:z-1 hover:text-primary-700 select-none relative text-left border-r-0 leading-normal';
export const defaultColumnClassName =
    'p-2 flex items-center font-600 text-base-600 text-left border-r-0 leading-normal';
export const wrapClassName = 'whitespace-normal overflow-visible';
export const rtTrActionsClassName =
    'rt-tr-actions hidden pin-r p-0 mr-2 w-auto text-left self-center';
export const pageSize = 50;
const headerTooltipContent = (
    <div>
        <div className="text-sm flex justify-between pb-1">
            <span>Sort Asc/Desc</span>
            <span>
                <Icon.ArrowUp className="h-2 w-2" />
                <Icon.ArrowDown className="h-2 w-2" />
            </span>
        </div>
        <div className="border-t border-base-100 text-xs pt-1">(hold shift to multi-sort)</div>
    </div>
);

class Table extends Component {
    static propTypes = {
        columns: ReactTablePropTypes.columns.isRequired,
        rows: PropTypes.arrayOf(PropTypes.object).isRequired,
        onRowClick: PropTypes.func,
        selectedRowId: PropTypes.string,
        idAttribute: PropTypes.string,
        noDataText: ReactTablePropTypes.noDataText,
        setTableRef: PropTypes.func,
        page: PropTypes.number,
        trClassName: PropTypes.string,
        showThead: PropTypes.bool
    };

    static defaultProps = {
        noDataText: 'No records.',
        selectedRowId: null,
        idAttribute: 'id',
        onRowClick: null,
        setTableRef: null,
        page: 0,
        trClassName: '',
        showThead: true
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
        return {
            onClick: () => {
                if (this.props.onRowClick) this.props.onRowClick(rowInfo.original);
            },
            className: classes.join(' ')
        };
    };

    getColumnClassName = column => column.className || defaultColumnClassName;

    getHeaderClassName = column => column.headerClassName || defaultHeaderClassName;

    renderTooltip = headerText => () => (
        <Tooltip
            useContext
            position="top"
            trigger="mouseenter"
            animation="none"
            duration={0}
            arrow
            html={headerTooltipContent}
            unmountHTMLWhenHide
        >
            <div className="table-header-text">{headerText}</div>
        </Tooltip>
    );

    render() {
        const { rows, columns, ...rest } = this.props;
        columns.forEach(column => {
            const headerText = column.Header;
            if (typeof column.Header === 'string') {
                Object.assign(column, {
                    HeaderText: headerText,
                    Header: this.renderTooltip(headerText)()
                });
            }
            Object.assign(column, {
                className: this.getColumnClassName(column),
                headerClassName: this.getHeaderClassName(column)
            });
        });
        return (
            <ReactTable
                ref={this.props.setTableRef}
                data={rows}
                columns={columns}
                getTrGroupProps={this.getTrGroupProps}
                getTrProps={this.getTrProps}
                getTheadProps={this.getTheadProps}
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
