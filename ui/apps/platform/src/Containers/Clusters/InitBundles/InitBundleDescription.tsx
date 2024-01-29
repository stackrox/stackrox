import React, { ReactElement } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
} from '@patternfly/react-core';

import { ClusterInitBundle } from 'services/ClustersService';

export type InitBundleDescriptionProps = {
    initBundle: ClusterInitBundle;
};

function InitBundleDescription({ initBundle }: InitBundleDescriptionProps): ReactElement {
    return (
        <DescriptionList isCompact isHorizontal className="pf-u-background-color-100 pf-u-p-lg">
            <DescriptionListGroup>
                <DescriptionListTerm>Name</DescriptionListTerm>
                <DescriptionListDescription>{initBundle.name}</DescriptionListDescription>
            </DescriptionListGroup>
            <DescriptionListGroup>
                <DescriptionListTerm>Created by</DescriptionListTerm>
                <DescriptionListDescription>{initBundle.createdBy.id}</DescriptionListDescription>
            </DescriptionListGroup>
            {initBundle.createdBy.attributes.map((attribute) => {
                return (
                    <DescriptionListGroup key={attribute.key}>
                        <DescriptionListTerm>{attribute.key}</DescriptionListTerm>
                        <DescriptionListDescription>{attribute.value}</DescriptionListDescription>
                    </DescriptionListGroup>
                );
            })}
            <DescriptionListGroup>
                <DescriptionListTerm>Created at</DescriptionListTerm>
                <DescriptionListDescription>{initBundle.createdAt}</DescriptionListDescription>
            </DescriptionListGroup>
            <DescriptionListGroup>
                <DescriptionListTerm>Expires at</DescriptionListTerm>
                <DescriptionListDescription>{initBundle.expiresAt}</DescriptionListDescription>
            </DescriptionListGroup>
        </DescriptionList>
    );
}

export default InitBundleDescription;
