import React, { ReactElement } from 'react';
import { DescriptionList } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';

type FlatObjectDescriptionListProps = {
    data: Record<string, boolean | number | string>;
};

function FlatObjectDescriptionList({ data }: FlatObjectDescriptionListProps): ReactElement {
    return (
        <DescriptionList isCompact isHorizontal>
            {Object.entries(data).map(([key, value]) => (
                <DescriptionListItem key={key} term={key} desc={String(value)} />
            ))}
        </DescriptionList>
    );
}

export default FlatObjectDescriptionList;
