/*
 * Descriptors for rendering search filters as chips in toolbars, reports, and other views.
 *
 * If you add a new descriptor that should be available in view-based reports,
 * add it to the viewBasedReportFilterChipDescriptors array at the bottom of this file.
 */

import React from 'react';
import type { FilterChipGroupDescriptor } from 'Components/PatternFly/SearchFilterChips';

function renderFixableStatus(value: string): React.ReactNode {
    if (value === 'true') {
        return 'Fixable';
    }
    if (value === 'false') {
        return 'Not fixable';
    }
    return value;
}

export const cveSeverityFilterDescriptor: FilterChipGroupDescriptor = {
    displayName: 'CVE severity',
    searchFilterName: 'SEVERITY',
};

export const cveStatusFixableDescriptor: FilterChipGroupDescriptor = {
    displayName: 'CVE status',
    searchFilterName: 'FIXABLE',
    render: renderFixableStatus,
};

export const cveStatusClusterFixableDescriptor: FilterChipGroupDescriptor = {
    displayName: 'CVE status',
    searchFilterName: 'CLUSTER CVE FIXABLE',
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
