import React from 'react';
import { Select, SelectOption } from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';

export type Resource = 'CVE' | 'IMAGE' | 'DEPLOYMENT' | 'NAMESPACE' | 'CLUSTER';

type FilterResourceDropdownProps = {
    setResource: (selection) => void;
    resource: Resource;
    resourceContext?: Resource;
};

function FilterResourceDropdown({
    setResource,
    resource,
    resourceContext,
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
            aria-label="resource"
            onToggle={onToggle}
            onSelect={onResourceSelect}
            selections={resource}
            isOpen={isOpen}
            className="pf-u-w-25"
        >
            {resourceContext
                ? resourceOptions.filter((res) => res.key !== resourceContext)
                : resourceOptions}
        </Select>
    );
}

export default FilterResourceDropdown;
