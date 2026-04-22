import type { ReactElement } from 'react';
import {
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    FlexItem,
} from '@patternfly/react-core';

import type { SearchFilter } from 'types/search';

import type { CompoundSearchFilterAttribute } from '../types';
import { getCompoundSearchFilterLabelDescriptions } from '../utils/utils';

export type CompoundSearchFilterDescriptionListGroupsProps = {
    attributes: CompoundSearchFilterAttribute[];
    searchFilter: SearchFilter;
};

// For maximum composability and reusability:
// Render description list groups instead of description list.
// If rules array is empty, caller is reponsible for conditional rendering, like a warning alert.
function CompoundSearchFilterDescriptionListGroups({
    attributes,
    searchFilter,
}: CompoundSearchFilterDescriptionListGroupsProps): ReactElement {
    const labelGroupDescriptions = getCompoundSearchFilterLabelDescriptions(
        attributes,
        searchFilter
    );

    return (
        <>
            {labelGroupDescriptions.map(({ group, items }) => {
                return (
                    <DescriptionListGroup key={group.label}>
                        <DescriptionListTerm>{group.label}</DescriptionListTerm>
                        <DescriptionListDescription>
                            <Flex direction={{ default: 'column' }}>
                                {items.map((item) => (
                                    <FlexItem key={item.label}>{item.label}</FlexItem>
                                ))}
                            </Flex>
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                );
            })}
        </>
    );
}

export default CompoundSearchFilterDescriptionListGroups;
