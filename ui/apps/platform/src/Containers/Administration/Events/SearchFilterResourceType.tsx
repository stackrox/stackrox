import { Divider, SelectOption } from '@patternfly/react-core';
import SelectSingle from 'Components/SelectSingle/SelectSingle';

import { resourceTypes } from 'services/AdministrationEventsService';

const optionAll = 'All resource types';

type SearchFilterResourceTypeProps = {
    isDisabled: boolean;
    resourceType: string | undefined;
    setResourceType: (resourceType: string | undefined) => void;
};

function SearchFilterResourceType({
    isDisabled,
    resourceType,
    setResourceType,
}: SearchFilterResourceTypeProps) {
    function onSelect(_id: string, selection: string) {
        setResourceType(selection === optionAll ? undefined : selection);
    }

    const options = [
        <SelectOption key="All" value={optionAll}>
            {optionAll}
        </SelectOption>,
        <Divider key="Divider" />,
        ...resourceTypes.map((resourceTypeArg) => (
            <SelectOption key={resourceTypeArg} value={resourceTypeArg}>
                {resourceTypeArg}
            </SelectOption>
        )),
    ];

    return (
        <SelectSingle
            id="resource-type-filter"
            value={resourceType ?? optionAll}
            handleSelect={onSelect}
            isDisabled={isDisabled}
            placeholderText="Select resource type"
            toggleAriaLabel="Resource type filter menu toggle"
        >
            {options}
        </SelectSingle>
    );
}

export default SearchFilterResourceType;
