import React from 'react';
import { Select, SelectOption } from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';

export const resources = ['CVE', 'IMAGE', 'DEPLOYMENT', 'NAMESPACE', 'CLUSTER'] as const;

export type Resource = (typeof resources)[number];

export type FilterResourceDropdownProps = {
    setResource: (selection) => void;
    resource: Resource;
    supportedResourceFilters?: Set<Resource>;
};

function FilterResourceDropdown({
    setResource,
    resource,
    supportedResourceFilters,
}: FilterResourceDropdownProps) {
    const { isOpen, onToggle } = useSelectToggle();

    function onResourceSelect(e, selection) {
        setResource(selection);
    }

    // TODO: this will need to be dynamic once the endpoint is in
    // /v1/internal/search/metadata/options
    const resourceOptions = [
        <SelectOption key="CVE" value="CVE">
            CVE
        </SelectOption>,
        <SelectOption key="IMAGE" value="IMAGE">
            Image
        </SelectOption>,
        <SelectOption key="DEPLOYMENT" value="DEPLOYMENT">
            Deployment
        </SelectOption>,
        <SelectOption key="NAMESPACE" value="NAMESPACE">
            Namespace
        </SelectOption>,
        <SelectOption key="CLUSTER" value="CLUSTER">
            Cluster
        </SelectOption>,
    ];

    return (
        <Select
            variant="single"
            toggleAriaLabel="resource filter menu toggle"
            aria-label="resource filter menu items"
            onToggle={onToggle}
            onSelect={onResourceSelect}
            selections={resource}
            isOpen={isOpen}
            className="pf-u-flex-basis-0"
        >
            {supportedResourceFilters
                ? resourceOptions.filter((res) => supportedResourceFilters.has(res.key as Resource))
                : resourceOptions}
        </Select>
    );
}

export default FilterResourceDropdown;
