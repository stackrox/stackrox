import React, { FunctionComponent, ReactElement } from 'react';

import { Integration, IntegrationSource, IntegrationType } from '../utils/integrationUtils';
import ClairifyIntegrationForm from './Forms/ClairifyIntegrationForm';

type IntegrationFormProps = {
    source: IntegrationSource;
    type: IntegrationType;
    initialValues?: Integration | null;
    isEdittable?: boolean;
};

type FormProps = {
    initialValues?: Integration | null;
    isEdittable?: boolean;
};

const ComponentFormMap = {
    imageIntegrations: {
        clairify: ClairifyIntegrationForm,
    },
} as Record<IntegrationSource, Record<IntegrationType, FunctionComponent<FormProps>>>;

function IntegrationForm({
    source,
    type,
    initialValues,
    isEdittable,
}: IntegrationFormProps): ReactElement {
    const Form: FunctionComponent<FormProps> = ComponentFormMap?.[source]?.[type];
    if (!Form) {
        throw new Error(
            `There are no integration form components for source (${source}) and type (${type})`
        );
    }
    return <Form initialValues={initialValues} isEdittable={isEdittable} />;
}

export default IntegrationForm;
