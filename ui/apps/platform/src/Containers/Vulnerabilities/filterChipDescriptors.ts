import type { FilterChipGroupDescriptor } from 'Components/PatternFly/SearchFilterChips';

// Filter chip descriptors for displaying filter values in chip format
// These are used for rendering filter chips in reports, toolbars, and other views
// where filters need to be displayed to users.

export const cveSeverityFilterDescriptor: FilterChipGroupDescriptor = {
    displayName: 'CVE severity',
    searchFilterName: 'SEVERITY',
};

export const cveStatusFixableDescriptor: FilterChipGroupDescriptor = {
    displayName: 'CVE status',
    searchFilterName: 'FIXABLE',
};

export const cveStatusClusterFixableDescriptor: FilterChipGroupDescriptor = {
    displayName: 'CVE status',
    searchFilterName: 'CLUSTER CVE FIXABLE',
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
