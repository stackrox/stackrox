import React, { ReactElement } from 'react';
import {
    Alert,
    AlertVariant,
    Button,
    Card,
    CardBody,
    CardTitle,
    Divider,
    Flex,
    FlexItem,
    Form,
    PageSection,
    Title,
    Tooltip,
} from '@patternfly/react-core';
import { HelpIcon, PencilAltIcon, PlusCircleIcon, TrashIcon } from '@patternfly/react-icons';
import { FormikProps } from 'formik';
import isEqual from 'lodash/isEqual';

import {
    EmailTemplateFormData,
    isDefaultEmailTemplate,
} from 'Components/EmailTemplate/EmailTemplate.utils';
import EmailTemplateModal, {
    TemplatePreviewArgs,
} from 'Components/EmailTemplate/EmailTemplateModal';
import RepeatScheduleDropdown from 'Components/PatternFly/RepeatScheduleDropdown';
import DayPickerDropdown from 'Components/PatternFly/DayPickerDropdown';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import useIndexKey from 'hooks/useIndexKey';
import usePermissions from 'hooks/usePermissions';
import { NotifierConfiguration } from 'services/ReportsService.types';

import NotifierSelection from './NotifierSelection';
import { defaultEmailBody, getDefaultEmailSubject } from './emailTemplateFormUtils';
import { ReportFormValues } from './useReportFormValues';
import EmailTemplatePreview from '../components/EmailTemplatePreview';
import useEmailTemplateModal from '../hooks/useEmailTemplateModal';

export type DeliveryDestinationsFormParams = {
    title: string;
    formik: FormikProps<ReportFormValues>;
};

function DeliveryDestinationsForm({ title, formik }: DeliveryDestinationsFormParams): ReactElement {
    const { hasReadWriteAccess } = usePermissions();
    const hasNotifierWriteAccess = hasReadWriteAccess('Integration');
    const { keyFor } = useIndexKey();

    const {
        isEmailTemplateModalOpen,
        closeEmailTemplateModal,
        selectedEmailSubject,
        selectedEmailBody,
        selectedDeliveryDestination,
        setSelectedDeliveryDestination,
    } = useEmailTemplateModal();

    const defaultEmailSubject = getDefaultEmailSubject(
        formik.values.reportParameters.reportName,
        formik.values.reportParameters.reportScope?.name
    );

    function onEmailTemplateChange(formData: EmailTemplateFormData) {
        const index = formik.values.deliveryDestinations.findIndex((deliveryDestination) =>
            isEqual(deliveryDestination, selectedDeliveryDestination)
        );
        if (index >= 0) {
            const prevDeliveryDestination = formik.values.deliveryDestinations[index];
            const { emailConfig } = prevDeliveryDestination;
            formik.setFieldValue(`deliveryDestinations[${index}]`, {
                ...prevDeliveryDestination,
                emailConfig: {
                    ...emailConfig,
                    customSubject: formData.customSubject,
                    customBody: formData.customBody,
                },
            });
        }
    }

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

    function addDeliveryDestination() {
        const newDeliveryDestination: NotifierConfiguration = {
            emailConfig: {
                notifierId: '',
                mailingLists: [],
                customSubject: '',
                customBody: '',
            },
            notifierName: '',
        };
        const newDeliveryDestinations = [
            ...formik.values.deliveryDestinations,
            newDeliveryDestination,
        ];
        formik.setFieldValue('deliveryDestinations', newDeliveryDestinations);
    }

    function removeDeliveryDestination(index: number) {
        const newDeliveryDestinations = formik.values.deliveryDestinations.filter((item, i) => {
            return index !== i;
        });
        if (newDeliveryDestinations.length === 0) {
            formik.setValues({
                ...formik.values,
                deliveryDestinations: newDeliveryDestinations,
                schedule: {
                    intervalType: null,
                    daysOfWeek: [],
                    daysOfMonth: [],
                },
            });
        } else {
            formik.setFieldValue('deliveryDestinations', newDeliveryDestinations);
        }
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
                />
            )}
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Form className="pf-v5-u-py-lg pf-v5-u-px-lg">
                    <Flex direction={{ default: 'column' }}>
                        <FlexItem flex={{ default: 'flexNone' }}>
                            <ul>
                                {formik.values.deliveryDestinations.map(
                                    (deliveryDestination, index) => {
                                        const { emailConfig, notifierName } = deliveryDestination;
                                        const {
                                            customBody,
                                            customSubject,
                                            mailingLists,
                                            notifierId,
                                        } = emailConfig;
                                        const selectedNotifier =
                                            notifierId.length === 0
                                                ? null
                                                : { id: notifierId, name: notifierName };
                                        const fieldId = `deliveryDestinations[${index}]`;
                                        const isDefaultEmailTemplateApplied =
                                            isDefaultEmailTemplate({ customBody, customSubject });
                                        return (
                                            <li key={keyFor(index)} className="pf-v5-u-mb-md">
                                                <Card>
                                                    <CardTitle>
                                                        <Flex
                                                            alignItems={{
                                                                default: 'alignItemsCenter',
                                                            }}
                                                        >
                                                            <FlexItem flex={{ default: 'flex_1' }}>
                                                                Delivery destination
                                                            </FlexItem>
                                                            <FlexItem>
                                                                <Button
                                                                    variant="plain"
                                                                    aria-label="Delete delivery destination"
                                                                    onClick={() => {
                                                                        removeDeliveryDestination(
                                                                            index
                                                                        );
                                                                    }}
                                                                >
                                                                    <TrashIcon />
                                                                </Button>
                                                            </FlexItem>
                                                        </Flex>
                                                    </CardTitle>
                                                    <CardBody>
                                                        <NotifierSelection
                                                            prefixId={fieldId}
                                                            selectedNotifier={selectedNotifier}
                                                            mailingLists={mailingLists}
                                                            allowCreate={hasNotifierWriteAccess}
                                                            formik={formik}
                                                        />
                                                        <div className="pf-v5-u-mt-md">
                                                            <FormLabelGroup
                                                                label="Email template"
                                                                labelIcon={
                                                                    <Tooltip
                                                                        content={
                                                                            isDefaultEmailTemplateApplied ? (
                                                                                <div>
                                                                                    Default template
                                                                                    applied. Edit to
                                                                                    customize.
                                                                                </div>
                                                                            ) : (
                                                                                <div>
                                                                                    Custom template
                                                                                    applied. Edit to
                                                                                    customize.
                                                                                </div>
                                                                            )
                                                                        }
                                                                    >
                                                                        <Button
                                                                            variant="plain"
                                                                            aria-label="More info for email template field"
                                                                            aria-describedby={`${fieldId}.customSubject`}
                                                                        >
                                                                            <HelpIcon aria-label="More info for email template field" />
                                                                        </Button>
                                                                    </Tooltip>
                                                                }
                                                                fieldId={`${fieldId}.customSubject`}
                                                                errors={formik.errors}
                                                                isRequired
                                                            >
                                                                <Button
                                                                    variant="link"
                                                                    isInline
                                                                    icon={<PencilAltIcon />}
                                                                    onClick={() => {
                                                                        setSelectedDeliveryDestination(
                                                                            deliveryDestination
                                                                        );
                                                                    }}
                                                                    iconPosition="right"
                                                                >
                                                                    {isDefaultEmailTemplateApplied
                                                                        ? 'Default template applied'
                                                                        : 'Custom template applied'}
                                                                </Button>
                                                            </FormLabelGroup>
                                                        </div>
                                                    </CardBody>
                                                </Card>
                                            </li>
                                        );
                                    }
                                )}
                                <li>
                                    <Button
                                        variant="link"
                                        icon={<PlusCircleIcon />}
                                        onClick={addDeliveryDestination}
                                    >
                                        Add delivery destination
                                    </Button>
                                </li>
                            </ul>
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
            <EmailTemplateModal
                isOpen={isEmailTemplateModalOpen}
                onClose={closeEmailTemplateModal}
                onChange={onEmailTemplateChange}
                customBodyDefault={defaultEmailBody}
                customBodyInitial={selectedEmailBody}
                customSubjectDefault={defaultEmailSubject}
                customSubjectInitial={selectedEmailSubject}
                renderTemplatePreview={renderTemplatePreview}
            />
        </>
    );
}

export default DeliveryDestinationsForm;
