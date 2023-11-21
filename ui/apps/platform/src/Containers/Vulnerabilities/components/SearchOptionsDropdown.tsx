// @TODO: Replace the usage of FilterResourceDropdown with this

import React, { ReactElement } from 'react';
import { Select, SelectOption } from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { SearchCategory } from 'services/SearchService';

export type SearchOption = { label: string; value: string; category: SearchCategory };

// @TODO: If list gets too long, consider putting it in it's own file
export const IMAGE_CVE_SEARCH_OPTION: SearchOption = {
    label: 'CVE',
    value: 'CVE',
    category: 'IMAGE_VULNERABILITIES',
};

export const IMAGE_SEARCH_OPTION: SearchOption = {
    label: 'Image',
    value: 'IMAGE',
    category: 'IMAGES',
};

export const DEPLOYMENT_SEARCH_OPTION: SearchOption = {
    label: 'Deployment',
    value: 'DEPLOYMENT',
    category: 'DEPLOYMENTS',
};

export const NAMESPACE_SEARCH_OPTION: SearchOption = {
    label: 'Namespace',
    value: 'NAMESPACE',
    category: 'NAMESPACES',
};

export const CLUSTER_SEARCH_OPTION: SearchOption = {
    label: 'Cluster',
    value: 'CLUSTER',
    category: 'CLUSTERS',
};

export const COMPONENT_SEARCH_OPTION: SearchOption = {
    label: 'Component',
    value: 'COMPONENT',
    category: 'IMAGE_COMPONENTS',
};

export const COMPONENT_SOURCE_SEARCH_OPTION: SearchOption = {
    label: 'Component Source',
    value: 'COMPONENT SOURCE',
    category: 'IMAGE_VULNERABILITIES',
};

export const REQUEST_NAME_SEARCH_OPTION: SearchOption = {
    label: 'Request name',
    value: 'Request Name',
    category: 'VULN_REQUEST', // This might need to change
};

export const REQUESTER_SEARCH_OPTION: SearchOption = {
    label: 'Requester',
    value: 'Requester User Name',
    category: 'VULN_REQUEST', // This might need to change
};

export type SearchOptionsDropdownProps = {
    setSearchOption: (selection) => void;
    searchOption: SearchOption;
    children: ReactElement<typeof SelectOption>[];
};

function SearchOptionsDropdown({
    setSearchOption,
    searchOption,
    children,
}: SearchOptionsDropdownProps) {
    const { isOpen, onToggle } = useSelectToggle();

    function onSearchOptionSelect(e, selection) {
        setSearchOption(selection);
    }

    return (
        <Select
            variant="single"
            toggleAriaLabel="search options filter menu toggle"
            aria-label="search options filter menu items"
            onToggle={onToggle}
            onSelect={onSearchOptionSelect}
            selections={searchOption.value}
            isOpen={isOpen}
            className="pf-u-flex-basis-0"
        >
            {children}
        </Select>
    );
}

export default SearchOptionsDropdown;
