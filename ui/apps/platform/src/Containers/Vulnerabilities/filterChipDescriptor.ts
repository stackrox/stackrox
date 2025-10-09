/*
 * Descriptors for rendering search filters as chips in toolbars, reports, and other views.
 *
 * If you add a new descriptor that should be available in view-based reports,
 * add it to the viewBasedReportFilterChipDescriptors array at the bottom of this file.
 */

import React from 'react';
import type { FilterChipGroupDescriptor } from 'Components/PatternFly/SearchFilterChips';

function renderFixableStatus(value: string): React.ReactNode {
    // Handle backend values (true/false) and display values (Fixable/Not fixable)
    if (value === 'true' || value === 'Fixable') {
        return 'Fixable';
    }
    if (value === 'false' || value === 'Not fixable') {
        return 'Not fixable';
    }
    // For any other values, return as-is
    return value;
}

export const cveSeverityFilterDescriptor: FilterChipGroupDescriptor = {
    displayName: 'CVE severity',
    searchFilterName: 'Severity',
};

export const cveStatusFixableDescriptor: FilterChipGroupDescriptor = {
    displayName: 'CVE status',
    searchFilterName: 'Fixable',
    render: renderFixableStatus,
};

export const cveStatusClusterFixableDescriptor: FilterChipGroupDescriptor = {
    displayName: 'CVE status',
    searchFilterName: 'Cluster CVE Fixable',
    render: renderFixableStatus,
};

export const cveSnoozedDescriptor: FilterChipGroupDescriptor = {
    displayName: 'CVE snoozed',
    searchFilterName: 'CVE Snoozed',
};

export const platformComponentDescriptor: FilterChipGroupDescriptor = {
    displayName: 'Platform component',
    searchFilterName: 'Platform Component',
};

export const vulnerabilityStateDescriptor: FilterChipGroupDescriptor = {
    displayName: 'Vulnerability state',
    searchFilterName: 'Vulnerability State',
};

// These descriptors represent special filters that aren't part of CompoundSearchFilter config.
// Only add descriptors here if they should be available in view-based reports.
export const viewBasedReportFilterChipDescriptors = [
    cveSeverityFilterDescriptor,
    cveStatusFixableDescriptor,
    cveStatusClusterFixableDescriptor,
    platformComponentDescriptor,
    vulnerabilityStateDescriptor,
];
