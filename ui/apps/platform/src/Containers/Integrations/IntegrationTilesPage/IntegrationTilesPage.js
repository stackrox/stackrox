import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { PageSection, Title } from '@patternfly/react-core';

import useFeatureFlags from 'hooks/useFeatureFlags';
import { integrationsPath } from 'routePaths';
import { selectors } from 'reducers';
import integrationsList from '../utils/integrationsList';

import IntegrationTile from './IntegrationTile';
import IntegrationsSection from './IntegrationsSection';

const IntegrationTilesPage = ({
    apiTokens,
    clusterInitBundles,
    authProviders,
    authPlugins,
    backups,
    imageIntegrations,
    notifiers,
    signatureIntegrations,
}) => {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isSignatureIntegrationEnabled = isFeatureFlagEnabled('ROX_VERIFY_IMAGE_SIGNATURE');

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
            case 'signatureIntegrations': {
                return signatureIntegrations;
            }
            default: {
                throw new Error(`Unknown source ${source}`);
            }
        }
    }

    function renderIntegrationTiles(source) {
        return (
            integrationsList[source]
                // filter out non-visible integrations
                .filter((integration) => {
                    if (typeof integration.featureFlagDependency === 'string') {
                        if (!isFeatureFlagEnabled(integration.featureFlagDependency)) {
                            return false;
                        }
                    }
                    if (source !== 'authPlugins') {
                        return true;
                    }
                    const numIntegrations = findIntegrations(
                        integration.source,
                        integration.type
                    ).length;
                    return numIntegrations !== 0;
                })
                // get a list of rendered integration tiles
                .map((integration) => {
                    const numIntegrations = findIntegrations(
                        integration.source,
                        integration.type
                    ).length;
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
    const signatureTiles = renderIntegrationTiles('signatureIntegrations');

    return (
        <>
            <PageSection variant="light">
                <Title headingLevel="h1">Integrations</Title>
            </PageSection>
            <PageSection>
                <IntegrationsSection headerName="Image Integrations" testId="image-integrations">
                    {imageIntegrationTiles}
                </IntegrationsSection>
                {isSignatureIntegrationEnabled && (
                    <IntegrationsSection
                        headerName="Signature Integrations"
                        testId="signature-integrations"
                    >
                        {signatureTiles}
                    </IntegrationsSection>
                )}
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
            </PageSection>
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
    imageIntegrations: PropTypes.arrayOf(PropTypes.object).isRequired,
    signatureIntegrations: PropTypes.arrayOf(PropTypes.object).isRequired,
};

const mapStateToProps = createStructuredSelector({
    authPlugins: selectors.getAuthPlugins,
    authProviders: selectors.getAuthProviders,
    apiTokens: selectors.getAPITokens,
    clusterInitBundles: selectors.getClusterInitBundles,
    notifiers: selectors.getNotifiers,
    imageIntegrations: selectors.getImageIntegrations,
    backups: selectors.getBackups,
    signatureIntegrations: selectors.getSignatureIntegrations,
});

export default connect(mapStateToProps)(IntegrationTilesPage);
