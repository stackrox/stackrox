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

import NotifierSelection from './NotifierSelection';

export type DeliveryDestinationsFormParams = {
    formValues: ReportFormValues;
    setFormFieldValue: SetReportFormFieldValue;
};

function DeliveryDestinationsForm({
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

    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">Configure delivery destinations (Optional)</Title>
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
                        <FlexItem flex={{ default: 'flexNone' }} />
                    </Flex>
                </Form>
            </PageSection>
        </>
    );
}

export default DeliveryDestinationsForm;
