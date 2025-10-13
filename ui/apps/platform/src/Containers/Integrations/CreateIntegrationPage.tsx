import React from 'react';
import type { ReactElement } from 'react';

import usePageState from './hooks/usePageState';

import IntegrationPage from './IntegrationPage';
import IntegrationForm from './IntegrationForm';
import { getIsMachineAccessConfig } from './utils/integrationUtils';

function CreateIntegrationPage(): ReactElement {
    const {
        params: { source, type },
    } = usePageState();

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
