import React, { ReactElement } from 'react';
import {
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

import {
    DeliveryDestination,
    ReportFormValues,
    SetReportFormFieldValue,
} from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';
import usePermissions from 'hooks/usePermissions';

import RepeatScheduleDropdown from 'Components/PatternFly/RepeatScheduleDropdown';
import DayPickerDropdown from 'Components/PatternFly/DayPickerDropdown';
import NotifierSelection from './NotifierSelection';

export type DeliveryDestinationsFormParams = {
    title: string;
    formValues: ReportFormValues;
    setFormFieldValue: SetReportFormFieldValue;
};

function DeliveryDestinationsForm({
    title,
    formValues,
    setFormFieldValue,
}: DeliveryDestinationsFormParams): ReactElement {
    const { hasReadWriteAccess } = usePermissions();
    const hasNotifierWriteAccess = hasReadWriteAccess('Integration');

    function addDeliveryDestination() {
        const newDeliveryDestination: DeliveryDestination = { notifier: null, mailingLists: [] };
        const newDeliveryDestinations = [
            ...formValues.deliveryDestinations,
            newDeliveryDestination,
        ];
        setFormFieldValue('deliveryDestinations', newDeliveryDestinations);
    }

    function removeDeliveryDestination(index: number) {
        const newDeliveryDestinations = formValues.deliveryDestinations.filter((item, i) => {
            return index !== i;
        });
        setFormFieldValue('deliveryDestinations', newDeliveryDestinations);
    }

    function onScheduledRepeatChange(_id, selection) {
        setFormFieldValue('schedule.intervalType', selection);
        setFormFieldValue('schedule.daysOfWeek', []);
        setFormFieldValue('schedule.daysOfMonth', []);
    }

    function onScheduledDaysChange(id, selection) {
        setFormFieldValue(id, selection);
    }

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
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Form className="pf-u-py-lg pf-u-px-lg">
                    <Flex direction={{ default: 'column' }}>
                        <FlexItem flex={{ default: 'flexNone' }}>
                            <ul>
                                {formValues.deliveryDestinations.map(
                                    (deliveryDestination, index) => {
                                        return (
                                            <li className="pf-u-mb-md">
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
                                                            selectedNotifier={
                                                                deliveryDestination.notifier
                                                            }
                                                            mailingLists={
                                                                deliveryDestination.mailingLists
                                                            }
                                                            setFieldValue={(
                                                                field: string,
                                                                value: string
                                                            ) => {
                                                                setFormFieldValue(
                                                                    `deliveryDestinations[${index}][${field}]`,
                                                                    value
                                                                );
                                                            }}
                                                            allowCreate={hasNotifierWriteAccess}
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
                            <Title headingLevel="h3">Configure schedule (Optional)</Title>
                        </FlexItem>
                        <FlexItem>
                            Configure or setup a schedule to share reports on a recurring basis.
                        </FlexItem>
                        <FlexItem flex={{ default: 'flexNone' }}>
                            <Flex direction={{ default: 'row' }}>
                                <FlexItem>
                                    <RepeatScheduleDropdown
                                        label="Repeat every"
                                        isRequired
                                        fieldId="schedule.intervalType"
                                        value={formValues.schedule.intervalType || ''}
                                        handleSelect={onScheduledRepeatChange}
                                    />
                                </FlexItem>
                                <FlexItem>
                                    <DayPickerDropdown
                                        label="On day(s)"
                                        isRequired
                                        fieldId={
                                            formValues.schedule.intervalType === 'WEEKLY'
                                                ? 'schedule.daysOfWeek'
                                                : 'schedule.daysOfMonth'
                                        }
                                        value={
                                            formValues.schedule.intervalType === 'WEEKLY'
                                                ? formValues.schedule.daysOfWeek || []
                                                : formValues.schedule.daysOfMonth || []
                                        }
                                        handleSelect={onScheduledDaysChange}
                                        intervalType={formValues.schedule.intervalType}
                                        isEditable={formValues.schedule.intervalType !== null}
                                    />
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
