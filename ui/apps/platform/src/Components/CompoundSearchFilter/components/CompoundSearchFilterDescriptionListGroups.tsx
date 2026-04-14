import type { ReactElement } from 'react';
import {
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Label,
    LabelGroup,
} from '@patternfly/react-core';

import type { SearchFilter } from 'types/search';

import type { CompoundSearchFilterAttribute } from '../types';
import { getCompoundSearchFilterLabelDescriptions } from '../utils/utils';

export type CompoundSearchFilterDescriptionListGroupsProps = {
    attributes: CompoundSearchFilterAttribute[];
    searchFilter: SearchFilter;
};

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
                            <LabelGroup categoryName={group.label}>
                                {items.map((item) => (
                                    <Label key={item.label} variant="outline">
                                        {item.label}
                                    </Label>
                                ))}
                            </LabelGroup>
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                );
            })}
        </>
    );
}

export default CompoundSearchFilterDescriptionListGroups;
