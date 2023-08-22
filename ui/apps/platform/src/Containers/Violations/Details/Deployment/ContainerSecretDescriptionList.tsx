import React, { ReactElement } from 'react';
import { DescriptionList } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import { EmbeddedSecret } from 'types/deployment.proto';

export type ContainerSecretDescriptionListProps = {
    secret: EmbeddedSecret;
};

function ContainerSecretDescriptionList({
    secret: { name, path },
}: ContainerSecretDescriptionListProps): ReactElement {
    return (
        <DescriptionList isCompact isHorizontal>
            <DescriptionListItem term="Name" desc={name} />
            <DescriptionListItem term="Container path" desc={path} />
        </DescriptionList>
    );
}

export default ContainerSecretDescriptionList;
