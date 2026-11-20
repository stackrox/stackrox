import {
    Annotation as DeploymentAnnotation,
    ContainerType as DeploymentContainerType,
    Label as DeploymentLabel,
    Name as DeploymentName,
    Secret,
    ServiceAccount,
    Type as DeploymentType,
} from 'Components/CompoundSearchFilter/attributes/deployment';
import { clusterNameAttribute } from 'Components/CompoundSearchFilter/attributes/cluster';
import { Name as NamespaceName } from 'Components/CompoundSearchFilter/attributes/namespace';
import type {
    CompoundSearchFilterAttribute,
    CompoundSearchFilterConfig,
} from 'Components/CompoundSearchFilter/types';
import type { SearchFilter } from 'types/search';

// Risk-curated attributes. The select/numeric fields below stay local; autocomplete text fields that
// need regex matching (Type, Service Account, Secret) live in shared deployment attributes so
// searchUtils can register them in regexSearchOptions.

// Numeric: registered in numericSearchOptions (utils/searchUtils) so autocomplete drops the
// regex/quote modifiers the search API rejects for numeric queries.
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

// Cluster/Namespace use the DEPLOYMENTS category (not their own) so autocomplete resolves against
// the deployments the table shows.
export const searchFilterConfig: CompoundSearchFilterConfig = [
    {
        displayName: 'Deployment',
        searchCategory: 'DEPLOYMENTS',
        attributes: [
            DeploymentName,
            DeploymentLabel,
            DeploymentAnnotation,
            DeploymentType,
            DeploymentContainerType,
            attributeForPrivileged,
            attributeForReadOnlyRootFilesystem,
            ServiceAccount,
            Secret,
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

function getHandledConfigSearchKeys(config: CompoundSearchFilterConfig): Set<string> {
    const keys = new Set<string>();
    config.forEach((entity) => {
        entity.attributes.forEach((attribute) => {
            keys.add(attribute.searchTerm.toLowerCase());
        });
    });
    return keys;
}

// A bookmarked URL may carry filter keys absent from the curated config (a dropped field, or a
// workflow-view key like "Platform Component"). They still scope the query, so surface them as
// removable chips instead of hidden filters.
export function getLegacySearchFilterAttributes(
    searchFilter: SearchFilter,
    config: CompoundSearchFilterConfig = searchFilterConfig
): CompoundSearchFilterAttribute[] {
    const configKeys = getHandledConfigSearchKeys(config);
    return Object.keys(searchFilter)
        .filter((key) => {
            const lowerKey = key.toLowerCase();
            if (configKeys.has(lowerKey)) {
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
