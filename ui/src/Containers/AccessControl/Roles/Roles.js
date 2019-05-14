import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector, createSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions } from 'reducers/roles';
import Dialog from 'Components/Dialog';

import SideBar from 'Containers/AccessControl/SideBar';
import Permissions from 'Containers/AccessControl/Roles/Permissions/Permissions';
import { defaultRoles, defaultSelectedRole } from 'constants/accessControl';

const Roles = ({ roles, selectRole, selectedRole, saveRole, deleteRole }) => {
    const [isEditing, setIsEditing] = useState(false);
    const [roleToDelete, setRoleToDelete] = useState(null);

    function onSave(data) {
        saveRole(data);
        setIsEditing(false);
    }

    function onEdit() {
        setIsEditing(true);
    }

    function onCreateNewRole() {
        selectRole(defaultSelectedRole);
        setIsEditing(true);
    }

    function onCancel() {
        setIsEditing(false);
        if (selectedRole && selectedRole.name === '') {
            selectRole(roles[0]);
        }
    }

    function onDelete(role) {
        setRoleToDelete(role);
    }

    function deleteRoleHandler() {
        const roleName = roleToDelete && roleToDelete.name;
        deleteRole(roleName);
        setIsEditing(false);
        setRoleToDelete(null);
    }

    function renderAddRoleButton() {
        return (
            <button
                className="border-2 bg-primary-200 border-primary-400 text-sm text-primary-700 hover:bg-primary-300 hover:border-primary-500 rounded-sm block px-3 py-2 uppercase"
                type="button"
                onClick={onCreateNewRole}
            >
                Add New Role
            </button>
        );
    }

    function onCancelDeleteRole() {
        setRoleToDelete(null);
    }

    const curRoleToDelete = roleToDelete && roleToDelete.name;
    const className = isEditing
        ? 'before before:absolute before:h-full before:opacity-50 before:bg-base-400 before:w-full before:z-10'
        : '';
    return (
        <section className="flex flex-1 h-full">
            <div className={`w-1/4 flex flex-col ${className}`}>
                <div className="m-4 h-full shadow-sm">
                    <SideBar
                        header="StackRox Roles"
                        rows={roles}
                        selected={selectedRole}
                        onSelectRow={selectRole}
                        addRowButton={renderAddRoleButton()}
                        onCancel={onCancel}
                        onDelete={onDelete}
                        type="role"
                    />
                </div>
            </div>
            <div className="w-3/4 my-4 mr-4 z-10">
                <Permissions
                    isEditing={isEditing}
                    selectedRole={selectedRole}
                    onSave={onSave}
                    onEdit={onEdit}
                    onCancel={onCancel}
                />
            </div>
            <Dialog
                isOpen={!!curRoleToDelete}
                text={`Deleting "${curRoleToDelete}" may cause users to lose access. Are you sure you want to delete "${curRoleToDelete}"?`}
                onConfirm={deleteRoleHandler}
                onCancel={onCancelDeleteRole}
                confirmText="Delete"
            />
        </section>
    );
};

Roles.propTypes = {
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

Roles.defaultProps = {
    selectedRole: null
};

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
