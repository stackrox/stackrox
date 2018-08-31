import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTablePropTypes from 'react-table/lib/propTypes';
import Table from 'Components/Table';

class CheckboxTable extends Component {
    static propTypes = {
        columns: ReactTablePropTypes.columns.isRequired,
        rows: PropTypes.arrayOf(PropTypes.object).isRequired,
        onRowClick: PropTypes.func,
        selectedRowId: PropTypes.string
    };

    static defaultProps = {
        selectedRowId: null,
        onRowClick: null
    };

    constructor(props) {
        super(props);

        this.state = { selection: [] };
    }

    toggleRow = ({ id }) => e => {
        e.stopPropagation();
        const selection = [...this.state.selection];
        const keyIndex = selection.indexOf(id);
        // check to see if the key exists
        if (keyIndex >= 0) selection.splice(keyIndex, 1);
        else selection.push(id);
        // update the state
        this.setState({ selection });
    };

    toggleSelectAll = () => () => {
        const selectedAll = this.allSelected();
        let selection = [];
        // we need to get at the internals of ReactTable, passed through by ref
        const wrappedInstance = this.table.reactTable;
        // the 'sortedData' property contains the currently accessible records based on the filter and sort
        const { sortedData, page, pageSize } = wrappedInstance.getResolvedState();
        const startIndex = page * pageSize;
        const nextPageIndex = (page + 1) * pageSize;

        if (!selectedAll) {
            selection = [...this.state.selection];
            let previouslySelected = 0;
            // we just push all the IDs onto the selection array of the currently selected page
            for (let i = startIndex; i < nextPageIndex; i += 1) {
                if (!sortedData[i]) break;
                const { id } = sortedData[i].checkbox;
                const keyIndex = selection.indexOf(id);
                // if already selected, don't add again, else add to the selection
                if (keyIndex >= 0) previouslySelected += 1;
                else selection.push(id);
            }
            // if all were previously selected on the current page, unselect all on page
            if (
                previouslySelected === pageSize ||
                previouslySelected === sortedData.length % pageSize
            ) {
                for (let i = startIndex; i < nextPageIndex; i += 1) {
                    if (!sortedData[i]) break;
                    const { id } = sortedData[i].checkbox;
                    const keyIndex = selection.indexOf(id);
                    selection.splice(keyIndex, 1);
                }
            }
        }
        this.setState({ selection });
    };

    clearSelectedRows = () => this.setState({ selection: [] });

    someSelected = () =>
        this.state.selection.length !== 0 && this.state.selection.length < this.props.rows.length;

    allSelected = () =>
        this.state.selection.length !== 0 && this.state.selection.length === this.props.rows.length;

    addCheckboxColumns = () => {
        const { columns } = this.props;
        return [
            {
                id: 'checkbox',
                accessor: '',
                Cell: ({ original }) => (
                    <input
                        type="checkbox"
                        checked={this.state.selection.includes(original.id)}
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
        return <Table {...rest} ref={r => (this.table = r)} columns={columns} />; // eslint-disable-line
    }
}

export default CheckboxTable;
