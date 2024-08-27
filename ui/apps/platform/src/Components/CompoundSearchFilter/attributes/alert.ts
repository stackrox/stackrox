// If you're adding a new attribute, make sure to add it to "alertAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';

export const ViolationTime: CompoundSearchFilterAttribute = {
    displayName: 'Violation time',
    filterChipLabel: 'Violation time',
    searchTerm: 'Violation Time',
    inputType: 'date-picker',
};

export const EntityType: CompoundSearchFilterAttribute = {
    displayName: 'Entity type',
    filterChipLabel: 'Entity type',
    searchTerm: 'Resource Type',
    inputType: 'select',
    inputProps: {
        groupOptions: [
            {
                name: 'Workload & Container image',
                options: [
                    {
                        label: 'Deployments & Container images',
                        value: 'UNKNOWN',
                    },
                ],
            },
            {
                name: 'Resource',
                options: [
                    { label: 'Cluster role bindings', value: 'CLUSTER_ROLE_BINDINGS' },
                    { label: 'Cluster roles', value: 'CLUSTER_ROLES' },
                    { label: 'Configmaps', value: 'CONFIGMAPS' },
                    { label: 'Egress firewalls', value: 'EGRESS_FIREWALLS' },
                    { label: 'Network policies', value: 'NETWORK_POLICIES' },
                    { label: 'Secrets', value: 'SECRETS' },
                    {
                        label: 'Security context constraints',
                        value: 'SECURITY_CONTEXT_CONSTRAINTS',
                    },
                ],
            },
        ],
    },
};

export const alertAttributes = [ViolationTime, EntityType];
