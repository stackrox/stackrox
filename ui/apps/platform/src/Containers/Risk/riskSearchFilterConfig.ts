import {
    Annotation as DeploymentAnnotation,
    ContainerType as DeploymentContainerType,
    Label as DeploymentLabel,
    Name as DeploymentName,
} from 'Components/CompoundSearchFilter/attributes/deployment';
import { clusterNameAttribute } from 'Components/CompoundSearchFilter/attributes/cluster';
import { Name as NamespaceName } from 'Components/CompoundSearchFilter/attributes/namespace';
import type {
    CompoundSearchFilterAttribute,
    CompoundSearchFilterConfig,
} from 'Components/CompoundSearchFilter/types';
import type { SearchFilter } from 'types/search';

// Attributes below are not part of the shared attribute files because they are not (yet) reused by
// other pages. They are standard DEPLOYMENTS search fields approved for the Risk curated filter set.
// Within the Deployment entity the displayName is the short form (the entity already conveys
// "Deployment"); the chip label and search term keep the full wording.

const attributeForDeploymentType: CompoundSearchFilterAttribute = {
    displayName: 'Type',
    filterChipLabel: 'Deployment type',
    searchTerm: 'Deployment Type',
    inputType: 'autocomplete',
};

const attributeForServiceAccount: CompoundSearchFilterAttribute = {
    displayName: 'Service account',
    filterChipLabel: 'Service account',
    searchTerm: 'Service Account',
    inputType: 'autocomplete',
};

const attributeForSecret: CompoundSearchFilterAttribute = {
    displayName: 'Secret',
    filterChipLabel: 'Secret',
    searchTerm: 'Secret',
    inputType: 'autocomplete',
};

// Port is a numeric field. It uses 'autocomplete' for value suggestions, but is registered in
// numericSearchOptions (see utils/searchUtils) so the shared autocomplete sends its value without
// the regex/exact-match modifiers that the search API rejects for numeric queries
// ("modifiers not supported for numeric query: [Equality]"). The shared 'condition-number' input
// is hardcoded to the CVSS 0-10 range, so it is not usable here.
const attributeForPort: CompoundSearchFilterAttribute = {
    displayName: 'Port',
    filterChipLabel: 'Port',
    searchTerm: 'Port',
    inputType: 'autocomplete',
};

const attributeForPrivileged: CompoundSearchFilterAttribute = {
    displayName: 'Privileged',
    filterChipLabel: 'Privileged',
    searchTerm: 'Privileged',
    inputType: 'select',
    inputProps: {
        options: [
            { value: 'true', label: 'Privileged' },
            { value: 'false', label: 'Not privileged' },
        ],
    },
};

const attributeForReadOnlyRootFilesystem: CompoundSearchFilterAttribute = {
    displayName: 'Read-only root filesystem',
    filterChipLabel: 'Read-only root filesystem',
    searchTerm: 'Read Only Root Filesystem',
    inputType: 'select',
    inputProps: {
        options: [
            { value: 'true', label: 'Read-only' },
            { value: 'false', label: 'Writable' },
        ],
    },
};

const attributeForExposureLevel: CompoundSearchFilterAttribute = {
    displayName: 'Exposure level',
    filterChipLabel: 'Exposure level',
    searchTerm: 'Exposure Level',
    inputType: 'select',
    inputProps: {
        options: [
            { value: 'EXTERNAL', label: 'External' },
            { value: 'NODE', label: 'Node' },
            { value: 'ROUTE', label: 'Route' },
            { value: 'INTERNAL', label: 'Internal' },
            { value: 'HOST', label: 'Host' },
            { value: 'UNSET', label: 'Unset' },
        ],
    },
};

// All entities use the DEPLOYMENTS search category so value autocomplete resolves against the same
// space as the Risk deployments table (scoped further by the workflow-view additionalContextFilter).
export const searchFilterConfig: CompoundSearchFilterConfig = [
    {
        displayName: 'Deployment',
        searchCategory: 'DEPLOYMENTS',
        attributes: [
            DeploymentName,
            DeploymentLabel,
            DeploymentAnnotation,
            attributeForDeploymentType,
            DeploymentContainerType,
            attributeForPrivileged,
            attributeForReadOnlyRootFilesystem,
            attributeForServiceAccount,
            attributeForSecret,
            attributeForExposureLevel,
            attributeForPort,
        ],
    },
    {
        displayName: 'Cluster',
        searchCategory: 'DEPLOYMENTS',
        attributes: [clusterNameAttribute],
    },
    {
        displayName: 'Namespace',
        searchCategory: 'DEPLOYMENTS',
        attributes: [NamespaceName],
    },
];

// Keys that are managed by the FilteredWorkflowView selector (not by the search filter). They are
// merged into the query separately, so they must never render as a removable catch-all chip.
const workflowViewSearchKeys = new Set(['platform component', 'orchestrator component']);

function getHandledConfigSearchKeys(config: CompoundSearchFilterConfig): Set<string> {
    const keys = new Set<string>();
    config.forEach((entity) => {
        entity.attributes.forEach((attribute) => {
            keys.add(attribute.searchTerm.toLowerCase());
        });
    });
    return keys;
}

// Backwards compatibility: a bookmarked or saved URL may carry filter keys that are not in the
// curated config (for example a dropped long-tail field). Those filters are still applied to the
// deployments query, so surface each remaining key as a removable text chip rather than leaving it
// as a hidden, unremovable filter. Returned as synthetic "text" attributes consumed by
// CompoundSearchFilterLabels via its attributesSeparateFromConfig prop.
export function getLegacySearchFilterAttributes(
    searchFilter: SearchFilter,
    config: CompoundSearchFilterConfig = searchFilterConfig
): CompoundSearchFilterAttribute[] {
    const configKeys = getHandledConfigSearchKeys(config);
    return Object.keys(searchFilter)
        .filter((key) => {
            const lowerKey = key.toLowerCase();
            if (configKeys.has(lowerKey) || workflowViewSearchKeys.has(lowerKey)) {
                return false;
            }
            const value = searchFilter[key];
            return Array.isArray(value) ? value.length > 0 : Boolean(value);
        })
        .map((key) => ({
            displayName: key,
            filterChipLabel: key,
            searchTerm: key as CompoundSearchFilterAttribute['searchTerm'],
            inputType: 'text',
        }));
}
