import type { ReactElement } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
} from '@patternfly/react-core';

import { isDefaultEmailTemplate } from 'Components/EmailTemplate/EmailTemplate.utils';
import type { NotifierConfiguration } from 'services/ReportsService.types';

export type NotifierConfigurationDescriptionListProps = {
    notifier: NotifierConfiguration;
};

function NotifierConfigurationDescriptionList({
    notifier,
}: NotifierConfigurationDescriptionListProps): ReactElement {
    const { emailConfig, notifierName } = notifier;
    const { customBody, customSubject, mailingLists } = emailConfig;
    const hasDefaultEmailTemplate = isDefaultEmailTemplate({
        customBody,
        customSubject,
    });

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
            <DescriptionListGroup>
                <DescriptionListTerm>Email template</DescriptionListTerm>
                <DescriptionListDescription>
                    {hasDefaultEmailTemplate ? 'Default template' : 'Custom template'}
                </DescriptionListDescription>
            </DescriptionListGroup>
        </DescriptionList>
    );
}

export default NotifierConfigurationDescriptionList;
