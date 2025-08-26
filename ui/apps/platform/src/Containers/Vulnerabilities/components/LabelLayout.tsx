import React from 'react';
import type { ReactNode } from 'react';
import { Flex, FlexItem } from '@patternfly/react-core';

export type LabelLayoutProps = {
    labels: ReactNode[];
};

function LabelLayout({ labels }: LabelLayoutProps) {
    // FlexItem is needed for spaceItems if label is wrapped in a Tooltip element.
    /* eslint-disable react/no-array-index-key */
    return (
        <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsSm' }}>
            {labels.map((label, key) => (
                <FlexItem key={key}>{label}</FlexItem>
            ))}
        </Flex>
    );
    /* eslint-enable react/no-array-index-key */
}

export default LabelLayout;
