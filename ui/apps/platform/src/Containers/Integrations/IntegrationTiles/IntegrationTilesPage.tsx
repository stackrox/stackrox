import React from 'react';
import type { ReactElement } from 'react';
import { Flex, PageSection, Title } from '@patternfly/react-core';

import useCentralCapabilities from 'hooks/useCentralCapabilities';

import AuthenticationTokensSection from './AuthenticationTokensSection';
import BackupIntegrationsSection from './BackupIntegrationsSection';
import ImageIntegrationsSection from './ImageIntegrationsSection';
import NotifierIntegrationsSection from './NotifierIntegrationsSection';
import SignatureIntegrationsSection from './SignatureIntegrationsSection';
import CloudSourceIntegrationsSection from './CloudSourceIntegrationsSection';
import OcmDeprecatedToken from '../Banners/OcmDeprecatedToken';

function IntegrationTilesPage(): ReactElement {
    const { isCentralCapabilityAvailable } = useCentralCapabilities();
    const canUseCloudBackupIntegrations = isCentralCapabilityAvailable(
        'centralCanUseCloudBackupIntegrations'
    );

    return (
        <>
            <PageSection variant="light" component="div">
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
                    <Title headingLevel="h1">Integrations</Title>
                    {/*TODO(ROX-25633): Remove the banner again.*/}
                    <OcmDeprecatedToken />
                </Flex>
            </PageSection>
            <PageSection component="div">
                <ImageIntegrationsSection />
                <SignatureIntegrationsSection />
                <NotifierIntegrationsSection />
                {canUseCloudBackupIntegrations && <BackupIntegrationsSection />}
                <CloudSourceIntegrationsSection />
                <AuthenticationTokensSection />
            </PageSection>
        </>
    );
}

export default IntegrationTilesPage;
