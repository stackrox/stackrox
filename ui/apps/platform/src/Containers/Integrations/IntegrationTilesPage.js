import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { PageSection, Title } from '@patternfly/react-core';
import { useParams, useHistory } from 'react-router-dom';

import { integrationsPath } from 'routePaths';
import integrationsList from 'Containers/Integrations/integrationsList';
import IntegrationTile from 'Containers/Integrations/IntegrationTile';
import { actions as authActions } from 'reducers/auth';
import { actions as apiTokenActions } from 'reducers/apitokens';
import { actions as clusterInitBundleActions } from 'reducers/clusterInitBundles';
import { actions as integrationActions } from 'reducers/integrations';
import { selectors } from 'reducers';
import { isBackendFeatureFlagEnabled, knownBackendFlags } from 'utils/featureFlags';
import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import IntegrationsSection from './IntegrationsSection';
import GenericIntegrationModal from './GenericIntegrationModal';

const IntegrationTilesPage = ({
    apiTokens,
    clusterInitBundles,
    authProviders,
    authPlugins,
    backups,
    imageIntegrations,
    notifiers,
    logIntegrations,
    fetchAuthPlugins,
    fetchAPITokens,
    fetchClusterInitBundles,
    fetchAuthProviders,
    fetchImageIntegrations,
    fetchNotifiers,
    fetchBackups,
    fetchLogIntegrations,
    featureFlags,
}) => {
    const { source: selectedSource, type: selectedType } = useParams();
    const history = useHistory();
    const isK8sAuditLoggingEnabled = useFeatureFlagEnabled(
        knownBackendFlags.ROX_K8S_AUDIT_LOG_DETECTION
    );

    function getSelectedEntities(source, type) {
        switch (source) {
            case 'authPlugins':
                fetchAuthPlugins();
                break;
            case 'authProviders':
                if (type === 'apitoken') {
                    fetchAPITokens();
                    break;
                }
                if (type === 'clusterInitBundle') {
                    fetchClusterInitBundles();
                }
                fetchAuthProviders();
                break;
            case 'imageIntegrations':
                fetchImageIntegrations();
                break;
            case 'notifiers':
                fetchNotifiers();
                break;
            case 'backups':
                fetchBackups();
                break;
            case 'logIntegrations':
                fetchLogIntegrations();
                break;
            default:
                throw new Error(`Unknown source ${source}`);
        }
    }

    function findIntegrations(source, type) {
        const typeLowerMatches = (integration) =>
            integration.type.toLowerCase() === type.toLowerCase();

        switch (source) {
            case 'authPlugins': {
                return authPlugins;
            }
            case 'authProviders': {
                if (type === 'apitoken') {
                    return apiTokens;
                }
                if (type === 'clusterInitBundle') {
                    return clusterInitBundles;
                }
                return authProviders.filter(typeLowerMatches);
            }
            case 'notifiers': {
                return notifiers.filter(typeLowerMatches);
            }
            case 'backups': {
                return backups.filter(typeLowerMatches);
            }
            case 'imageIntegrations': {
                return imageIntegrations.filter(typeLowerMatches);
            }
            case 'logIntegrations': {
                return logIntegrations;
            }
            default: {
                throw new Error(`Unknown source ${source}`);
            }
        }
    }

    function getIntegration(source, type) {
        return integrationsList[source].find((integration) => integration.type === type);
    }

    function fetchEntitiesAndCloseModal() {
        getSelectedEntities(selectedSource, selectedType);
        history.push(integrationsPath);
    }

    function getIsIntegrationFeatureFlagEnabled(integration) {
        if (integration.featureFlagDependency) {
            return isBackendFeatureFlagEnabled(featureFlags, integration.featureFlagDependency);
        }
        return true;
    }

    function renderIntegrationTiles(source) {
        return (
            integrationsList[source]
                // filter out non-visible integrations
                .filter((integration) => {
                    const isIntegrationFeatureFlagEnabled = getIsIntegrationFeatureFlagEnabled(
                        integration
                    );
                    if (!isIntegrationFeatureFlagEnabled) {
                        return false;
                    }
                    if (source !== 'authPlugins') {
                        return true;
                    }
                    const numIntegrations = findIntegrations(integration.source, integration.type)
                        .length;
                    return numIntegrations !== 0;
                })
                // get a list of rendered integration tiles
                .map((integration) => {
                    const numIntegrations = findIntegrations(integration.source, integration.type)
                        .length;
                    const linkTo = `${integrationsPath}/${integration.source}/${integration.type}`;

                    return (
                        <IntegrationTile
                            key={integration.label}
                            integration={integration}
                            numIntegrations={numIntegrations}
                            linkTo={linkTo}
                        />
                    );
                })
        );
    }

    const imageIntegrationTiles = renderIntegrationTiles('imageIntegrations');
    const notifierTiles = renderIntegrationTiles('notifiers');
    const authPluginTiles = renderIntegrationTiles('authPlugins');
    const authProviderTiles = renderIntegrationTiles('authProviders');
    const backupTiles = renderIntegrationTiles('backups');
    const logIntegrationTiles = renderIntegrationTiles('logIntegrations');

    let modal;

    if (selectedSource && selectedType) {
        const selectedTile = getIntegration(selectedSource, selectedType);
        modal = (
            <GenericIntegrationModal
                apiTokens={apiTokens}
                clusterInitBundles={clusterInitBundles}
                fetchEntitiesAndCloseModal={fetchEntitiesAndCloseModal}
                findIntegrations={findIntegrations}
                selectedTile={selectedTile}
            />
        );
    }

    return (
        <>
            <PageSection variant="light">
                <Title headingLevel="h1">Integrations</Title>
            </PageSection>
            <PageSection>
                <IntegrationsSection headerName="Image Integrations" testId="image-integrations">
                    {imageIntegrationTiles}
                </IntegrationsSection>
                <IntegrationsSection
                    headerName="Notifier Integrations"
                    testId="notifier-integrations"
                >
                    {notifierTiles}
                </IntegrationsSection>
                <IntegrationsSection headerName="Backup Integrations" testId="backup-integrations">
                    {backupTiles}
                </IntegrationsSection>
                <IntegrationsSection headerName="Authentication Tokens" testId="token-integrations">
                    {authProviderTiles}
                </IntegrationsSection>
                {authPluginTiles.length !== 0 && (
                    <IntegrationsSection
                        headerName="Authorization Plugins"
                        testId="auth-integrations"
                    >
                        {authPluginTiles}
                    </IntegrationsSection>
                )}

                {isK8sAuditLoggingEnabled && (
                    <IntegrationsSection headerName="Log Consumption" testId="log-integrations">
                        {logIntegrationTiles}
                    </IntegrationsSection>
                )}
            </PageSection>
            {modal}
        </>
    );
};

IntegrationTilesPage.propTypes = {
    authPlugins: PropTypes.arrayOf(
        PropTypes.shape({
            endpoint: PropTypes.string.isRequired,
        })
    ).isRequired,
    authProviders: PropTypes.arrayOf(
        PropTypes.shape({
            name: PropTypes.string.isRequired,
        })
    ).isRequired,
    apiTokens: PropTypes.arrayOf(
        PropTypes.shape({
            name: PropTypes.string.isRequired,
            role: PropTypes.string.isRequired,
        })
    ).isRequired,
    clusterInitBundles: PropTypes.arrayOf(
        PropTypes.shape({
            name: PropTypes.string.isRequired,
        })
    ).isRequired,
    backups: PropTypes.arrayOf(
        PropTypes.shape({
            name: PropTypes.string.isRequired,
        })
    ).isRequired,
    notifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
    logIntegrations: PropTypes.arrayOf(PropTypes.object).isRequired,
    imageIntegrations: PropTypes.arrayOf(PropTypes.object).isRequired,
    fetchAuthPlugins: PropTypes.func.isRequired,
    fetchAuthProviders: PropTypes.func.isRequired,
    fetchAPITokens: PropTypes.func.isRequired,
    fetchBackups: PropTypes.func.isRequired,
    fetchClusterInitBundles: PropTypes.func.isRequired,
    fetchNotifiers: PropTypes.func.isRequired,
    fetchImageIntegrations: PropTypes.func.isRequired,
    fetchLogIntegrations: PropTypes.func.isRequired,
    featureFlags: PropTypes.arrayOf(
        PropTypes.shape({
            envVar: PropTypes.string.isRequired,
            enabled: PropTypes.bool.isRequired,
        })
    ).isRequired,
};

const mapStateToProps = createStructuredSelector({
    authPlugins: selectors.getAuthPlugins,
    authProviders: selectors.getAuthProviders,
    apiTokens: selectors.getAPITokens,
    clusterInitBundles: selectors.getClusterInitBundles,
    notifiers: selectors.getNotifiers,
    imageIntegrations: selectors.getImageIntegrations,
    logIntegrations: selectors.getLogIntegrations,
    backups: selectors.getBackups,
    featureFlags: selectors.getFeatureFlags,
});

const mapDispatchToProps = (dispatch) => ({
    fetchAuthProviders: () => dispatch(authActions.fetchAuthProviders.request()),
    fetchAPITokens: () => dispatch(apiTokenActions.fetchAPITokens.request()),
    fetchClusterInitBundles: () =>
        dispatch(clusterInitBundleActions.fetchClusterInitBundles.request()),
    fetchBackups: () => dispatch(integrationActions.fetchBackups.request()),
    fetchNotifiers: () => dispatch(integrationActions.fetchNotifiers.request()),
    fetchImageIntegrations: () => dispatch(integrationActions.fetchImageIntegrations.request()),
    fetchLogIntegrations: () => dispatch(integrationActions.fetchLogIntegrations.request()),
    fetchRegistries: () => dispatch(integrationActions.fetchRegistries.request()),
    fetchScanners: () => dispatch(integrationActions.fetchScanners.request()),
    fetchAuthPlugins: () => dispatch(integrationActions.fetchAuthPlugins.request()),
});

export default connect(mapStateToProps, mapDispatchToProps)(IntegrationTilesPage);
