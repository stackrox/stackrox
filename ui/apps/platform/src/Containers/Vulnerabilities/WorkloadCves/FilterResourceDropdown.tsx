import React, { useState } from 'react';
import { Select, SelectOption } from '@patternfly/react-core';

import { SearchFilter } from 'types/search';

export type Resource = 'CVE' | 'Image' | 'Deployment' | 'Namespace' | 'Cluster';

type FilterResourceDropdownProps = {
    onSelect: (filterType, e, selection) => void;
    searchFilter: SearchFilter;
    resourceContext?: Resource;
};

function FilterResourceDropdown({
    onSelect,
    searchFilter,
    resourceContext,
}: FilterResourceDropdownProps) {
    const [resourceIsOpen, setResourceIsOpen] = useState(false);
    function onResourceToggle(isOpen: boolean) {
        setResourceIsOpen(isOpen);
    }
    function onResourceSelect(e, selection) {
        onSelect('resource', e, selection);
    }
    const resourceOptions = [
        <SelectOption key="CVE" value="CVE" />,
        <SelectOption key="Image" value="Image" />,
        <SelectOption key="Deployment" value="Deployment" />,
        <SelectOption key="Namespace" value="Namespace" />,
        <SelectOption key="Cluster" value="Cluster" />,
    ];

    return (
        <Select
            variant="single"
            aria-label="resource"
            onToggle={onResourceToggle}
            onSelect={onResourceSelect}
            selections={searchFilter.resource}
            isOpen={resourceIsOpen}
        >
            {resourceContext
                ? resourceOptions.filter((res) => res.key !== resourceContext)
                : resourceOptions}
        </Select>
    );
}

export default FilterResourceDropdown;
