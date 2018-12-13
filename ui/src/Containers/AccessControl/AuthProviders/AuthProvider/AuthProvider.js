import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import set from 'lodash/set';

import NoResultsMessage from 'Components/NoResultsMessage';
import Panel, { headerClassName } from 'Components/Panel';
import Button from 'Containers/AccessControl/AuthProviders/AuthProvider/Button/Button';
import Form from 'Containers/AccessControl/AuthProviders/AuthProvider/Form/Form';
import Details from 'Containers/AccessControl/AuthProviders/AuthProvider/Details/Details';
import formDescriptor from './Form/formDescriptor';

class AuthProvider extends Component {
    static propTypes = {
        selectedAuthProvider: PropTypes.shape({}),
        isEditing: PropTypes.bool.isRequired,
        onSave: PropTypes.func.isRequired,
        onEdit: PropTypes.func.isRequired,
        onDelete: PropTypes.func.isRequired,
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
            formDescriptor[initialValues.type].filter(field => field.default).forEach(field => {
                set(newInitialValues, field.jsonPath, field.default);
            });
        }
        return newInitialValues;
    };

    displayEmptyState = () => (
        <NoResultsMessage message="No Auth Providers integrated. Please add one." />
    );

    displayContent = () => {
        const { selectedAuthProvider, isEditing, onSave, groups, onDelete } = this.props;
        let initialValues = { ...selectedAuthProvider };
        if (!selectedAuthProvider.name) {
            initialValues = this.populateDefaultValues(initialValues);
        }
        const filteredGroups = groups.filter(
            group =>
                group.props &&
                group.props.authProviderId &&
                selectedAuthProvider.id === group.props.authProviderId
        );
        const modifiedInitialValues = Object.assign(initialValues, { groups: filteredGroups });
        const content = isEditing ? (
            <Form
                key={initialValues.type}
                onSubmit={onSave}
                initialValues={modifiedInitialValues}
                onDelete={onDelete}
            />
        ) : (
            <Details authProvider={selectedAuthProvider} groups={filteredGroups} />
        );
        return content;
    };

    render() {
        const { selectedAuthProvider, isEditing, onSave, onEdit } = this.props;
        const isEmptyState = !selectedAuthProvider;
        let headerText = '';
        let headerComponents = null;
        if (isEmptyState) {
            headerText = 'Getting Started';
        } else {
            headerText = selectedAuthProvider.name
                ? `"${selectedAuthProvider.name}" Auth Provider`
                : 'Create New Auth Provider';
            headerComponents = <Button isEditing={isEditing} onEdit={onEdit} onSave={onSave} />;
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

const mapStateToProps = createStructuredSelector({
    selectedRole: selectors.getSelectedRole
});

const mapDispatchToProps = {};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(AuthProvider);
