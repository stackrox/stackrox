import React, { ReactElement } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
} from '@patternfly/react-core';

import { ClusterRegistrationSecret } from 'services/ClustersService';

export type ClusterRegistrationSecretDescriptionProps = {
    clusterRegistrationSecret: ClusterRegistrationSecret;
};

function ClusterRegistrationSecretDescription({
    clusterRegistrationSecret,
}: ClusterRegistrationSecretDescriptionProps): ReactElement {
    return (
        <DescriptionList
            isCompact
            isHorizontal
            className="pf-v5-u-background-color-100 pf-v5-u-p-lg"
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
            {clusterRegistrationSecret.createdBy.attributes.map((attribute) => {
                return (
                    <DescriptionListGroup key={attribute.key}>
                        <DescriptionListTerm>{attribute.key}</DescriptionListTerm>
                        <DescriptionListDescription>{attribute.value}</DescriptionListDescription>
                    </DescriptionListGroup>
                );
            })}
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
        </DescriptionList>
    );
}

export default ClusterRegistrationSecretDescription;
