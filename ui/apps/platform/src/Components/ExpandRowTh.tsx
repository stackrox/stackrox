import React from 'react';
import { Th, ThProps } from '@patternfly/react-table';

const expandButtonWidth = '1em';
const expandButtonPaddingX = '(var(--pf-v5-global--spacer--md) * 2)';
const firstTableCellPadding = 'var(--pf-v5-c-table--cell--PaddingLeft)';

export default function ExpandRowTh(props: ThProps) {
    return (
        <Th
            {...props}
            aria-label={props['aria-label'] || 'Expand row'}
            style={{
                // Setting a defined width here prevents column shift when the table is in a loading state
                width: `calc(${expandButtonWidth} + ${expandButtonPaddingX} + ${firstTableCellPadding})`,
                ...props.style,
            }}
        />
    );
}
