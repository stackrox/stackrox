import React, { ReactElement, ReactNode } from 'react';
import { Button, FlexItem } from '@patternfly/react-core';

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
        <FlexItem spacer={{ default: 'spacerSm' }}>
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
        </FlexItem>
    );
}

export default RelatedLink;
