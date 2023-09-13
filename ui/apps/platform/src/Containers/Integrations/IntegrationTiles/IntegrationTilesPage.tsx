import React, { ReactElement } from 'react';
import { PageSection, Title } from '@patternfly/react-core';

import useCentralCapabilities from 'hooks/useCentralCapabilities';

import AuthenticationTokensSection from './AuthenticationTokensSection';
import BackupIntegrationsSection from './BackupIntegrationsSection';
import ImageIntegrationsSection from './ImageIntegrationsSection';
import NotifierIntegrationsSection from './NotifierIntegrationsSection';
import SignatureIntegrationsSection from './SignatureIntegrationsSection';

function IntegrationTilesPage(): ReactElement {
    const { isCentralCapabilityAvailable } = useCentralCapabilities();
    const canUseCloudBackupIntegrations = isCentralCapabilityAvailable(
        'centralCanUseCloudBackupIntegrations'
    );

    return (
        <>
            <PageSection variant="light" component="div">
                <Title headingLevel="h1">Integrations</Title>
            </PageSection>
            <PageSection component="div">
                <ImageIntegrationsSection />
                <SignatureIntegrationsSection />
                <NotifierIntegrationsSection />
                {canUseCloudBackupIntegrations && <BackupIntegrationsSection />}
                <AuthenticationTokensSection />
            </PageSection>
        </>
    );
}

export default IntegrationTilesPage;
