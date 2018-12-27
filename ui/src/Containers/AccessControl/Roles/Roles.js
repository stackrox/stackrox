import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector, createSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions } from 'reducers/roles';

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
    };

    onDelete = role => {
        this.props.deleteRole(role.name);
        this.setState({ isEditing: false });
    };

    renderAddRoleButton = () => (
        <button className="btn btn-primary" type="button" onClick={this.onCreateNewRole}>
            Add New Role
        </button>
    );

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
        return (
            <section className="flex flex-1 h-full">
                <div className="w-1/4 m-4">{this.renderSideBar()}</div>
                <div className="w-3/4 my-4 mr-4">
                    <Permissions
                        isEditing={this.state.isEditing}
                        selectedRole={selectedRole}
                        onSave={this.onSave}
                        onEdit={this.onEdit}
                        onCancel={this.onCancel}
                    />
                </div>
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
