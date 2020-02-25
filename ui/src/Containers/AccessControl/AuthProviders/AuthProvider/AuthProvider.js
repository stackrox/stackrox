import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { createStructuredSelector } from 'reselect';
import { connect } from 'react-redux';

import { selectors } from 'reducers';
import { isBackendFeatureFlagEnabled, knownBackendFlags } from 'utils/featureFlags';

import NoResultsMessage from 'Components/NoResultsMessage';
import Panel, { headerClassName } from 'Components/Panel';
import Button from 'Containers/AccessControl/AuthProviders/AuthProvider/Button';
import Form from 'Containers/AccessControl/AuthProviders/AuthProvider/Form/Form';
import Details from 'Containers/AccessControl/AuthProviders/AuthProvider/Details';
import { getAuthProviderLabelByValue } from 'constants/accessControl';

class AuthProvider extends Component {
    static propTypes = {
        selectedAuthProvider: PropTypes.shape({
            name: PropTypes.string,
            id: PropTypes.string,
            type: PropTypes.string,
            active: PropTypes.bool
        }),
        isEditing: PropTypes.bool.isRequired,
        onSave: PropTypes.func.isRequired,
        onEdit: PropTypes.func.isRequired,
        onCancel: PropTypes.func.isRequired,
        groups: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        featureFlags: PropTypes.shape({}).isRequired
    };

    static defaultProps = {
        selectedAuthProvider: null
    };

    populateDefaultValues = initialValues => {
        const newInitialValues = { ...initialValues };
        newInitialValues.uiEndpoint = window.location.host;
        newInitialValues.enabled = true;
        if (initialValues.type === 'oidc') {
            newInitialValues.config = { mode: 'post', do_not_use_client_secret: false };
        }
        return newInitialValues;
    };

    transformInitialValues = initialValues => {
        const ROX_REFRESH_TOKENS = isBackendFeatureFlagEnabled(
            this.props.featureFlags,
            knownBackendFlags.ROX_REFRESH_TOKENS,
            false
        );
        if (!ROX_REFRESH_TOKENS) return initialValues;

        // TODO-ivan: eventually logic for different auth provider type should live
        // with the form component that renders form for the corresponding auth provider
        // type, probably makes sense to refactor after moving away from redux-form
        if (initialValues.type === 'oidc') {
            const alteredConfig = { ...initialValues.config };

            // backend doesn't return the exact value for the client secret for the security reasons,
            // instead it'll return some obfuscated data, but not an empty one
            alteredConfig.clientOnly = {
                clientSecretStored: !!alteredConfig.client_secret
            };

            if (initialValues.name) {
                // if it's an existing auth provider, then we're using the secret if we have it
                alteredConfig.do_not_use_client_secret = !alteredConfig.client_secret;
            }

            // clean-up obfuscated value if any as we don't need to show it
            alteredConfig.client_secret = '';

            return {
                ...initialValues,
                config: alteredConfig
            };
        }
        if (initialValues.type === 'saml') {
            const alteredConfig = { ...initialValues.config };
            // unless static config values are present, assume dynamic configuration is selected
            alteredConfig.type = alteredConfig.idp_issuer ? 'static' : 'dynamic';
            return {
                ...initialValues,
                config: alteredConfig
            };
        }
        return initialValues;
    };

    transformValuesBeforeSaving = values => {
        const ROX_REFRESH_TOKENS = isBackendFeatureFlagEnabled(
            this.props.featureFlags,
            knownBackendFlags.ROX_REFRESH_TOKENS,
            false
        );
        if (!ROX_REFRESH_TOKENS) return values;

        if (values.type === 'oidc') {
            const alteredConfig = { ...values.config };

            // if client secret is stored on the backend and user didn't enter any value,
            // it means that user wants to preserve the stored secret, delete then
            const preserveStoredClientSecret =
                alteredConfig.clientOnly.clientSecretStored && !alteredConfig.client_secret;
            if (alteredConfig.do_not_use_client_secret || preserveStoredClientSecret) {
                delete alteredConfig.client_secret;
            }

            // backend expects only string values for the config
            alteredConfig.do_not_use_client_secret = alteredConfig.do_not_use_client_secret
                ? 'true'
                : 'false';

            // finally delete client only values
            delete alteredConfig.clientOnly;

            return {
                ...values,
                config: alteredConfig
            };
        }
        if (values.type === 'saml') {
            const alteredConfig = { ...values.config };
            if (alteredConfig.type === 'dynamic') {
                ['idp_issuer', 'idp_sso_url', 'idp_nameid_format', 'idp_cert_pem'].forEach(
                    p => delete alteredConfig[p]
                );
            } else if (alteredConfig.type === 'static') {
                delete alteredConfig.idp_metadata_url;
            }
            delete alteredConfig.type; // that was UI only field

            return {
                ...values,
                config: alteredConfig
            };
        }
        return values;
    };

    onSave = values => {
        const transformedValues = this.transformValuesBeforeSaving(values);
        this.props.onSave(transformedValues);
    };

    getGroupsByAuthProviderId = (groups, id) => {
        const filteredGroups = groups.filter(
            group =>
                group.props &&
                group.props.authProviderId &&
                group.props.authProviderId === id &&
                group.props.key !== ''
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
        const { selectedAuthProvider, isEditing, groups } = this.props;
        let initialValues = { ...selectedAuthProvider };
        if (!selectedAuthProvider.name) {
            initialValues = this.populateDefaultValues(initialValues);
        }
        const filteredGroups = this.getGroupsByAuthProviderId(groups, selectedAuthProvider.id);
        const defaultRole = this.getDefaultRoleByAuthProviderId(groups, selectedAuthProvider.id);

        const modifiedInitialValues = {
            ...this.transformInitialValues(initialValues),
            groups: filteredGroups,
            defaultRole
        };
        const content = isEditing ? (
            <Form
                key={initialValues.type}
                onSubmit={this.onSave}
                initialValues={modifiedInitialValues}
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
        const { selectedAuthProvider, isEditing, onEdit, onCancel } = this.props;
        const isEmptyState = !selectedAuthProvider;
        let headerText = '';
        let headerComponents = null;
        if (isEmptyState) {
            headerText = 'Getting Started';
        } else {
            headerText = selectedAuthProvider.name
                ? `"${selectedAuthProvider.name}" Auth Provider`
                : `Create New ${getAuthProviderLabelByValue(
                      selectedAuthProvider.type
                  )} Auth Provider`;
            const buttonText = selectedAuthProvider.active ? 'Edit Roles' : 'Edit Provider';
            headerComponents = (
                <Button
                    text={buttonText}
                    isEditing={isEditing}
                    onEdit={onEdit}
                    onSave={this.onSave}
                    onCancel={onCancel}
                />
            );
        }
        const panelHeaderClassName = `${headerClassName} bg-base-100`;
        return (
            <Panel
                className="border"
                header={headerText}
                headerClassName={panelHeaderClassName}
                headerComponents={headerComponents}
                id="auth-provider-panel"
            >
                <div className="w-full h-full bg-base-300 flex">
                    {isEmptyState ? this.displayEmptyState() : this.displayContent()}
                </div>
            </Panel>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    featureFlags: selectors.getFeatureFlags
});

export default connect(mapStateToProps)(AuthProvider);
