import React, { useState } from 'react';
import {
    Toolbar,
    ToolbarItem,
    ToolbarFilter,
    ToolbarToggleGroup,
    ToolbarGroup,
    ToolbarContent,
    SearchInput,
    Select,
    SelectOption,
} from '@patternfly/react-core';
import { FilterIcon } from '@patternfly/react-icons';
import { FixableStatus, VulnerabilitySeverityLabel } from './types';

type Resource = 'CVE' | 'Image' | 'Deployment' | 'Namespace' | 'Cluster';
type TableFilters = {
    resource: Resource;
    cveSeverity: VulnerabilitySeverityLabel[];
    cveStatus: FixableStatus[];
};
type FilterType = 'resource' | 'cveSeverity' | 'cveStatus';

function WorkloadTableToolbar() {
    const [resourceIsOpen, setResourceIsOpen] = useState(false);
    const [inputValue, setInputValue] = useState('');
    const [cveSeverityIsOpen, setCveSeverityIsOpen] = useState(false);
    const [cveStatusIsOpen, setCveStatusIsOpen] = useState(false);
    const [filters, setFilters] = useState<TableFilters>({
        resource: 'CVE',
        cveSeverity: [],
        cveStatus: [],
    });

    function onResourceToggle(isOpen: boolean) {
        setResourceIsOpen(isOpen);
    }

    function onInputChange(newValue: string) {
        setInputValue(newValue);
    }

    function onCveSeverityToggle(isOpen: boolean) {
        setCveSeverityIsOpen(isOpen);
    }

    function onCveStatusToggle(isOpen: boolean) {
        setCveStatusIsOpen(isOpen);
    }

    function onSelect(type: FilterType, e, selection) {
        if (type === 'resource') {
            setFilters((prevFilters) => {
                return { ...prevFilters, resource: selection };
            });
        } else {
            const { checked } = e.target as HTMLInputElement;
            setFilters((prevFilters) => {
                const prevSelections = prevFilters[type];
                return {
                    ...prevFilters,
                    [type]: checked
                        ? [...prevSelections, selection]
                        : prevSelections.filter((value) => value !== selection),
                };
            });
        }
    }

    function onResourceSelect(e, selection) {
        onSelect('resource', e, selection);
    }

    function onCveSeveritySelect(e, selection) {
        onSelect('cveSeverity', e, selection);
    }

    function onCveStatusSelect(e, selection) {
        onSelect('cveStatus', e, selection);
    }

    function onDelete(type: FilterType, id: string) {
        if (type === 'cveSeverity') {
            setFilters((prevFilters) => ({
                ...prevFilters,
                cveSeverity: filters.cveSeverity.filter((fil: string) => fil !== id),
            }));
        } else if (type === 'cveStatus') {
            setFilters((prevFilters) => ({
                ...prevFilters,
                cveStatus: filters.cveStatus.filter((fil: string) => fil !== id),
            }));
        }
    }

    function onDeleteGroup(type: FilterType) {
        if (type === 'cveSeverity') {
            setFilters((prevFilters) => ({ ...prevFilters, cveSeverity: [] }));
        } else if (type === 'cveStatus') {
            setFilters((prevFilters) => ({ ...prevFilters, cveStatus: [] }));
        }
    }

    function onDeleteAll() {
        setFilters((prevFilters) => ({
            ...prevFilters,
            cveSeverity: [],
            cveStatus: [],
        }));
    }

    return (
        <Toolbar
            id="workload-cves-table-toolbar"
            collapseListedFiltersBreakpoint="xl"
            clearAllFilters={onDeleteAll}
        >
            <ToolbarContent>
                <ToolbarToggleGroup toggleIcon={<FilterIcon />} breakpoint="xl">
                    <ToolbarGroup variant="filter-group">
                        <ToolbarItem>
                            <Select
                                variant="single"
                                aria-label="resource"
                                onToggle={onResourceToggle}
                                onSelect={onResourceSelect}
                                selections={filters.resource}
                                isOpen={resourceIsOpen}
                            >
                                <SelectOption key="CVE" value="CVE" />
                                <SelectOption key="Image" value="Image" />
                                <SelectOption key="Deployment" value="Deployment" />
                                <SelectOption key="Namespace" value="Namespace" />
                                <SelectOption key="Cluster" value="Cluster" />
                            </Select>
                        </ToolbarItem>
                        <ToolbarItem variant="search-filter">
                            <SearchInput
                                aria-label="filter by CVE ID"
                                onChange={(e, value) => {
                                    onInputChange(value);
                                }}
                                value={inputValue}
                                onClear={() => {
                                    onInputChange('');
                                }}
                                placeholder="Filter by CVE ID"
                            />
                        </ToolbarItem>
                    </ToolbarGroup>
                    <ToolbarGroup>
                        <ToolbarFilter
                            chips={filters.cveSeverity}
                            deleteChip={(category, chip) =>
                                onDelete(category as FilterType, chip as string)
                            }
                            deleteChipGroup={(category) => onDeleteGroup(category as FilterType)}
                            categoryName="CVE severity"
                        >
                            <Select
                                variant="checkbox"
                                aria-label="cve-severity"
                                onToggle={onCveSeverityToggle}
                                onSelect={onCveSeveritySelect}
                                selections={filters.cveSeverity}
                                isOpen={cveSeverityIsOpen}
                                placeholderText="CVE severity"
                            >
                                <SelectOption key="Critical" value="Critical" />
                                <SelectOption key="Important" value="Important" />
                                <SelectOption key="Moderate" value="Moderate" />
                                <SelectOption key="Low" value="Low" />
                            </Select>
                        </ToolbarFilter>
                        <ToolbarFilter
                            chips={filters.cveStatus}
                            deleteChip={(category, chip) =>
                                onDelete(category as FilterType, chip as string)
                            }
                            deleteChipGroup={(category) => onDeleteGroup(category as FilterType)}
                            categoryName="CVE status"
                        >
                            <Select
                                variant="checkbox"
                                aria-label="cve-status"
                                onToggle={onCveStatusToggle}
                                onSelect={onCveStatusSelect}
                                selections={filters.cveStatus}
                                isOpen={cveStatusIsOpen}
                                placeholderText="CVE status"
                            >
                                <SelectOption key="Fixable" value="Fixable" />
                                <SelectOption key="Important" value="Not fixable" />
                            </Select>
                        </ToolbarFilter>
                    </ToolbarGroup>
                </ToolbarToggleGroup>
            </ToolbarContent>
        </Toolbar>
    );
}

export default WorkloadTableToolbar;
