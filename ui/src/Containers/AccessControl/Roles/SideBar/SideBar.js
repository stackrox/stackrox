import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions } from 'reducers/roles';
import Tooltip from 'rc-tooltip';
import * as Icon from 'react-feather';

import { defaultRoles } from 'constants/accessControl';
import List from 'Components/List';
import Panel, { headerClassName } from 'Components/Panel';
import { defaultColumnClassName, rtTrActionsClassName } from 'Components/Table';

class SideBar extends Component {
    static propTypes = {
        header: PropTypes.string.isRequired,
        roles: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        selectedRole: PropTypes.shape({}),
        selectRole: PropTypes.func.isRequired,
        onCreateNewRole: PropTypes.func.isRequired,
        onCancel: PropTypes.func.isRequired,
        onDelete: PropTypes.func.isRequired
    };

    static defaultProps = {
        selectedRole: null
    };

    onRowSelectHandler = () => role => {
        this.props.selectRole(role);
        this.props.onCancel();
    };

    onDeleteHandler = role => e => {
        e.stopPropagation();
        this.props.onDelete(role);
    };

    renderRowActionButtons = role => {
        if (defaultRoles[role.name]) return null;
        return (
            <div className="border-2 border-base-400 bg-base-100 flex">
                <Tooltip placement="top" overlay={<div>Delete role</div>} mouseLeaveDelay={0}>
                    <button
                        type="button"
                        className="p-1 px-4 hover:bg-primary-200 text-primary-600 hover:text-primary-700"
                        onClick={this.onDeleteHandler(role)}
                    >
                        <Icon.Trash2 className="mt-1 h-4 w-4" />
                    </button>
                </Tooltip>
            </div>
        );
    };

    render() {
        const { header, roles, selectedRole, onCreateNewRole } = this.props;
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
                            rows={roles}
                            selectRow={this.onRowSelectHandler()}
                            selectedListItem={selectedRole}
                            selectedIdAttribute="name"
                        />
                    </div>
                    <div className="flex items-center justify-center p-4 border-t border-base-300">
                        <div>
                            <button
                                className="btn btn-primary"
                                type="button"
                                onClick={onCreateNewRole}
                            >
                                Add New Role
                            </button>
                        </div>
                    </div>
                </div>
            </Panel>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    roles: selectors.getRoles,
    selectedRole: selectors.getSelectedRole
});

const mapDispatchToProps = {
    selectRole: actions.selectRole
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(SideBar);
