import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';

import { selectors } from 'reducers';

import IntegrationsSection from './IntegrationsSection';
import IntegrationTile from './IntegrationTile';
import {
    getIntegrationsListPath,
    signatureIntegrationDescriptor as descriptor,
    signatureIntegrationsSource as source,
} from '../utils/integrationsList';

const { image, label, type } = descriptor;

function SignatureIntegrationsSection(): ReactElement {
    const integrations = useSelector(selectors.getSignatureIntegrations);

    return (
        <IntegrationsSection headerName="Signature Integrations" id="signature-integrations">
            <IntegrationTile
                image={image}
                label={label}
                linkTo={getIntegrationsListPath(source, type)}
                numIntegrations={integrations.length}
            />
        </IntegrationsSection>
    );
}

export default SignatureIntegrationsSection;
