import type { ReactElement } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    FlexItem,
    Title,
} from '@patternfly/react-core';
import type { BreakpointModifiers } from '@patternfly/react-core';

import type { DetailsType } from '../reports.types';

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
                    <DescriptionListGroup>
                        <DescriptionListTerm>Name</DescriptionListTerm>
                        <DescriptionListDescription>
                            {values.name || '-'}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Description</DescriptionListTerm>
                        <DescriptionListDescription>
                            {values.description || '-'}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                </DescriptionList>
            </FlexItem>
        </Flex>
    );
}

export default DetailsView;
