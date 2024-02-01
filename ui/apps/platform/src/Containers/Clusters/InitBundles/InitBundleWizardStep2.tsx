import React, { ReactElement } from 'react';
import { Alert, Flex, Title } from '@patternfly/react-core';

import { InitBundleWizardFormikProps } from './InitBundleWizard.utils';
import SecureClusterUsingHelmChart from './SecureClusterUsingHelmChart';
import SecureClusterUsingOperator from './SecureClusterUsingOperator';

const headingLevel = 'h3';

export type InitBundleWizardStep2Props = {
    errorMessage: string;
    formik: InitBundleWizardFormikProps;
};

function InitBundleWizardStep2({ errorMessage, formik }: InitBundleWizardStep2Props): ReactElement {
    const { values } = formik;
    const { installation } = values;

    /* eslint-disable no-nested-ternary */
    return (
        <Flex direction={{ default: 'column' }}>
            <Title headingLevel="h2">Download bundle</Title>
            {installation === 'Operator' ? (
                <SecureClusterUsingOperator headingLevel={headingLevel} />
            ) : (
                <SecureClusterUsingHelmChart headingLevel={headingLevel} />
            )}
            <Alert
                variant="info"
                isInline
                title="You can download the YAML file only once when you create a cluster init bundle."
                component="p"
            >
                Store the YAML file securely because it contains secrets.
            </Alert>
            {errorMessage && (
                <Alert
                    variant="danger"
                    isInline
                    title="Unable to create or download bundle"
                    component="p"
                >
                    {errorMessage}
                </Alert>
            )}
        </Flex>
    );
    /* eslint-enable no-nested-ternary */
}

export default InitBundleWizardStep2;
