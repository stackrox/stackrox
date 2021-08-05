import React, { ReactElement } from 'react';
import {
    Divider,
    DescriptionList,
    DescriptionListTerm,
    DescriptionListGroup,
    DescriptionListDescription,
} from '@patternfly/react-core';

import { ClusterInitBundle } from 'services/ClustersService';
import { getDateTime } from 'utils/dateUtils';

export type ClusterInitBundleDetailsProps = {
    meta: ClusterInitBundle;
};

function ClusterInitBundleDetails({ meta }: ClusterInitBundleDetailsProps): ReactElement {
    return (
        <DescriptionList isHorizontal>
            <DescriptionListGroup>
                <DescriptionListTerm>Name</DescriptionListTerm>
                <DescriptionListDescription>{meta.name}</DescriptionListDescription>
            </DescriptionListGroup>
            <DescriptionListGroup>
                <DescriptionListTerm>Issued</DescriptionListTerm>
                <DescriptionListDescription>
                    {getDateTime(meta.createdAt)}
                </DescriptionListDescription>
            </DescriptionListGroup>
            <DescriptionListGroup>
                <DescriptionListTerm>Expiration</DescriptionListTerm>
                <DescriptionListDescription>
                    {getDateTime(meta.expiresAt)}
                </DescriptionListDescription>
            </DescriptionListGroup>
            <DescriptionListGroup>
                <DescriptionListTerm>Created By</DescriptionListTerm>
                <DescriptionListDescription>{meta.createdBy.id}</DescriptionListDescription>
            </DescriptionListGroup>
            <Divider component="div" />
            {meta.createdBy.attributes.map((attribute) => {
                return (
                    <DescriptionListGroup key={attribute.key}>
                        <DescriptionListTerm>{attribute.key}</DescriptionListTerm>
                        <DescriptionListDescription>{attribute.value}</DescriptionListDescription>
                    </DescriptionListGroup>
                );
            })}
        </DescriptionList>
    );
}

export default ClusterInitBundleDetails;
