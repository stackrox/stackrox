import React, { ReactElement } from 'react';
import { FormikContextType, useFormikContext } from 'formik';
import { Divider, Flex, FlexItem, Form, PageSection, Title } from '@patternfly/react-core';

import NotifierConfigurationForm from 'Components/NotifierConfiguration/NotifierConfigurationForm';
import usePermissions from 'hooks/usePermissions';

import {
    ScanConfigFormValues,
    getBodyDefault,
    getSubjectDefault,
} from '../compliance.scanConfigs.utils';

function ReportConfiguration(): ReactElement {
    const formik: FormikContextType<ScanConfigFormValues> = useFormikContext();
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForIntegration = hasReadWriteAccess('Integration');

    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-v5-u-py-lg pf-v5-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">Report</Title>
                    </FlexItem>
                    <FlexItem>
                        Optionally configure e-mail delivery destinations for manually triggered
                        reports
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <Form className="pf-v5-u-py-lg pf-v5-u-px-lg">
                <NotifierConfigurationForm
                    customBodyDefault={getBodyDefault(formik.values.profiles)}
                    customSubjectDefault={getSubjectDefault(
                        formik.values.parameters.name,
                        formik.values.profiles
                    )}
                    errors={formik.errors}
                    fieldIdPrefixForFormikAndPatternFly="report.notifierConfigurations"
                    hasWriteAccessForIntegration={hasWriteAccessForIntegration}
                    notifierConfigurations={formik.values.report.notifierConfigurations}
                    setFieldValue={formik.setFieldValue}
                />
            </Form>
        </>
    );
}

export default ReportConfiguration;
