// If you're adding a new attribute, make sure to add it to "policyAttributes" as well

import { severityLabels } from 'messages/common';
import { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Policy name',
    searchTerm: 'Policy',
    inputType: 'autocomplete',
};

export const Category: CompoundSearchFilterAttribute = {
    displayName: 'Category',
    filterChipLabel: 'Policy category',
    searchTerm: 'Category',
    inputType: 'autocomplete',
};

export const Severity: CompoundSearchFilterAttribute = {
    displayName: 'Severity',
    filterChipLabel: 'Policy severity',
    searchTerm: 'Severity',
    inputType: 'select',
    inputProps: {
        options: [
            { label: severityLabels.CRITICAL_SEVERITY, value: 'CRITICAL_SEVERITY' },
            { label: severityLabels.HIGH_SEVERITY, value: 'HIGH_SEVERITY' },
            { label: severityLabels.MEDIUM_SEVERITY, value: 'MEDIUM_SEVERITY' },
            { label: severityLabels.LOW_SEVERITY, value: 'LOW_SEVERITY' },
        ],
    },
};

export const LifecycleStage: CompoundSearchFilterAttribute = {
    displayName: 'Lifecycle stage',
    filterChipLabel: 'Lifecycle stage',
    searchTerm: 'Lifecycle Stage',
    inputType: 'select',
    inputProps: {
        options: [
            { value: 'DEPLOY', label: 'Deploy' },
            { value: 'BUILD', label: 'Build' },
            { value: 'RUNTIME', label: 'Runtime' },
        ],
    },
};

export const EnforcementAction: CompoundSearchFilterAttribute = {
    displayName: 'Enforcement action',
    filterChipLabel: 'Enforcement action',
    searchTerm: 'Enforcement',
    inputType: 'select',
    inputProps: {
        options: [
            { value: 'FAIL_BUILD_ENFORCEMENT', label: 'Fail build' },
            {
                value: 'FAIL_DEPLOYMENT_CREATE_ENFORCEMENT',
                label: 'Fail deployment create',
            },
            {
                value: 'FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT',
                label: 'Fail deployment update',
            },
            { value: 'FAIL_KUBE_REQUEST_ENFORCEMENT', label: 'Fail kube request' },
            { value: 'KILL_POD_ENFORCEMENT', label: 'Kill pod' },
            { value: 'SCALE_TO_ZERO_ENFORCEMENT', label: 'Scale to zero' },
            {
                value: 'UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT',
                label: 'Unsatisfiable node constraint',
            },
            { value: 'UNSET_ENFORCEMENT', label: 'Unset' },
        ],
    },
};

export const InactiveDeployment: CompoundSearchFilterAttribute = {
    displayName: 'Deployment status',
    filterChipLabel: 'Deployment status',
    searchTerm: 'Inactive Deployment',
    inputType: 'select',
    inputProps: {
        options: [
            { value: 'false', label: 'Active' },
            { value: 'true', label: 'Inactive' },
        ],
    },
};

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
                        label: 'Workload & Container images',
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

export const policyAttributes = [
    Name,
    Category,
    Severity,
    LifecycleStage,
    EnforcementAction,
    InactiveDeployment,
    ViolationTime,
    EntityType,
];
