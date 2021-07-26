import React, { ReactElement } from 'react';

import usePageState from './hooks/usePageState';

import IntegrationPage from './IntegrationPage';
import IntegrationForm from './IntegrationForm';

function CreateIntegrationPage(): ReactElement {
    const {
        params: { source, type },
    } = usePageState();

    return (
        <IntegrationPage title="Create Integration">
            <IntegrationForm source={source} type={type} isEdittable />
        </IntegrationPage>
    );
}

export default CreateIntegrationPage;
