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

export const policyAttributes = [Name, Category, Severity, LifecycleStage];
