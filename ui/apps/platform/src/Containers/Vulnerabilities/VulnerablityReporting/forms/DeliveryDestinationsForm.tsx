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
} from '@patternfly/react-core';
import { PlusCircleIcon, TrashIcon } from '@patternfly/react-icons';
import { FormikProps } from 'formik';

import {
    DeliveryDestination,
    ReportFormValues,
} from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';
import usePermissions from 'hooks/usePermissions';

import RepeatScheduleDropdown from 'Components/PatternFly/RepeatScheduleDropdown';
import DayPickerDropdown from 'Components/PatternFly/DayPickerDropdown';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import NotifierSelection from './NotifierSelection';

export type DeliveryDestinationsFormParams = {
    title: string;
    formik: FormikProps<ReportFormValues>;
};

function DeliveryDestinationsForm({ title, formik }: DeliveryDestinationsFormParams): ReactElement {
    const { hasReadWriteAccess } = usePermissions();
    const hasNotifierWriteAccess = hasReadWriteAccess('Integration');

    function addDeliveryDestination() {
        const newDeliveryDestination: DeliveryDestination = { notifier: null, mailingLists: [] };
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
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
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
                    title="Delivery destination & schedule are both required to be configured since the 'Last successful scheduled run report' option has been selected in Step 1."
                />
            )}
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Form className="pf-u-py-lg pf-u-px-lg">
                    <Flex direction={{ default: 'column' }}>
                        <FlexItem flex={{ default: 'flexNone' }}>
                            <ul>
                                {formik.values.deliveryDestinations.map(
                                    (deliveryDestination, index) => {
                                        return (
                                            <li
                                                key={deliveryDestination.notifier?.id}
                                                className="pf-u-mb-md"
                                            >
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
                                                            prefixId={`deliveryDestinations[${index}]`}
                                                            selectedNotifier={
                                                                deliveryDestination.notifier
                                                            }
                                                            mailingLists={
                                                                deliveryDestination.mailingLists
                                                            }
                                                            allowCreate={hasNotifierWriteAccess}
                                                            formik={formik}
                                                        />
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
                                    >
                                        <RepeatScheduleDropdown
                                            fieldId="schedule.intervalType"
                                            value={formik.values.schedule.intervalType || ''}
                                            handleSelect={onScheduledRepeatChange}
                                            isEditable={
                                                formik.values.deliveryDestinations.length > 0
                                            }
                                        />
                                    </FormLabelGroup>
                                </FlexItem>
                                <FlexItem>
                                    <FormLabelGroup
                                        isRequired={!!formik.values.schedule.intervalType}
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
