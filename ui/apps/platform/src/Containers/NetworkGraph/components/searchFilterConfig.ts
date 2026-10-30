import {
    Inactive as attributeForDeploymentStatus,
    Name as attributeForDeploymentName,
    deploymentAttributes,
} from 'Components/CompoundSearchFilter/attributes/deployment';
import type {
    CompoundSearchFilterAttribute,
    CompoundSearchFilterConfig,
    CompoundSearchFilterEntity,
} from 'Components/CompoundSearchFilter/types';
import type { SearchFilter } from 'types/search';
import type { SearchFieldLabel } from 'types/searchOptions';

// Deployment Type is not part of the shared deploymentAttributes, but the legacy Network Graph
// filters surfaced every DEPLOYMENTS search option (including Deployment Type), so add it here.
const attributeForDeploymentType: CompoundSearchFilterAttribute = {
    // displayName is "Type" (not "Deployment Type") to avoid redundancy with the Deployment entity
    // in the dropdown; the chip label and search term keep the full "Deployment Type" wording.
    displayName: 'Type',
    filterChipLabel: 'Deployment type',
    searchTerm: 'Deployment Type',
    inputType: 'autocomplete',
};

// The Network Graph manages Cluster, Namespace, and Deployment selection through dedicated
// scope selectors, which write those values into the same URL search filter. Exclude the
// Deployment name attribute so the compound search filter does not compete with the Deployment
// scope selector for the "Deployment" search key (matches the legacy searchOptionExclusions).
// Also exclude the Status attribute, which the Network Graph does not use.
const excludedAttributes = [attributeForDeploymentName, attributeForDeploymentStatus];
const entityForDeployment: CompoundSearchFilterEntity = {
    displayName: 'Deployment',
    searchCategory: 'DEPLOYMENTS',
    attributes: [
        ...deploymentAttributes.filter((attribute) => !excludedAttributes.includes(attribute)),
        attributeForDeploymentType,
    ],
};

export const searchFilterConfig: CompoundSearchFilterConfig = [entityForDeployment];

// Search keys owned by the dedicated scope selectors (Cluster/Namespace/Deployment); they are
// managed outside the compound search filter and must never render a legacy catch-all chip.
// Matched case-sensitively because useScopeHierarchy reads these keys by exact property access
// (searchFilter.Cluster, etc.): a case-different variant is not consumed there, so it must stay in
// the catch-all as a removable chip rather than becoming a hidden, unremovable filter.
const scopeSearchKeys: SearchFieldLabel[] = ['Cluster', 'Namespace', 'Deployment'];

// Keys represented by the config attributes already render a chip via the compound search filter's
// own label component, so they are excluded from the catch-all. Compared lowercased because that
// label component reads values with getValueByCaseInsensitiveKey (case-insensitive).
function getHandledConfigSearchKeys(config: CompoundSearchFilterConfig): Set<string> {
    const keys = new Set<string>();
    config.forEach((entity) => {
        entity.attributes.forEach((attribute) => {
            keys.add(attribute.searchTerm.toLowerCase());
            if (attribute.inputType === 'select-exclusive-double') {
                keys.add(attribute.inputProps.category2.toLowerCase());
            }
        });
    });
    return keys;
}

// Backwards compatibility: a bookmarked URL may contain filter keys that are not in the current
// compound search filter config. Those filters are still applied to the graph query, so we surface
// each remaining key as a removable text chip. Returns synthetic "text" attributes consumed by
// CompoundSearchFilterLabels via its attributesSeparateFromConfig prop, reusing the existing chip
// rendering and removal logic.
export function getLegacySearchFilterAttributes(
    searchFilter: SearchFilter,
    config: CompoundSearchFilterConfig = searchFilterConfig
): CompoundSearchFilterAttribute[] {
    const scopeKeys = new Set<string>(scopeSearchKeys);
    const configKeys = getHandledConfigSearchKeys(config);
    return Object.keys(searchFilter)
        .filter((key) => {
            // Scope keys match exactly; config keys match case-insensitively (see comments above).
            if (scopeKeys.has(key) || configKeys.has(key.toLowerCase())) {
                return false;
            }
            const value = searchFilter[key];
            return Array.isArray(value) ? value.length > 0 : Boolean(value);
        })
        .map((key) => ({
            displayName: key,
            filterChipLabel: key,
            searchTerm: key as SearchFieldLabel,
            inputType: 'text',
        }));
}
