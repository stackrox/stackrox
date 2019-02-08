import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Table from 'Components/Table';
import Collapsible from 'react-collapsible';
import * as Icon from 'react-feather';
import pluralize from 'pluralize';

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
        totalRows: PropTypes.number.isRequired,
        onRowClick: PropTypes.func.isRequired,
        tableColumns: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        idAttribute: PropTypes.string.isRequired,
        entityType: PropTypes.string.isRequired,
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
                defaultSorted={[
                    {
                        id: 'control',
                        desc: false
                    }
                ]}
            />
        );
    };

    renderGroupByCollapsible = (state, { name, rows }) => (
        <div className="flex justify-between cursor-pointer bg-base-300 border-b border-base-400 w-full">
            <div className="flex w-full justify-between">
                <div className="flex">
                    <div className="flex pl-3 p-2">{icons[state]}</div>
                    <h1 className="p-3 pb-2 pl-0 text-base-600 font-600 text-lg">{name}</h1>
                </div>
                <div className="flex items-center flex-no-shrink italic font-700 text-sm p-2">{`${
                    rows.length
                } ${pluralize(this.props.entityType, rows.length)}`}</div>
            </div>
        </div>
    );

    renderWhenOpened = group => this.renderGroupByCollapsible('opened', group);

    renderWhenClosed = group => this.renderGroupByCollapsible('closed', group);

    render() {
        const { groups, totalRows } = this.props;
        return (
            <div className="flex flex-col w-full">
                {groups.map((group, idx) => (
                    <Collapsible
                        open={idx === 0 || totalRows < 25}
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
