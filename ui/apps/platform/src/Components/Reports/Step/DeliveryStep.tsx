import type { ReactElement } from 'react';
import { Flex, Form, PageSection, Title } from '@patternfly/react-core';
import type { FormikProps } from 'formik';

import NotifierConfigurationForm from 'Components/NotifierConfiguration/NotifierConfigurationForm';
import usePermissions from 'hooks/usePermissions';

import type { DeliveryType } from '../reports.types';

import ScheduleFormSection from './ScheduleFormSection';

export type DeliveryStepProps<T extends DeliveryType = DeliveryType> = {
    formik: FormikProps<T>;
};

function DeliveryStep<T extends DeliveryType = DeliveryType>({
    formik,
}: DeliveryStepProps<T>): ReactElement {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForIntegration = hasReadWriteAccess('Integration');

    function onDeleteLastNotifierConfiguration() {
        formik.setFieldValue('schedule', null);
    }

    return (
        <PageSection>
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
                <Title headingLevel="h2">Delivery</Title>
                <Form isHorizontal isWidthLimited>
                    <NotifierConfigurationForm
                        errors={formik.errors}
                        fieldIdPrefixForFormikAndPatternFly="notifiers"
                        hasWriteAccessForIntegration={hasWriteAccessForIntegration}
                        notifierConfigurations={formik.values.notifiers}
                        onDeleteLastNotifierConfiguration={onDeleteLastNotifierConfiguration}
                        setFieldValue={formik.setFieldValue}
                    />
                    {formik.values.notifiers.length !== 0 && (
                        <ScheduleFormSection formik={formik} />
                    )}
                </Form>
            </Flex>
        </PageSection>
    );
}

export default DeliveryStep;
