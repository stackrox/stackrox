import type { ReactElement } from 'react';
import { useFormikContext } from 'formik';
import type { FormikContextType } from 'formik';
import { Flex, Form, PageSection, Title } from '@patternfly/react-core';

import NotifierConfigurationForm from 'Components/NotifierConfiguration/NotifierConfigurationForm';
import usePermissions from 'hooks/usePermissions';

import type { ScanConfigFormValues } from '../compliance.scanConfigs.utils';

function ScanConfigDeliveryStep(): ReactElement {
    const formik: FormikContextType<ScanConfigFormValues> = useFormikContext();
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForIntegration = hasReadWriteAccess('Integration');

    return (
        <PageSection>
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
                <Title headingLevel="h2">Delivery</Title>
                <Form isWidthLimited>
                    <NotifierConfigurationForm
                        errors={formik.errors}
                        fieldIdPrefixForFormikAndPatternFly="report.notifierConfigurations"
                        hasWriteAccessForIntegration={hasWriteAccessForIntegration}
                        notifierConfigurations={formik.values.report.notifierConfigurations}
                        setFieldValue={formik.setFieldValue}
                    />
                </Form>
            </Flex>
        </PageSection>
    );
}

export default ScanConfigDeliveryStep;
