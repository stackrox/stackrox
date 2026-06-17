import type { ReactElement } from 'react';
import { Tab, TabTitleText, Tabs } from '@patternfly/react-core';

import type { SearchFilter } from 'types/search';
import { searchValueAsArray } from 'utils/searchUtils';

import type { SelectExclusiveSingleSearchFilterAttribute } from '../types';

export type SelectExclusiveSingleTabsProps = {
    attribute: SelectExclusiveSingleSearchFilterAttribute;
    onSelectValue: (value: string) => void;
    searchFilter: SearchFilter;
    tabContentId: string;
    usePageInsets?: boolean;
};

// Generic presentation of search filter attribute as tabs.
// For example, CVE Snoozed for Nodes Results pages.
// Callback function might need to reset pagination in addition to update search filter.
function SelectExclusiveSingleTabs({
    attribute,
    onSelectValue,
    searchFilter,
    tabContentId,
    usePageInsets = true,
}: SelectExclusiveSingleTabsProps): ReactElement {
    const { inputProps, searchTerm } = attribute;
    const { options } = inputProps;

    const activeKey = searchValueAsArray(searchFilter[searchTerm])[0] ?? options[0].value;

    function onSelect(_, eventKey) {
        onSelectValue(eventKey);
    }

    return (
        <Tabs activeKey={activeKey} onSelect={onSelect} usePageInsets={usePageInsets}>
            {options.map(({ label, value }) => (
                <Tab
                    key={value}
                    eventKey={value}
                    tabContentId={tabContentId}
                    title={<TabTitleText>{label}</TabTitleText>}
                />
            ))}
        </Tabs>
    );
}

export default SelectExclusiveSingleTabs;
