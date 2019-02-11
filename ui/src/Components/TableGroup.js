import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Table from 'Components/Table';
import Collapsible from 'react-collapsible';
import * as Icon from 'react-feather';
import pluralize from 'pluralize';

const icons = {
    opened: <Icon.ChevronUp size="14" />,
    closed: <Icon.ChevronDown size="14" />
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
        <div className="flex justify-between cursor-pointer w-full py-1">
            <div className="flex w-full justify-between">
                <div className="flex items-center">
                    <div className="flex ml-4 mr-3 rounded-full bg-base-100 h-5 w-5 justify-center text-success-700 items-center border border-success-400">
                        {icons[state]}
                    </div>
                    <h1 className="p-3 pl-0 font-600 text-lg leading-normal">{name}</h1>
                </div>
                <div className="flex items-center flex-no-shrink italic font-700 text-sm p-3 pr-4 opacity-50">{`${
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
                        triggerClassName="table-group block bg-base-100 hover:bg-success-200 hover:text-success-800 z-10 relative hover:z-20"
                        triggerOpenedClassName="table-group-active bg-success-300 text-success-900 block z-30 pin-t sticky"
                        trigger={this.renderWhenClosed(group)}
                        triggerWhenOpen={this.renderWhenOpened(group)}
                        transitionTime={0.001}
                        contentOuterClassName="before before:absolute before:bg-success-300 before:h-full before:pin-l before:w-2 before:z-10 px-1 relative"
                    >
                        {this.renderSubTable(group)}
                    </Collapsible>
                ))}
            </div>
        );
    }
}

export default TableGroup;
