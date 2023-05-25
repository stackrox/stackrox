import React, { ReactElement } from 'react';

import usePageState from './hooks/usePageState';
import useIntegrations from './hooks/useIntegrations';

import IntegrationPage from './IntegrationPage';
import IntegrationForm from './IntegrationForm';
import IntegrationsNotFoundPage from './IntegrationsNotFoundPage';

function EditIntegrationPage(): ReactElement {
    const {
        params: { source, type, id },
    } = usePageState();
    const integrations = useIntegrations({ source, type });
    const integration = integrations.find((datum) => datum.id === id);

    if (!integration) {
        return <IntegrationsNotFoundPage />;
    }

    return (
        <IntegrationPage title="Edit Integration" name={integration.name}>
            <IntegrationForm source={source} type={type} initialValues={integration} isEditable />
        </IntegrationPage>
    );
}

export default EditIntegrationPage;
