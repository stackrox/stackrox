import type { ReactElement } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    List,
    ListItem,
    Title,
} from '@patternfly/react-core';

import type { ClusterRegistrationSecret, InitBundleAttribute } from 'services/ClustersService';

export type ClusterRegistrationSecretDescriptionProps = {
    clusterRegistrationSecret: ClusterRegistrationSecret;
};

function groupAttributesByKey(attributes: InitBundleAttribute[]): Map<string, string[]> {
    const grouped = new Map<string, string[]>();
    attributes.forEach(({ key, value }) => {
        const existing = grouped.get(key) ?? [];
        grouped.set(key, [...existing, value]);
    });
    return grouped;
}

function ClusterRegistrationSecretDescription({
    clusterRegistrationSecret,
}: ClusterRegistrationSecretDescriptionProps): ReactElement {
    const groupedAttributes = groupAttributesByKey(clusterRegistrationSecret.createdBy.attributes);
    const maxRegistrations = parseInt(clusterRegistrationSecret.maxRegistrations, 10) || 0;
    const completedSet = new Set(clusterRegistrationSecret.registrationsCompleted);

    return (
        <Flex direction={{ default: 'column' }} gap={{ default: 'gapLg' }}>
            <DescriptionList
                isCompact
                isHorizontal
                horizontalTermWidthModifier={{ default: '20ch' }}
            >
                <DescriptionListGroup>
                    <DescriptionListTerm>Name</DescriptionListTerm>
                    <DescriptionListDescription>
                        {clusterRegistrationSecret.name}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Created by</DescriptionListTerm>
                    <DescriptionListDescription>
                        {clusterRegistrationSecret.createdBy.id}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Created at</DescriptionListTerm>
                    <DescriptionListDescription>
                        {clusterRegistrationSecret.createdAt}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Expires at</DescriptionListTerm>
                    <DescriptionListDescription>
                        {clusterRegistrationSecret.expiresAt}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Max registrations</DescriptionListTerm>
                    <DescriptionListDescription>
                        {maxRegistrations === 0 ? 'Unlimited' : maxRegistrations}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Used registrations</DescriptionListTerm>
                    <DescriptionListDescription>
                        {clusterRegistrationSecret.registrationsInitiated.length}
                    </DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>
            {clusterRegistrationSecret.registrationsInitiated.length > 0 && (
                <Flex direction={{ default: 'column' }} gap={{ default: 'gapMd' }}>
                    <Title headingLevel="h2">
                        {clusterRegistrationSecret.registrationsInitiated.length === 1
                            ? 'Registered cluster'
                            : 'Registered clusters'}
                    </Title>
                    <List isPlain>
                        {clusterRegistrationSecret.registrationsInitiated.map((clusterName) => {
                            const isComplete = completedSet.has(clusterName);
                            return (
                                <ListItem
                                    key={clusterName}
                                    className={isComplete ? '' : 'pf-v6-u-text-color-subtle'}
                                >
                                    {isComplete
                                        ? clusterName
                                        : `${clusterName} (registration not yet complete)`}
                                </ListItem>
                            );
                        })}
                    </List>
                </Flex>
            )}
            {groupedAttributes.size > 0 && (
                <Flex direction={{ default: 'column' }} gap={{ default: 'gapMd' }}>
                    <Title headingLevel="h2">Attributes</Title>
                    <DescriptionList
                        isCompact
                        isHorizontal
                        horizontalTermWidthModifier={{ default: '20ch' }}
                    >
                        {Array.from(groupedAttributes.entries()).map(([key, values]) => (
                            <DescriptionListGroup key={key}>
                                <DescriptionListTerm>{key}</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <List isPlain>
                                        {values.map((value) => (
                                            <ListItem key={value}>{value}</ListItem>
                                        ))}
                                    </List>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        ))}
                    </DescriptionList>
                </Flex>
            )}
        </Flex>
    );
}

export default ClusterRegistrationSecretDescription;
