import React, { ReactElement, ReactNode } from 'react';
import {
    DescriptionListGroup,
    DescriptionListTerm,
    DescriptionListDescription,
} from '@patternfly/react-core';

type DescriptionListItemProps = {
    term: string | ReactNode;
    desc: string | ReactNode;
    groupClassName?: string;
};

function DescriptionListItem({
    term,
    desc,
    groupClassName,
}: DescriptionListItemProps): ReactElement {
    return (
        <DescriptionListGroup className={groupClassName}>
            <DescriptionListTerm>{term}</DescriptionListTerm>
            <DescriptionListDescription>{desc}</DescriptionListDescription>
        </DescriptionListGroup>
    );
}

export default DescriptionListItem;
