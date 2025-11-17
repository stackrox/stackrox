import type { ReactElement } from 'react';

import IntegrationPage from './IntegrationPage';
import IntegrationForm from './IntegrationForm';
import { getIsMachineAccessConfig } from './utils/integrationUtils';
import type { IntegrationSource, IntegrationType } from './utils/integrationUtils';

export type CreateIntegrationPageProps = {
    source: IntegrationSource;
    type: IntegrationType;
};

function CreateIntegrationPage({ source, type }: CreateIntegrationPageProps): ReactElement {
    const title = getIsMachineAccessConfig(source, type)
        ? 'Create configuration'
        : 'Create integration';
    return (
        <IntegrationPage title={title} name={title}>
            <IntegrationForm source={source} type={type} isEditable />
        </IntegrationPage>
    );
}

export default CreateIntegrationPage;
