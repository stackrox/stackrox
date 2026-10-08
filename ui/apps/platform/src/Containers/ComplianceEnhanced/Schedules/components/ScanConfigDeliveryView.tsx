import type { ReactElement } from 'react';
import { DescriptionList, Flex, Title } from '@patternfly/react-core';

import NotifierConfigurationsDescriptionListGroup from 'Components/Reports/View/NotifierConfigurationsDescriptionListGroup';
import type { NotifierConfiguration } from 'services/ReportsService.types';

type ScanConfigDeliveryViewProps = {
    headingLevel: 'h2' | 'h3';
    notifiers: NotifierConfiguration[];
};

function ScanConfigDeliveryView({
    headingLevel,
    notifiers,
}: ScanConfigDeliveryViewProps): ReactElement {
    return (
        <Flex direction={{ default: 'column' }}>
            <Title headingLevel={headingLevel}>Delivery</Title>
            <DescriptionList isCompact isHorizontal>
                <NotifierConfigurationsDescriptionListGroup notifiers={notifiers} />
            </DescriptionList>
        </Flex>
    );
}

export default ScanConfigDeliveryView;
