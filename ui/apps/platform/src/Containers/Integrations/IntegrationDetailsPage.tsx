import type { ReactElement } from 'react';
import { useParams } from 'react-router-dom-v5-compat';

import useIntegrations from './hooks/useIntegrations';
import type { IntegrationSource, IntegrationType } from './utils/integrationUtils';

import IntegrationPage from './IntegrationPage';
import IntegrationForm from './IntegrationForm';
import IntegrationsNotFoundPage from './IntegrationsNotFoundPage';

export type IntegrationDetailsPageProps = {
    source: IntegrationSource;
    type: IntegrationType;
};

function IntegrationDetailsPage({ source, type }: IntegrationDetailsPageProps): ReactElement {
    const { id } = useParams();
    const integrations = useIntegrations({ source, type });
    const integration = integrations.find((datum) => datum.id === id);

    if (!integration) {
        return <IntegrationsNotFoundPage />;
    }

    return (
        <IntegrationPage
            title={integration.name}
            name={integration.name}
            traits={integration.traits}
        >
            <IntegrationForm source={source} type={type} initialValues={integration} />
        </IntegrationPage>
    );
}

export default IntegrationDetailsPage;
