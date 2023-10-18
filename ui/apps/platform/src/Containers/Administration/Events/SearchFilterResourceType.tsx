import React, { useState } from 'react';
import { Divider, Select, SelectOption } from '@patternfly/react-core';

import { resourceTypes } from 'services/AdministrationEventsService';

const optionAll = 'All';

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
    const [isOpen, setIsOpen] = useState(false);

    function onSelect(_event, selection) {
        setResourceType(selection === optionAll ? undefined : selection);
        setIsOpen(false);
    }

    const options = resourceTypes.map((resourceTypeArg) => (
        <SelectOption key={resourceTypeArg} value={resourceTypeArg}>
            {resourceTypeArg}
        </SelectOption>
    ));
    options.push(
        <Divider key="Divider" />,
        <SelectOption key="All" value={optionAll} isPlaceholder>
            All resource types
        </SelectOption>
    );

    return (
        <Select
            variant="single"
            aria-label="Resource type filter menu items"
            toggleAriaLabel="Resource type filter menu toggle"
            onToggle={setIsOpen}
            onSelect={onSelect}
            selections={resourceType ?? optionAll}
            isDisabled={isDisabled}
            isOpen={isOpen}
        >
            {options}
        </Select>
    );
}

export default SearchFilterResourceType;
