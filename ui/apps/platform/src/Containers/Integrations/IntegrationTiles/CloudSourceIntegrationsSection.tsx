import type { ReactElement } from 'react';
import IntegrationsSection from './IntegrationsSection';
import PaladinCloudTile from './PaladinCloudTile';
import OcmTile from './OcmTile';

function CloudSourceIntegrationsSection(): ReactElement {
    return (
        <IntegrationsSection headerName="Cloud Source Integrations" id="cloud-source-integrations">
            <PaladinCloudTile />
            <OcmTile />
        </IntegrationsSection>
    );
}

export default CloudSourceIntegrationsSection;
