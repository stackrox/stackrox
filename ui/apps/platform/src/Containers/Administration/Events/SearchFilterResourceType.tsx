import { Divider, SelectOption } from '@patternfly/react-core';
import SimpleSelect from 'Components/CompoundSearchFilter/components/SimpleSelect';

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
    function onSelect(selection: string | number | undefined) {
        setResourceType(selection === optionAll ? undefined : (selection as string | undefined));
    }

    const options = resourceTypes.map((resourceTypeArg) => (
        <SelectOption key={resourceTypeArg} value={resourceTypeArg}>
            {resourceTypeArg}
        </SelectOption>
    ));
    options.push(
        <Divider key="Divider" />,
        <SelectOption key="All" value={optionAll}>
            {optionAll}
        </SelectOption>
    );

    return (
        <SimpleSelect
            value={resourceType ?? optionAll}
            onChange={onSelect}
            isDisabled={isDisabled}
            ariaLabelMenu="Resource type filter menu items"
            ariaLabelToggle="Resource type filter menu toggle"
        >
            {options}
        </SimpleSelect>
    );
}

export default SearchFilterResourceType;
