import type { ReactElement } from 'react';
import {
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
} from '@patternfly/react-core';

// Generic component for use cases:
// Parameters of compliance scan schedule configuration
// Details of vulnerability report configuration

export type DetailsDescriptionListGroupsProps = {
    description: string;
    name: string;
};

function DetailsDescriptionListGroups({
    description,
    name,
}: DetailsDescriptionListGroupsProps): ReactElement {
    return (
        <>
            <DescriptionListGroup>
                <DescriptionListTerm>Name</DescriptionListTerm>
                <DescriptionListDescription>{name || '-'}</DescriptionListDescription>
            </DescriptionListGroup>
            <DescriptionListGroup>
                <DescriptionListTerm>Description</DescriptionListTerm>
                <DescriptionListDescription>{description || '-'}</DescriptionListDescription>
            </DescriptionListGroup>
        </>
    );
}

export default DetailsDescriptionListGroups;
