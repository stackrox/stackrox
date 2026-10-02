import type { ReactElement } from 'react';
import {
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
} from '@patternfly/react-core';

import type { NotifierConfiguration } from 'services/ReportsService.types';

import NotifierConfigurationDescriptionList from './NotifierConfigurationDescriptionList';

// Generic component for use cases:
// Delivery of compliance scan schedule configuration does not have schedules
// Delivery of vulnerability report configuration does have schedules

export type NotifierConfigurationsDescriptionListGroupProps = {
    notifiers: NotifierConfiguration[];
};

function NotifierConfigurationsDescriptionListGroup({
    notifiers,
}: NotifierConfigurationsDescriptionListGroupProps): ReactElement {
    /* eslint-disable react/no-array-index-key */
    return (
        <DescriptionListGroup>
            <DescriptionListTerm>Destinations</DescriptionListTerm>
            <DescriptionListDescription>
                {notifiers.length === 0 ? (
                    '-'
                ) : (
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsLg' }}
                    >
                        {notifiers.map((notifier, index) => (
                            <NotifierConfigurationDescriptionList key={index} notifier={notifier} />
                        ))}
                    </Flex>
                )}
            </DescriptionListDescription>
        </DescriptionListGroup>
    );
    /* eslint-enable react/no-array-index-key */
}

export default NotifierConfigurationsDescriptionListGroup;
