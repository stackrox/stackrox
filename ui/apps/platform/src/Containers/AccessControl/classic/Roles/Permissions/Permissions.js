import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';

import { defaultRoles } from 'constants/accessControl';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';

import Button from './Button';
import Form from './Form';
import Details from './Details';
import addDefaultPermissionsToRole from './addDefaultPermissionsToRole';

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
    if (!selectedRole) {
        return null;
    }
    let headerText = 'Create New Role';
    const { name, username } = selectedRole;
    if (name || username) {
        headerText = name ? `"${name}" Permissions` : 'User Permissions';
    }
    const headerComponents = defaultRoles[selectedRole.name] ? (
        <span className="uppercase text-base-500 leading-normal mr-3 font-700">system default</span>
    ) : (
        <>
            {!readOnly && (
                <Button isEditing={isEditing} onEdit={onEdit} onSave={onSave} onCancel={onCancel} />
            )}
        </>
    );

    return (
        <PanelNew testid="panel">
            <PanelHead>
                <PanelTitle isUpperCase testid="panel-header" text={headerText} />
                <PanelHeadEnd>{headerComponents}</PanelHeadEnd>
            </PanelHead>
            <PanelBody>{displayContent()}</PanelBody>
        </PanelNew>
    );
};

Permissions.propTypes = {
    resources: PropTypes.arrayOf(PropTypes.string).isRequired,
    selectedRole: PropTypes.shape({
        name: PropTypes.string,
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
