import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector, createSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions } from 'reducers/roles';
import Dialog from 'Components/Dialog';

import SideBar from 'Containers/AccessControl/SideBar';
import Permissions from 'Containers/AccessControl/Roles/Permissions/Permissions';
import { defaultRoles, defaultSelectedRole } from 'constants/accessControl';

class Roles extends Component {
    static propTypes = {
        roles: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string,
                globalAccess: PropTypes.string
            })
        ).isRequired,
        selectedRole: PropTypes.shape({
            name: PropTypes.string,
            globalAccess: PropTypes.string
        }),
        selectRole: PropTypes.func.isRequired,
        saveRole: PropTypes.func.isRequired,
        deleteRole: PropTypes.func.isRequired
    };

    static defaultProps = {
        selectedRole: null
    };

    constructor(props) {
        super(props);
        this.state = {
            isEditing: false
        };
    }

    onSave = data => {
        this.props.saveRole(data);
        this.setState({ isEditing: false });
    };

    onEdit = () => {
        this.setState({ isEditing: true });
    };

    onCreateNewRole = () => {
        this.props.selectRole(defaultSelectedRole);
        this.setState({ isEditing: true });
    };

    onCancel = () => {
        this.setState({ isEditing: false });
        const { selectedRole, selectRole, roles } = this.props;
        if (selectedRole && selectedRole.name === '') {
            selectRole(roles[0]);
        }
    };

    onDelete = role => {
        this.setState({
            roleToDelete: role
        });
    };

    deleteRole = () => {
        const roleName = this.state.roleToDelete && this.state.roleToDelete.name;
        this.props.deleteRole(roleName);
        this.setState({
            isEditing: false,
            roleToDelete: null
        });
    };

    renderAddRoleButton = () => (
        <button
            className="border-2 bg-primary-200 border-primary-400 text-sm text-primary-700 hover:bg-primary-300 hover:border-primary-500 rounded-sm block px-3 py-2 uppercase"
            type="button"
            onClick={this.onCreateNewRole}
        >
            Add New Role
        </button>
    );

    onCancelDeleteRole = () => {
        this.setState({
            roleToDelete: null
        });
    };

    renderSideBar = () => {
        const header = 'StackRox Roles';
        const { roles, selectedRole, selectRole } = this.props;
        return (
            <SideBar
                header={header}
                rows={roles}
                selected={selectedRole}
                onSelectRow={selectRole}
                addRowButton={this.renderAddRoleButton()}
                onCancel={this.onCancel}
                onDelete={this.onDelete}
                type="role"
            />
        );
    };

    render() {
        const { selectedRole } = this.props;
        const roleToDelete = this.state.roleToDelete && this.state.roleToDelete.name;
        const className = this.state.isEditing
            ? 'before before:absolute before:h-full before:opacity-50 before:bg-secondary-900 before:w-full before:z-10'
            : '';
        return (
            <section className="flex flex-1 h-full">
                <div className={`w-1/4 flex flex-col ${className}`}>
                    <div className="m-4 h-full shadow-sm">{this.renderSideBar()}</div>
                </div>
                <div className="w-3/4 my-4 mr-4 z-10">
                    <Permissions
                        isEditing={this.state.isEditing}
                        selectedRole={selectedRole}
                        onSave={this.onSave}
                        onEdit={this.onEdit}
                        onCancel={this.onCancel}
                    />
                </div>
                <Dialog
                    isOpen={!!this.state.roleToDelete}
                    text={`Deleting "${roleToDelete}" may cause users to lose access. Are you sure you want to delete "${roleToDelete}"?`}
                    onConfirm={this.deleteRole}
                    onCancel={this.onCancelDeleteRole}
                    confirmText="Delete"
                />
            </section>
        );
    }
}

const getRolesWithDefault = createSelector(
    [selectors.getRoles],
    roles => roles.map(role => Object.assign({}, role, { noAction: defaultRoles[role.name] }))
);

const mapStateToProps = createStructuredSelector({
    roles: getRolesWithDefault,
    selectedRole: selectors.getSelectedRole
});

const mapDispatchToProps = {
    selectRole: actions.selectRole,
    saveRole: actions.saveRole,
    deleteRole: actions.deleteRole
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Roles);
