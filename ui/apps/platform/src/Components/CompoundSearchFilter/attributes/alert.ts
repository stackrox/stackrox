// If you're adding a new attribute, make sure to add it to "alertAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';

export const ResourceType: CompoundSearchFilterAttribute = {
    displayName: 'Resource type',
    filterChipLabel: 'Resource type',
    searchTerm: 'Resource Type',
    inputType: 'select',
    inputProps: {
        options: [
            { value: 'UNKNOWN', label: 'Unknown' },
            { value: 'SECRETS', label: 'Secrets' },
            { value: 'CONFIGMAPS', label: 'Configmaps' },
            { value: 'CLUSTER_ROLES', label: 'Cluster roles' },
            { value: 'CLUSTER_ROLE_BINDINGS', label: 'Cluster role bindings' },
            { value: 'NETWORK_POLICIES', label: 'Network policies' },
            { value: 'SECURITY_CONTEXT_CONSTRAINTS', label: 'Security context constraints' },
            { value: 'EGRESS_FIREWALLS', label: 'Egress firewalls' },
        ],
    },
};

export const alertAttributes = [ResourceType];
