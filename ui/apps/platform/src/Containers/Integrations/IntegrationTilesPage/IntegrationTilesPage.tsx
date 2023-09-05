import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { PageSection, Title } from '@patternfly/react-core';

import useCentralCapabilities from 'hooks/useCentralCapabilities';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { integrationsPath } from 'routePaths';
import { selectors } from 'reducers';
import { ClusterInitBundle } from 'services/ClustersService';
import { ApiToken } from 'types/apiToken.proto';
import { BackupIntegration } from 'types/externalBackup.proto';
import { ImageIntegration } from 'types/imageIntegration.proto';
import { NotifierIntegration } from 'types/notifier.proto';
import { SignatureIntegration } from 'types/signatureIntegration.proto';

import integrationsList, { IntegrationDescriptor } from '../utils/integrationsList';

import IntegrationTile from './IntegrationTile';
import IntegrationsSection from './IntegrationsSection';

type IntegrationSource = keyof typeof integrationsList;

// Although unnecessary for reducers in TypeScript, we intend to replace reducers and sagas with requests.
type IntegrationsSelector = {
    apiTokens: ApiToken[];
    backups: BackupIntegration[];
    clusterInitBundles: ClusterInitBundle[];
    imageIntegrations: ImageIntegration[];
    notifiers: NotifierIntegration[];
    signatureIntegrations: SignatureIntegration[];
};

const integrationsSelector = createStructuredSelector<IntegrationsSelector>({
    apiTokens: selectors.getAPITokens,
    backups: selectors.getBackups,
    clusterInitBundles: selectors.getClusterInitBundles,
    imageIntegrations: selectors.getImageIntegrations,
    notifiers: selectors.getNotifiers,
    signatureIntegrations: selectors.getSignatureIntegrations,
});

function IntegrationTilesPage(): ReactElement {
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const { isCentralCapabilityAvailable } = useCentralCapabilities();
    const canUseCloudBackupIntegrations = isCentralCapabilityAvailable(
        'centralCanUseCloudBackupIntegrations'
    );

    const {
        apiTokens,
        clusterInitBundles,
        backups,
        imageIntegrations,
        notifiers,
        signatureIntegrations,
    } = useSelector(integrationsSelector);

    function countIntegrations(source: IntegrationSource, type): number {
        const typeLowerMatches = (integration) =>
            integration.type.toLowerCase() === type.toLowerCase();

        switch (source) {
            case 'authProviders': {
                // Integrations Authentication Tokens differ from Access Control Auth providers.
                if (type === 'apitoken') {
                    return apiTokens.length;
                }
                if (type === 'clusterInitBundle') {
                    return clusterInitBundles.length;
                }
                return 0;
            }
            case 'notifiers': {
                return notifiers.filter(typeLowerMatches).length;
            }
            case 'backups': {
                return backups.filter(typeLowerMatches).length;
            }
            case 'imageIntegrations': {
                return imageIntegrations.filter(typeLowerMatches).length;
            }
            case 'signatureIntegrations': {
                return signatureIntegrations.length;
            }
            default: {
                return 0;
            }
        }
    }

    // Maybe caused by TypeScript error below.
    /* eslint-disable @typescript-eslint/no-unsafe-return */
    function renderIntegrationTiles(source: IntegrationSource): ReactElement[] {
        return (
            integrationsList[source]
                // filter out non-visible integrations
                // TypeScript 5.2 fixes the error: This expression is not callable
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore TS2349
                .filter((integration) => {
                    if (typeof integration.featureFlagDependency === 'string') {
                        if (!isFeatureFlagEnabled(integration.featureFlagDependency)) {
                            return false;
                        }
                    }
                    return true;
                })
                // get a list of rendered integration tiles
                .map((integration: IntegrationDescriptor) => {
                    const numIntegrations = countIntegrations(integration.source, integration.type);
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
    /* eslint-enable @typescript-eslint/no-unsafe-return */

    const imageIntegrationTiles = renderIntegrationTiles('imageIntegrations');
    const notifierTiles = renderIntegrationTiles('notifiers');
    const authProviderTiles = renderIntegrationTiles('authProviders');
    const signatureTiles = renderIntegrationTiles('signatureIntegrations');

    return (
        <>
            <PageSection variant="light">
                <Title headingLevel="h1">Integrations</Title>
            </PageSection>
            <PageSection>
                <IntegrationsSection headerName="Image Integrations" id="image-integrations">
                    {imageIntegrationTiles}
                </IntegrationsSection>
                <IntegrationsSection
                    headerName="Signature Integrations"
                    id="signature-integrations"
                >
                    {signatureTiles}
                </IntegrationsSection>
                <IntegrationsSection headerName="Notifier Integrations" id="notifier-integrations">
                    {notifierTiles}
                </IntegrationsSection>
                {canUseCloudBackupIntegrations && (
                    <IntegrationsSection headerName="Backup Integrations" id="backup-integrations">
                        {renderIntegrationTiles('backups')}
                    </IntegrationsSection>
                )}
                <IntegrationsSection headerName="Authentication Tokens" id="token-integrations">
                    {authProviderTiles}
                </IntegrationsSection>
            </PageSection>
        </>
    );
}

export default IntegrationTilesPage;
