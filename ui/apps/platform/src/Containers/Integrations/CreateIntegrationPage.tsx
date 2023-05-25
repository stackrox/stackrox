import React, { ReactElement } from 'react';

import usePageState from './hooks/usePageState';

import IntegrationPage from './IntegrationPage';
import IntegrationForm from './IntegrationForm';

function CreateIntegrationPage(): ReactElement {
    const {
        params: { source, type },
    } = usePageState();

    return (
        <IntegrationPage title="Create Integration" name="Create Integration">
            <IntegrationForm source={source} type={type} isEditable />
        </IntegrationPage>
    );
}

export default CreateIntegrationPage;
