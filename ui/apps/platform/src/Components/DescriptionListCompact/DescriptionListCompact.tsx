import React, { ReactElement, CSSProperties } from 'react';
import { DescriptionList, DescriptionListProps } from '@patternfly/react-core';

// Specify top and bottom padding equivalent to variant="compact" of PatternFly tables.
const styleDescriptionListCompact = {
    '--pf-v5-c-description-list--RowGap': 'var(--pf-v5-global--spacer--xs)', // 8px (sm) = 2 * 4px (xs)
} as CSSProperties;

// TODO Replace occurrences with DescriptionList if variant="compact" becomes available.
// Component props have inconsistent name because DescriptionListProps is from PatternFly.
/* eslint-disable generic/react-props-name */
function DescriptionListCompact({ children, ...rest }: DescriptionListProps): ReactElement {
    return (
        <DescriptionList {...rest} style={styleDescriptionListCompact}>
            {children}
        </DescriptionList>
    );
}
/* eslint-enable generic/react-props-name */

export default DescriptionListCompact;
