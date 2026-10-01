import type { ReactElement } from 'react';
import { DescriptionList, Flex, FlexItem, Title } from '@patternfly/react-core';
import type { BreakpointModifiers } from '@patternfly/react-core';

import type { DeliveryType } from '../reports.types';

import NotifierConfigurationsDescriptionListGroup from './NotifierConfigurationsDescriptionListGroup';
import ScheduleDescriptionListGroup from './ScheduleDescriptionListGroup';

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
                    <NotifierConfigurationsDescriptionListGroup notifiers={values.notifiers} />
                    <ScheduleDescriptionListGroup schedule={values.schedule} />
                </DescriptionList>
            </FlexItem>
        </Flex>
    );
}

export default DeliveryView;
