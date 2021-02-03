import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import PageHeader from 'Components/PageHeader';

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

const emptyTile = {
    source: '',
    type: '',
    label: '',
};

const IntegrationsPage = ({
    apiTokens,
    clusterInitBundles,
    authProviders,
    authPlugins,
    backups,
    imageIntegrations,
    notifiers,
    logConsumptions,
    fetchAuthPlugins,
    fetchAPITokens,
    fetchClusterInitBundles,
    fetchAuthProviders,
    fetchImageIntegrations,
    fetchNotifiers,
    fetchBackups,
    featureFlags,
}) => {
    const [modalOpen, setModalOpen] = useState(false);
    const [selectedTile, setSelectedTile] = useState(emptyTile);
    const isHostScanningEnabled = useFeatureFlagEnabled(knownBackendFlags.ROX_HOST_SCANNING);
    const isK8sAuditLoggingEnabled = useFeatureFlagEnabled(
        knownBackendFlags.ROX_K8S_EVENTS_DETECTION
    );

    function getSelectedEntities() {
        const { source, type } = selectedTile;
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
            default:
                throw new Error(`Unknown source ${source}`);
        }
    }

    function openIntegrationModal(selectedIntegrationTile) {
        setModalOpen(true);
        setSelectedTile(selectedIntegrationTile);
    }

    function fetchEntitiesAndCloseModal() {
        getSelectedEntities();
        setModalOpen(false);
        setSelectedTile(emptyTile);
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
            case 'logConsumption': {
                return logConsumptions;
            }
            default: {
                throw new Error(`Unknown source ${source}`);
            }
        }
    }

    function renderIntegrationTiles(source) {
        return integrationsList[source].map((tile) => {
            if (tile.featureFlagDependency) {
                if (
                    isBackendFeatureFlagEnabled(
                        featureFlags,
                        tile.featureFlagDependency.featureFlag,
                        tile.featureFlagDependency.defaultValue
                    ) !== tile.featureFlagDependency.showIfValueIs
                ) {
                    return null;
                }
            }
            // TODO: remove this manual check after ROX_HOST_SCANNING feature flag turned on
            if (tile.label === 'StackRox Scanner' && !isHostScanningEnabled) {
                // eslint-disable-next-line no-param-reassign
                tile.categories = 'Scanner';
            }
            // end TODO block to remove

            return (
                <IntegrationTile
                    key={tile.label}
                    integration={tile}
                    onClick={openIntegrationModal}
                    numIntegrations={findIntegrations(tile.source, tile.type).length}
                />
            );
        });
    }

    const imageIntegrationTiles = renderIntegrationTiles('imageIntegrations');
    const pluginTiles = renderIntegrationTiles('plugins');
    const authPluginTiles = renderIntegrationTiles('authPlugins');
    const authProviderTiles = renderIntegrationTiles('authProviders');
    const backupTiles = renderIntegrationTiles('backups');
    const logConsumptionTiles = renderIntegrationTiles('logConsumption');

    return (
        <div className="h-full flex flex-col md:w-full bg-base-200" id="integrationsPage">
            <div className="flex flex-shrink-0">
                <PageHeader header="Integrations" subHeader="Setup & Configuration" />
            </div>
            <div className="w-full h-full overflow-auto">
                <IntegrationsSection
                    headerName="Image Integrations"
                    tiles={imageIntegrationTiles}
                    testId="image-integrations"
                />
                <IntegrationsSection
                    headerName="Notifier Integrations"
                    tiles={pluginTiles}
                    testId="notifier-integrations"
                />
                <IntegrationsSection
                    headerName="Backup Integrations"
                    tiles={backupTiles}
                    testId="backup-integrations"
                />
                <IntegrationsSection
                    headerName="Authentication Tokens"
                    tiles={authProviderTiles}
                    testId="token-integrations"
                />
                <IntegrationsSection
                    headerName="Authorization Plugins"
                    tiles={authPluginTiles}
                    testId="auth-integrations"
                />
                {isK8sAuditLoggingEnabled && (
                    <IntegrationsSection
                        headerName="Log Consumption"
                        tiles={logConsumptionTiles}
                        testId="log-integrations"
                    />
                )}
            </div>
            {modalOpen && (
                <GenericIntegrationModal
                    apiTokens={apiTokens}
                    clusterInitBundles={clusterInitBundles}
                    fetchEntitiesAndCloseModal={fetchEntitiesAndCloseModal}
                    findIntegrations={findIntegrations}
                    selectedTile={selectedTile}
                />
            )}
        </div>
    );
};

IntegrationsPage.propTypes = {
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
    logConsumptions: PropTypes.arrayOf(PropTypes.object),
    imageIntegrations: PropTypes.arrayOf(PropTypes.object).isRequired,
    fetchAuthPlugins: PropTypes.func.isRequired,
    fetchAuthProviders: PropTypes.func.isRequired,
    fetchAPITokens: PropTypes.func.isRequired,
    fetchBackups: PropTypes.func.isRequired,
    fetchClusterInitBundles: PropTypes.func.isRequired,
    fetchNotifiers: PropTypes.func.isRequired,
    fetchImageIntegrations: PropTypes.func.isRequired,
    featureFlags: PropTypes.arrayOf(
        PropTypes.shape({
            envVar: PropTypes.string.isRequired,
            enabled: PropTypes.bool.isRequired,
        })
    ).isRequired,
};

IntegrationsPage.defaultProps = {
    logConsumptions: [],
};

const mapStateToProps = createStructuredSelector({
    authPlugins: selectors.getAuthPlugins,
    authProviders: selectors.getAuthProviders,
    apiTokens: selectors.getAPITokens,
    clusterInitBundles: selectors.getClusterInitBundles,
    notifiers: selectors.getNotifiers,
    imageIntegrations: selectors.getImageIntegrations,
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
    fetchRegistries: () => dispatch(integrationActions.fetchRegistries.request()),
    fetchScanners: () => dispatch(integrationActions.fetchScanners.request()),
    fetchAuthPlugins: () => dispatch(integrationActions.fetchAuthPlugins.request()),
});

export default connect(mapStateToProps, mapDispatchToProps)(IntegrationsPage);
