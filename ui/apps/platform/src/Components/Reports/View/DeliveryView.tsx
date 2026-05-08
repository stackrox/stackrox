import type { ReactElement } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    FlexItem,
    Title,
} from '@patternfly/react-core';
import type { BreakpointModifiers } from '@patternfly/react-core';

import { formatRecurringSchedule } from 'utils/dateUtils';

import type { DeliveryType } from '../reports.types';

import NotifierConfigurationDescriptionList from './NotifierConfigurationDescriptionList';

export type DeliveryViewProps = {
    headingLevel: 'h2' | 'h3';
    horizontalTermWidthModifier: BreakpointModifiers;
    values: DeliveryType;
};

function DeliveryView({
    headingLevel,
    horizontalTermWidthModifier,
    values,
}: DeliveryViewProps): ReactElement {
    /* eslint-disable react/no-array-index-key */
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
            <FlexItem>
                <Title headingLevel={headingLevel}>Delivery</Title>
            </FlexItem>
            <FlexItem>
                <DescriptionList
                    isCompact
                    isHorizontal
                    horizontalTermWidthModifier={horizontalTermWidthModifier}
                >
                    <DescriptionListGroup>
                        <DescriptionListTerm>Destinations</DescriptionListTerm>
                        <DescriptionListDescription>
                            {values.notifiers.length === 0 ? (
                                '-'
                            ) : (
                                <Flex
                                    direction={{ default: 'column' }}
                                    spaceItems={{ default: 'spaceItemsLg' }}
                                >
                                    {values.notifiers.map((notifier, index) => (
                                        <NotifierConfigurationDescriptionList
                                            key={index}
                                            notifier={notifier}
                                        />
                                    ))}
                                </Flex>
                            )}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Schedule</DescriptionListTerm>
                        <DescriptionListDescription>
                            {values.schedule ? formatRecurringSchedule(values.schedule) : '-'}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                </DescriptionList>
            </FlexItem>
        </Flex>
    );
    /* eslint-enable react/no-array-index-key */
}

export default DeliveryView;
