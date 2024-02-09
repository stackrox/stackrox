import React, { ReactElement } from 'react';
import IntegrationsSection from './IntegrationsSection';
import PaladinCloudTile from './PaladinCloudTile';

function CloudSourceIntegrationsSection(): ReactElement {
    return (
        <IntegrationsSection headerName="Cloud Source Integrations" id="cloud-source-integrations">
            <PaladinCloudTile />
        </IntegrationsSection>
    );
}

export default CloudSourceIntegrationsSection;
