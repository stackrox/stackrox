import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTablePropTypes from 'react-table-6/lib/propTypes';
import Table, { rtTrActionsClassName } from 'Components/Table';

class CheckboxTable extends Component {
    static propTypes = {
        columns: ReactTablePropTypes.columns.isRequired,
        rows: PropTypes.arrayOf(PropTypes.object).isRequired,
        onRowClick: PropTypes.func,
        selectedRowId: PropTypes.string,
        toggleRow: PropTypes.func.isRequired,
        toggleSelectAll: PropTypes.func.isRequired,
        selection: PropTypes.arrayOf(PropTypes.string),
        page: PropTypes.number,
        pageSize: PropTypes.number,
        renderRowActionButtons: PropTypes.func,
        manual: PropTypes.bool,
        idAttribute: PropTypes.string,
    };

    static defaultProps = {
        selectedRowId: null,
        onRowClick: null,
        selection: [],
        page: 0,
        pageSize: undefined, // Defer to the default in the child component, if this is not specified.
        renderRowActionButtons: null,
        manual: false,
        idAttribute: 'id',
    };

    setTableRef = (table) => {
        this.reactTable = table;
    };

    toggleRowHandler =
        ({ id }) =>
        () => {
            this.props.toggleRow(id);
        };

    stopPropagationOnClick = (e) => e.stopPropagation();

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
        const { columns, selection, renderRowActionButtons } = this.props;
        let checkboxColumns = [
            {
                id: 'checkbox',
                accessor: '',
                /* eslint-disable react/prop-types */
                // original.id is the first item in the rows prop which would require a custom validator
                Cell: ({ original }) => (
                    <input
                        type="checkbox"
                        data-testid="checkbox-table-row-selector"
                        checked={selection.includes(original.id)}
                        onChange={this.toggleRowHandler(original)}
                        onClick={this.stopPropagationOnClick} // don't want checkbox click to select the row
                        aria-label="Toggle row select"
                    />
                ),
                /* eslint-enable react/prop-types */
                Header: () => (
                    <input
                        type="checkbox"
                        checked={this.allSelected()}
                        ref={(input) => {
                            if (input) {
                                input.indeterminate = this.someSelected(); // eslint-disable-line no-param-reassign
                            }
                        }}
                        onChange={this.toggleSelectAllHandler}
                        aria-label="Toggle select all rows"
                    />
                ),
                sortable: false,
                width: 28,
            },
            ...columns,
        ];
        if (renderRowActionButtons) {
            checkboxColumns = [
                ...checkboxColumns,
                {
                    Header: '',
                    accessor: '',
                    headerClassName: 'hidden',
                    className: rtTrActionsClassName,
                    Cell: ({ original }) => renderRowActionButtons(original),
                },
            ];
        }
        return checkboxColumns;
    };

    render() {
        const { manual, ...rest } = this.props;
        const columns = this.addCheckboxColumns();
        return <Table {...rest} columns={columns} manual={manual} setTableRef={this.setTableRef} />;
    }
}

export default CheckboxTable;
