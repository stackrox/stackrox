import React, { ReactElement, ReactNode } from 'react';
import { Button } from '@patternfly/react-core';

type RelatedLinkProps = {
    children: ReactNode;
    'data-testid'?: string;
    id: string;
    onClick?: () => void;
};

function RelatedLink({
    children,
    'data-testid': dataTestId,
    id,
    onClick,
}: RelatedLinkProps): ReactElement {
    return (
        <Button
            data-testid={dataTestId}
            key={id}
            variant="tertiary"
            isSmall
            isDisabled={!onClick}
            onClick={onClick}
        >
            {children}
        </Button>
    );
}

export default RelatedLink;
