import React, { ReactElement, ReactNode } from 'react';
import {
    DescriptionListGroup,
    DescriptionListTerm,
    DescriptionListDescription,
} from '@patternfly/react-core';

type DescriptionListItemProps = {
    term: string;
    desc: string | ReactNode;
};

function DescriptionListItem({ term, desc }: DescriptionListItemProps): ReactElement {
    return (
        <DescriptionListGroup>
            <DescriptionListTerm>{term}</DescriptionListTerm>
            <DescriptionListDescription>{desc}</DescriptionListDescription>
        </DescriptionListGroup>
    );
}

export default DescriptionListItem;
