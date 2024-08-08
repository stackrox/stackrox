import React, { ReactElement } from 'react';
import { PageSection, Title } from '@patternfly/react-core';

import useCentralCapabilities from 'hooks/useCentralCapabilities';

import AuthenticationTokensSection from './AuthenticationTokensSection';
import BackupIntegrationsSection from './BackupIntegrationsSection';
import ImageIntegrationsSection from './ImageIntegrationsSection';
import NotifierIntegrationsSection from './NotifierIntegrationsSection';
import SignatureIntegrationsSection from './SignatureIntegrationsSection';
import CloudSourceIntegrationsSection from './CloudSourceIntegrationsSection';
import OcmDeprecatedTokenBanner from '../Banners/OcmDeprecatedToken';

function IntegrationTilesPage(): ReactElement {
    const { isCentralCapabilityAvailable } = useCentralCapabilities();
    const canUseCloudBackupIntegrations = isCentralCapabilityAvailable(
        'centralCanUseCloudBackupIntegrations'
    );

    return (
        <>
            {/*TODO(ROX-25633): Remove the banner again.*/}
            <OcmDeprecatedTokenBanner />
            <PageSection variant="light" component="div">
                <Title headingLevel="h1">Integrations</Title>
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
