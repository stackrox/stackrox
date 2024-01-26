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
            <Alert
                variant="info"
                isInline
                title="A cluster init bundle can only be downloaded once"
                component="p"
            >
                Store this bundle securely because it contains secrets. You can use the same bundle
                to secure multiple clusters.
            </Alert>
            {installation === 'Operator' ? (
                <SecureClusterUsingOperator headingLevel={headingLevel} />
            ) : (
                <SecureClusterUsingHelmChart headingLevel={headingLevel} />
            )}
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
