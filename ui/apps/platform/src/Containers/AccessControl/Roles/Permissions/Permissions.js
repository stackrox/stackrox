import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';

import { defaultRoles } from 'constants/accessControl';
import Panel, { headerClassName } from 'Components/Panel';
import Button from 'Containers/AccessControl/Roles/Permissions/Button';
import Form from 'Containers/AccessControl/Roles/Permissions/Form';
import Details from 'Containers/AccessControl/Roles/Permissions/Details';
import addDefaultPermissionsToRole from 'Containers/AccessControl/Roles/Permissions/addDefaultPermissionsToRole';

const Permissions = ({
    resources,
    selectedRole,
    isEditing,
    onSave,
    onEdit,
    onCancel,
    readOnly,
}) => {
    function displayContent() {
        const modifiedSelectedRole = addDefaultPermissionsToRole(resources, selectedRole);
        const content = isEditing ? (
            <Form onSubmit={onSave} initialValues={modifiedSelectedRole} />
        ) : (
            <Details role={modifiedSelectedRole} />
        );
        return content;
    }
    if (!selectedRole) return null;
    let headerText = 'Create New Role';
    const { name, username } = selectedRole;
    if (name || username) {
        headerText = name ? `"${name}" Permissions` : 'User Permissions';
    }
    const headerComponents = defaultRoles[selectedRole.name] ? (
        <span className="uppercase text-base-500 leading-normal font-700">system default</span>
    ) : (
        <>
            {!readOnly && (
                <Button isEditing={isEditing} onEdit={onEdit} onSave={onSave} onCancel={onCancel} />
            )}
        </>
    );
    const panelHeaderClassName = `${headerClassName} bg-base-100`;
    return (
        <Panel
            header={headerText}
            className="border"
            headerClassName={panelHeaderClassName}
            headerComponents={headerComponents}
        >
            <div className="w-full h-full bg-base-100 flex flex-1">{displayContent()}</div>
        </Panel>
    );
};

Permissions.propTypes = {
    resources: PropTypes.arrayOf(PropTypes.string).isRequired,
    selectedRole: PropTypes.shape({
        name: PropTypes.string,
        globalAccess: PropTypes.string,
        resourceToAccess: PropTypes.shape({}),
        username: PropTypes.string,
    }),
    isEditing: PropTypes.bool,
    onSave: PropTypes.func,
    onCancel: PropTypes.func,
    onEdit: PropTypes.func,
    readOnly: PropTypes.bool,
};

Permissions.defaultProps = {
    isEditing: false,
    selectedRole: null,
    onSave: null,
    onCancel: null,
    onEdit: null,
    readOnly: false,
};

const getSelectedRole = (state, ownProps) => {
    return ownProps.selectedRole || selectors.getSelectedRole;
};

const mapStateToProps = createStructuredSelector({
    resources: selectors.getResources,
    selectedRole: getSelectedRole,
});

const mapDispatchToProps = {};

export default connect(mapStateToProps, mapDispatchToProps)(Permissions);
