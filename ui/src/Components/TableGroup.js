import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Table from 'Components/Table';
import Collapsible from 'react-collapsible';
import * as Icon from 'react-feather';

const icons = {
    opened: <Icon.ChevronUp className="h-5 w-5" />,
    closed: <Icon.ChevronDown className="h-5 w-5" />
};

class TableGroup extends Component {
    static propTypes = {
        groups: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string.isRequired,
                rows: PropTypes.arrayOf(PropTypes.shape())
            })
        ).isRequired,
        onRowClick: PropTypes.func.isRequired,
        tableColumns: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        idAttribute: PropTypes.string.isRequired,
        selectedRowId: PropTypes.string
    };

    static defaultProps = {
        selectedRowId: null
    };

    renderSubTable = ({ rows }) => {
        const { tableColumns, onRowClick, selectedRowId, idAttribute } = this.props;
        if (rows.length === 0) return null;
        return (
            <Table
                rows={rows}
                columns={tableColumns}
                onRowClick={onRowClick}
                selectedRowId={selectedRowId}
                idAttribute={idAttribute}
                showPagination={false}
                pageSize={rows.length}
            />
        );
    };

    renderGroupByCollapsible = (state, { name, rows }) => {
        const { idAttribute } = this.props;
        return (
            <div className="flex justify-between cursor-pointer bg-base-300 border-b border-base-400 w-full">
                <div className="flex w-full">
                    <div className="flex pl-3 p-2">{icons[state]}</div>
                    <h1 className="p-3 pb-2 pl-0 text-base-600 font-600 text-lg w-full">{name}</h1>
                    <div className="flex items-center flex-no-shrink italic font-700 text-sm p-2">{`${
                        rows.length
                    } ${idAttribute}${rows.length === 1 ? '' : 's'}`}</div>
                </div>
            </div>
        );
    };

    renderWhenOpened = group => this.renderGroupByCollapsible('opened', group);

    renderWhenClosed = group => this.renderGroupByCollapsible('closed', group);

    render() {
        const { groups } = this.props;
        return (
            <div className="flex flex-col w-full">
                {groups.map(group => (
                    <Collapsible
                        key={group.name}
                        trigger={this.renderWhenClosed(group)}
                        triggerWhenOpen={this.renderWhenOpened(group)}
                        transitionTime={100}
                    >
                        {this.renderSubTable(group)}
                    </Collapsible>
                ))}
            </div>
        );
    }
}

export default TableGroup;
