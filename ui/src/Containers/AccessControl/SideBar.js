import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Tooltip from 'rc-tooltip';
import * as Icon from 'react-feather';

import List from 'Components/List';
import Panel, { headerClassName } from 'Components/Panel';
import { defaultColumnClassName, rtTrActionsClassName } from 'Components/Table';

class SideBar extends Component {
    static propTypes = {
        header: PropTypes.string.isRequired,
        rows: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        selected: PropTypes.shape({}),
        onSelectRow: PropTypes.func.isRequired,
        addRowButton: PropTypes.node.isRequired,
        onCancel: PropTypes.func.isRequired,
        onDelete: PropTypes.func.isRequired,
        type: PropTypes.string.isRequired
    };

    static defaultProps = {
        selected: null
    };

    onRowSelectHandler = () => row => {
        this.props.onSelectRow(row);
        this.props.onCancel();
    };

    onDeleteHandler = row => e => {
        e.stopPropagation();
        this.props.onDelete(row);
        this.props.onSelectRow();
    };

    renderRowActionButtons = row => {
        if (row.noAction) return null;
        const { type } = this.props;
        return (
            <div className="border-2 border-base-400 bg-base-100 flex">
                <Tooltip placement="top" overlay={<div>Delete {type}</div>} mouseLeaveDelay={0}>
                    <button
                        type="button"
                        className="p-1 px-4 hover:bg-primary-200 text-primary-600 hover:text-primary-700"
                        onClick={this.onDeleteHandler(row)}
                    >
                        <Icon.Trash2 className="mt-1 h-4 w-4" />
                    </button>
                </Tooltip>
            </div>
        );
    };

    render() {
        const { header, rows, selected, addRowButton } = this.props;
        const panelHeaderClassName = `${headerClassName} bg-base-100`;
        const columns = [
            {
                id: 'name',
                accessor: 'name',
                className: `${defaultColumnClassName}`
            },
            {
                accessor: '',
                headerClassName: 'hidden',
                className: rtTrActionsClassName,
                Cell: ({ original }) => this.renderRowActionButtons(original)
            }
        ];
        return (
            <Panel header={header} headerClassName={panelHeaderClassName}>
                <div className="flex flex-col w-full h-full bg-base-100">
                    <div className="overflow-auto">
                        <List
                            columns={columns}
                            rows={rows}
                            selectRow={this.onRowSelectHandler()}
                            selectedListItem={selected}
                            selectedIdAttribute="name"
                        />
                    </div>
                    <div className="flex items-center justify-center p-4 border-t border-base-300">
                        {addRowButton}
                    </div>
                </div>
            </Panel>
        );
    }
}

export default SideBar;
