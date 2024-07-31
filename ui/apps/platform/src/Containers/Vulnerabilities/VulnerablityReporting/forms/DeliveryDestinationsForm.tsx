import React, { ReactElement } from 'react';
import {
    Alert,
    AlertVariant,
    Divider,
    Flex,
    FlexItem,
    Form,
    PageSection,
    Title,
} from '@patternfly/react-core';
import { FormikProps } from 'formik';

import { TemplatePreviewArgs } from 'Components/EmailTemplate/EmailTemplateModal';
import NotifierConfigurationForm from 'Components/NotifierConfiguration/NotifierConfigurationForm';
import RepeatScheduleDropdown from 'Components/PatternFly/RepeatScheduleDropdown';
import DayPickerDropdown from 'Components/PatternFly/DayPickerDropdown';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import usePermissions from 'hooks/usePermissions';

import {
    defaultEmailBody as customBodyDefault,
    getDefaultEmailSubject,
} from './emailTemplateFormUtils';
import { ReportFormValues } from './useReportFormValues';
import EmailTemplatePreview from '../components/EmailTemplatePreview';

export type DeliveryDestinationsFormParams = {
    title: string;
    formik: FormikProps<ReportFormValues>;
};

function DeliveryDestinationsForm({ title, formik }: DeliveryDestinationsFormParams): ReactElement {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForIntegration = hasReadWriteAccess('Integration');

    const customSubjectDefault = getDefaultEmailSubject(
        formik.values.reportParameters.reportName,
        formik.values.reportParameters.reportScope?.name
    );

    function renderTemplatePreview({
        customBody,
        customSubject,
        customSubjectDefault,
    }: TemplatePreviewArgs) {
        return (
            <EmailTemplatePreview
                emailSubject={customSubject}
                emailBody={customBody}
                defaultEmailSubject={customSubjectDefault}
                reportParameters={formik.values.reportParameters}
            />
        );
    }

    function onDeleteLastNotifierConfiguration() {
        // Update only the schedule because spread ...formik.values overwrites deletion of last notifier.
        formik.setFieldValue('schedule', {
            intervalType: null,
            daysOfWeek: [],
            daysOfMonth: [],
        });
    }

    function onScheduledRepeatChange(_id, selection) {
        formik.setFieldValue('schedule', {
            intervalType: selection === '' ? null : selection,
            daysOfWeek: [],
            daysOfMonth: [],
        });
    }

    function onScheduledDaysChange(id, selection) {
        formik.setFieldValue(id, selection);
    }

    const cvesDiscoveredSinceError =
        formik.values.reportParameters.cvesDiscoveredSince === 'SINCE_LAST_REPORT' &&
        (formik.errors.deliveryDestinations || formik.errors.schedule);
    const isOptional = formik.values.reportParameters.cvesDiscoveredSince !== 'SINCE_LAST_REPORT';

    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-v5-u-py-lg pf-v5-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">{title}</Title>
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            {cvesDiscoveredSinceError && (
                <Alert
                    isInline
                    variant={AlertVariant.danger}
                    title="Delivery destination & schedule are both required to be configured since the 'Last scheduled report that was successfully sent' option has been selected in Step 1."
                    component="p"
                />
            )}
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Form className="pf-v5-u-py-lg pf-v5-u-px-lg">
                    <Flex direction={{ default: 'column' }}>
                        <FlexItem flex={{ default: 'flexNone' }}>
                            <NotifierConfigurationForm
                                customBodyDefault={customBodyDefault}
                                customSubjectDefault={customSubjectDefault}
                                errors={formik.errors}
                                fieldIdPrefixForFormikAndPatternFly="deliveryDestinations"
                                hasWriteAccessForIntegration={hasWriteAccessForIntegration}
                                notifierConfigurations={formik.values.deliveryDestinations}
                                onDeleteLastNotifierConfiguration={
                                    onDeleteLastNotifierConfiguration
                                }
                                renderTemplatePreview={renderTemplatePreview}
                                setFieldValue={formik.setFieldValue}
                            />
                        </FlexItem>
                    </Flex>
                    <Divider component="div" />
                    <Flex direction={{ default: 'column' }}>
                        <FlexItem>
                            <Title headingLevel="h3">
                                Configure schedule {isOptional && '(Optional)'}
                            </Title>
                        </FlexItem>
                        <FlexItem>
                            Configure or setup a schedule to share reports on a recurring basis.
                        </FlexItem>
                        <FlexItem flex={{ default: 'flexNone' }}>
                            <Flex direction={{ default: 'row' }}>
                                <FlexItem>
                                    <FormLabelGroup
                                        label="Repeat every"
                                        fieldId="schedule.intervalType"
                                        errors={formik.errors}
                                        isRequired={
                                            formik.values.reportParameters.cvesDiscoveredSince ===
                                            'SINCE_LAST_REPORT'
                                        }
                                    >
                                        <RepeatScheduleDropdown
                                            fieldId="schedule.intervalType"
                                            value={formik.values.schedule.intervalType || ''}
                                            handleSelect={onScheduledRepeatChange}
                                            isEditable={
                                                formik.values.deliveryDestinations.length > 0 ||
                                                formik.values.reportParameters
                                                    .cvesDiscoveredSince === 'SINCE_LAST_REPORT'
                                            }
                                            showNoResultsOption={
                                                formik.values.reportParameters
                                                    .cvesDiscoveredSince !== 'SINCE_LAST_REPORT'
                                            }
                                        />
                                    </FormLabelGroup>
                                </FlexItem>
                                <FlexItem>
                                    <FormLabelGroup
                                        isRequired={
                                            !!formik.values.schedule.intervalType ||
                                            formik.values.reportParameters.cvesDiscoveredSince ===
                                                'SINCE_LAST_REPORT'
                                        }
                                        label="On day(s)"
                                        fieldId={
                                            formik.values.schedule.intervalType === 'WEEKLY'
                                                ? 'schedule.daysOfWeek'
                                                : 'schedule.daysOfMonth'
                                        }
                                        errors={formik.errors}
                                    >
                                        <DayPickerDropdown
                                            fieldId={
                                                formik.values.schedule.intervalType === 'WEEKLY'
                                                    ? 'schedule.daysOfWeek'
                                                    : 'schedule.daysOfMonth'
                                            }
                                            value={
                                                formik.values.schedule.intervalType === 'WEEKLY'
                                                    ? formik.values.schedule.daysOfWeek || []
                                                    : formik.values.schedule.daysOfMonth || []
                                            }
                                            handleSelect={onScheduledDaysChange}
                                            intervalType={formik.values.schedule.intervalType}
                                            isEditable={
                                                formik.values.schedule.intervalType !== null
                                            }
                                        />
                                    </FormLabelGroup>
                                </FlexItem>
                            </Flex>
                        </FlexItem>
                    </Flex>
                </Form>
            </PageSection>
        </>
    );
}

export default DeliveryDestinationsForm;
