import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';

import { defaultRoles } from 'constants/accessControl';
import Panel, { headerClassName } from 'Components/Panel';
import Button from 'Containers/AccessControl/Roles/Permissions/Button/Button';
import Form from 'Containers/AccessControl/Roles/Permissions/Form/Form';
import Details from 'Containers/AccessControl/Roles/Permissions/Details/Details';

class Permissions extends Component {
    static propTypes = {
        selectedRole: PropTypes.shape({
            name: PropTypes.string,
            globalAccess: PropTypes.string,
            resourceToAccess: PropTypes.shape({})
        }),
        isEditing: PropTypes.bool.isRequired,
        onSave: PropTypes.func.isRequired,
        onEdit: PropTypes.func.isRequired
    };

    static defaultProps = {
        selectedRole: null
    };

    displayContent = () => {
        const { selectedRole, isEditing, onSave } = this.props;
        const content = isEditing ? (
            <Form onSubmit={onSave} initialValues={selectedRole} />
        ) : (
            <Details role={selectedRole} />
        );
        return content;
    };

    render() {
        const { selectedRole, isEditing, onSave, onEdit } = this.props;
        if (!selectedRole) return null;
        const headerText = selectedRole.name
            ? `"${selectedRole.name}" Permissions`
            : 'Create New Role';
        const headerComponents = defaultRoles[selectedRole.name] ? null : (
            <Button isEditing={isEditing} onEdit={onEdit} onSave={onSave} />
        );
        const panelHeaderClassName = `${headerClassName} bg-base-100`;
        return (
            <Panel
                header={headerText}
                headerClassName={panelHeaderClassName}
                headerComponents={headerComponents}
            >
                <div className="w-full h-full bg-base-100 flex flex-1 p-4">
                    {this.displayContent()}
                </div>
            </Panel>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    selectedRole: selectors.getSelectedRole
});

const mapDispatchToProps = {};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Permissions);
