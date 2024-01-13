import React, { ReactElement } from 'react';
import { Alert, Flex, Title } from '@patternfly/react-core';

import { InitBundleWizardFormikProps } from './InitBundleWizard.utils';

export type InitBundleWizardStep2Props = {
    errorMessage: string;
    formik: InitBundleWizardFormikProps;
};

function InitBundleWizardStep2({ errorMessage, formik }: InitBundleWizardStep2Props): ReactElement {
    const { values } = formik;

    /* eslint-disable no-nested-ternary */
    return (
        <Flex direction={{ default: 'column' }}>
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                <Title headingLevel="h2">Download bundle</Title>
                <p>
                    {values.installation === 'Operator'
                        ? 'Use this bundle to install secured cluster services on OpenShift with an Operator.'
                        : values.platform === 'OpenShift'
                          ? 'Use this bundle to install secured cluster services on OpenShift with a Helm chart.'
                          : 'Use this bundle to install secured cluster services on xKS with a Helm chart.'}
                </p>
            </Flex>
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
