import type { ReactElement } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
} from '@patternfly/react-core';

import type { NotifierConfiguration } from 'services/ReportsService.types';

export type NotifierConfigurationDescriptionListProps = {
    notifier: NotifierConfiguration;
};

function NotifierConfigurationDescriptionList({
    notifier,
}: NotifierConfigurationDescriptionListProps): ReactElement {
    const { emailConfig, notifierName } = notifier;
    const { customBody, customSubject, mailingLists } = emailConfig;

    return (
        <DescriptionList isCompact isHorizontal>
            <DescriptionListGroup>
                <DescriptionListTerm>Email notifier</DescriptionListTerm>
                <DescriptionListDescription>{notifierName}</DescriptionListDescription>
            </DescriptionListGroup>
            <DescriptionListGroup>
                <DescriptionListTerm>Distribution list</DescriptionListTerm>
                <DescriptionListDescription>{mailingLists.join(', ')}</DescriptionListDescription>
            </DescriptionListGroup>
            {customSubject && (
                <DescriptionListGroup>
                    <DescriptionListTerm>Custom subject</DescriptionListTerm>
                    <DescriptionListDescription>{customSubject}</DescriptionListDescription>
                </DescriptionListGroup>
            )}
            {customBody && (
                <DescriptionListGroup>
                    <DescriptionListTerm>Custom body</DescriptionListTerm>
                    <DescriptionListDescription>{customBody}</DescriptionListDescription>
                </DescriptionListGroup>
            )}
        </DescriptionList>
    );
}

export default NotifierConfigurationDescriptionList;
