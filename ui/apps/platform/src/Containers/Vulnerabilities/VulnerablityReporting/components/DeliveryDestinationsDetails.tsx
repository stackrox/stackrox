import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    FlexItem,
    Title,
} from '@patternfly/react-core';
import React, { ReactElement } from 'react';

import { ReportFormValues } from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';

export type DeliveryDestinationsDetailsProps = {
    formValues: ReportFormValues;
};

function DeliveryDestinationsDetails({
    formValues,
}: DeliveryDestinationsDetailsProps): ReactElement {
    const deliveryDestinations =
        formValues.deliveryDestinations.length !== 0 ? (
            formValues.deliveryDestinations.map((deliveryDestination) => (
                <li key={deliveryDestination.notifier?.id}>{deliveryDestination.notifier?.name}</li>
            ))
        ) : (
            <li>None</li>
        );

    const mailingLists =
        formValues.deliveryDestinations.length !== 0 ? (
            formValues.deliveryDestinations.map((deliveryDestination) => {
                const emails = deliveryDestination?.mailingLists.join(', ');
                return <li key={emails}>{emails}</li>;
            })
        ) : (
            <li>None</li>
        );

    return (
        <Flex direction={{ default: 'column' }}>
            <FlexItem>
                <Title headingLevel="h3">Delivery destinations</Title>
            </FlexItem>
            <FlexItem flex={{ default: 'flexNone' }}>
                <DescriptionList
                    columnModifier={{
                        default: '2Col',
                        md: '2Col',
                        sm: '1Col',
                    }}
                >
                    <DescriptionListGroup>
                        <DescriptionListTerm>Email notifier</DescriptionListTerm>
                        <DescriptionListDescription>
                            <ul>{deliveryDestinations}</ul>
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Distribution list</DescriptionListTerm>
                        <DescriptionListDescription>
                            <ul>{mailingLists}</ul>
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                </DescriptionList>
            </FlexItem>
        </Flex>
    );
}

export default DeliveryDestinationsDetails;
