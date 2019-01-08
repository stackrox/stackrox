import React, { Component } from 'react';
import PropTypes from 'prop-types';
import set from 'lodash/set';

import NoResultsMessage from 'Components/NoResultsMessage';
import Panel, { headerClassName } from 'Components/Panel';
import Button from 'Containers/AccessControl/AuthProviders/AuthProvider/Button';
import Form from 'Containers/AccessControl/AuthProviders/AuthProvider/Form/Form';
import Details from 'Containers/AccessControl/AuthProviders/AuthProvider/Details';
import formDescriptor from './Form/formDescriptor';

class AuthProvider extends Component {
    static propTypes = {
        selectedAuthProvider: PropTypes.shape({}),
        isEditing: PropTypes.bool.isRequired,
        onSave: PropTypes.func.isRequired,
        onEdit: PropTypes.func.isRequired,
        onCancel: PropTypes.func.isRequired,
        groups: PropTypes.arrayOf(PropTypes.shape({})).isRequired
    };

    static defaultProps = {
        selectedAuthProvider: null
    };

    populateDefaultValues = initialValues => {
        const newInitialValues = { ...initialValues };
        newInitialValues.uiEndpoint = window.location.host;
        newInitialValues.enabled = true;
        // set initial values for default values from formDiscriptor
        if (formDescriptor[initialValues.type]) {
            formDescriptor[initialValues.type]
                .filter(field => field.default)
                .forEach(field => {
                    set(newInitialValues, field.jsonPath, field.default);
                });
        }
        return newInitialValues;
    };

    getGroupsByAuthProviderId = (groups, id) => {
        const filteredGroups = groups.filter(
            group =>
                group.props &&
                group.props.authProviderId &&
                group.props.authProviderId === id &&
                (group.props.key !== '' && group.props.value !== '')
        );
        return filteredGroups;
    };

    getDefaultRoleByAuthProviderId = (groups, id) => {
        let defaultRoleGroups = groups.filter(
            group =>
                group.props &&
                group.props.authProviderId &&
                group.props.authProviderId === id &&
                group.props.key === '' &&
                group.props.value === ''
        );
        if (defaultRoleGroups.length) {
            return defaultRoleGroups[0].roleName;
        }
        // if there is no default role specified for this auth provider then use the global default role
        defaultRoleGroups = groups.filter(group => !group.props);
        if (defaultRoleGroups.length) return defaultRoleGroups[0].roleName;
        return 'Admin';
    };

    displayEmptyState = () => (
        <NoResultsMessage message="No Auth Providers integrated. Please add one." />
    );

    displayContent = () => {
        const { selectedAuthProvider, isEditing, onSave, groups } = this.props;
        let initialValues = { ...selectedAuthProvider };
        if (!selectedAuthProvider.name) {
            initialValues = this.populateDefaultValues(initialValues);
        }
        const filteredGroups = this.getGroupsByAuthProviderId(groups, selectedAuthProvider.id);
        const defaultRole = this.getDefaultRoleByAuthProviderId(groups, selectedAuthProvider.id);

        const modifiedInitialValues = Object.assign(initialValues, {
            groups: filteredGroups,
            defaultRole
        });
        const content = isEditing ? (
            <Form
                key={initialValues.type}
                onSubmit={onSave}
                initialValues={modifiedInitialValues}
                selectedAuthProvider={selectedAuthProvider}
            />
        ) : (
            <Details
                authProvider={selectedAuthProvider}
                groups={filteredGroups}
                defaultRole={defaultRole}
            />
        );
        return content;
    };

    render() {
        const { selectedAuthProvider, isEditing, onSave, onEdit, onCancel } = this.props;
        const isEmptyState = !selectedAuthProvider;
        let headerText = '';
        let headerComponents = null;
        if (isEmptyState) {
            headerText = 'Getting Started';
        } else {
            headerText = selectedAuthProvider.name
                ? `"${selectedAuthProvider.name}" Auth Provider`
                : `Create New Auth ${selectedAuthProvider.type} Provider`;
            headerComponents = (
                <Button isEditing={isEditing} onEdit={onEdit} onSave={onSave} onCancel={onCancel} />
            );
        }
        const panelHeaderClassName = `${headerClassName} bg-base-100`;
        return (
            <Panel
                header={headerText}
                headerClassName={panelHeaderClassName}
                headerComponents={headerComponents}
            >
                <div className="w-full h-full bg-base-100 flex flex-1 p-4">
                    {isEmptyState ? this.displayEmptyState() : this.displayContent()}
                </div>
            </Panel>
        );
    }
}

export default AuthProvider;
