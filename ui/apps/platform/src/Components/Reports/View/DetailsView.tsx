import type { ReactElement } from 'react';
import { DescriptionList, Flex, FlexItem, Title } from '@patternfly/react-core';
import type { BreakpointModifiers } from '@patternfly/react-core';

import type { DetailsType } from '../reports.types';

import DetailsDescriptionListGroups from './DetailsDescriptionListGroups';

export type DetailsViewProps = {
    headingLevel: 'h2' | 'h3';
    horizontalTermWidthModifier: BreakpointModifiers;
    values: DetailsType;
};

function DetailsView({
    headingLevel,
    horizontalTermWidthModifier,
    values,
}: DetailsViewProps): ReactElement {
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
            <FlexItem>
                <Title headingLevel={headingLevel}>Details</Title>
            </FlexItem>
            <FlexItem>
                <DescriptionList
                    isCompact
                    isHorizontal
                    horizontalTermWidthModifier={horizontalTermWidthModifier}
                >
                    <DetailsDescriptionListGroups
                        description={values.description}
                        name={values.name}
                    />
                </DescriptionList>
            </FlexItem>
        </Flex>
    );
}

export default DetailsView;
