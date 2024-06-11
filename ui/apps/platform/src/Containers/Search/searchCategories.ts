import cloneDeep from 'lodash/cloneDeep';

import { IsRouteEnabled } from 'hooks/useIsRouteEnabled';
import { SearchResultCategory } from 'services/SearchService';
import {
    RouteKey,
    clustersBasePath,
    configManagementPath,
    policiesBasePath,
    riskBasePath,
    violationsBasePath,
    vulnManagementPath,
    vulnerabilitiesNodeCvesPath,
    vulnerabilitiesWorkloadCvesPath,
    vulnerabilityNamespaceViewPath,
} from 'routePaths';
import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import { getQueryString } from 'utils/queryStringUtils';

const configManagementRolesPath = `${configManagementPath}/roles`;
const configManagementSecretsPath = `${configManagementPath}/secrets`;
const configManagementServiceAccountsPath = `${configManagementPath}/serviceaccounts`;

const vulnManagementImagesPath = `${vulnManagementPath}/images`;
const vulnManagementNamespacesPath = `${vulnManagementPath}/namespaces`;
const vulnManagementNodesPath = `${vulnManagementPath}/nodes`;

type SearchResultCategoryDescriptor = {
    filterOn: FilterOnDescriptor | null;
    viewLinks: SearchLinkDescriptor[];
};

type FilterOnDescriptor = {
    filterCategory: string; // label and value in SearchEntry object which has type: 'categoryOption'
    filterLinks: SearchLinkDescriptor[];
};

/*
 * A filter link appends ?queryString which includes filterCategory and name from SearchResult.
 * A view link appends /id from SearchResult.
 */
type SearchLinkDescriptor = {
    basePath: string;
    linkText: string;
    routeKey: RouteKey;
};

const filterOnRisk: SearchLinkDescriptor = {
    basePath: riskBasePath,
    linkText: 'Risk',
    routeKey: 'risk',
};

const filterOnViolations: SearchLinkDescriptor = {
    basePath: violationsBasePath,
    linkText: 'Violations',
    routeKey: 'violations',
};

export type SearchResultCategoryMap = Record<SearchResultCategory, SearchResultCategoryDescriptor>;

// Global search route has conditional rendering according to resourceAccessRequirements in routePaths.ts file.
// Therefore update that property if response ever adds search categories.

function getSearchResultCategoryMap(
    isFeatureFlagEnabled: IsFeatureFlagEnabled
): SearchResultCategoryMap {
    const isVm2Ga = isFeatureFlagEnabled('ROX_VULN_MGMT_2_GA');
    return {
        ALERTS: {
            filterOn: null,
            viewLinks: [
                {
                    basePath: `${violationsBasePath}/:id`,
                    linkText: 'Violations',
                    routeKey: 'violations',
                },
            ],
        },
        CLUSTERS: {
            filterOn: null,
            viewLinks: [
                {
                    basePath: `${clustersBasePath}/:id`,
                    linkText: 'Clusters',
                    routeKey: 'clusters',
                },
            ],
        },
        DEPLOYMENTS: {
            filterOn: {
                filterCategory: 'Deployment',
                filterLinks: [filterOnViolations],
            },
            viewLinks: [
                {
                    basePath: `${riskBasePath}/:id`,
                    linkText: 'Risk',
                    routeKey: 'risk',
                },
            ],
        },
        IMAGES: {
            filterOn: {
                filterCategory: 'Image',
                filterLinks: [filterOnViolations, filterOnRisk],
            },
            viewLinks: [
                {
                    basePath: isVm2Ga
                        ? `${vulnerabilitiesWorkloadCvesPath}/images/:id`
                        : `${vulnManagementImagesPath}/:id`,
                    linkText: 'Images',
                    routeKey: 'vulnerability-management',
                },
            ],
        },
        NAMESPACES: {
            filterOn: null,
            viewLinks: [
                {
                    basePath: isVm2Ga
                        ? `${vulnerabilityNamespaceViewPath}${getQueryString({
                              // TODO - Add regex searching support for namespace view ROX-24484 when ROX_VULN_MGMT_2_GA is enabled
                              s: {
                                  NAMESPACE: [':name'],
                                  CLUSTER: [':locationTextForCategory'],
                              },
                          })}`
                        : `${vulnManagementNamespacesPath}/:id`,
                    linkText: 'Vulnerability Management',
                    routeKey: 'vulnerability-management',
                },
            ],
        },
        NODES: {
            filterOn: null,
            viewLinks: [
                {
                    basePath: isVm2Ga
                        ? `${vulnerabilitiesNodeCvesPath}/nodes/:id`
                        : `${vulnManagementNodesPath}/:id`,
                    linkText: 'Vulnerability Management',
                    routeKey: 'vulnerability-management',
                },
            ],
        },
        POLICIES: {
            filterOn: {
                filterCategory: 'Policy',
                filterLinks: [filterOnViolations],
            },
            viewLinks: [
                {
                    basePath: `${policiesBasePath}/:id`,
                    linkText: 'Policies',
                    routeKey: 'policy-management',
                },
            ],
        },
        POLICY_CATEGORIES: {
            filterOn: null,
            viewLinks: [],
        },
        ROLES: {
            filterOn: null,
            viewLinks: [
                {
                    basePath: `${configManagementRolesPath}/:id`,
                    linkText: 'Configuration Management',
                    routeKey: 'configmanagement',
                },
            ],
        },
        ROLEBINDINGS: {
            filterOn: null,
            viewLinks: [],
        },
        SECRETS: {
            filterOn: {
                filterCategory: 'Secret',
                filterLinks: [filterOnRisk],
            },
            viewLinks: [
                {
                    basePath: `${configManagementSecretsPath}/:id`,
                    linkText: 'Configuration Management',
                    routeKey: 'configmanagement',
                },
            ],
        },
        SERVICE_ACCOUNTS: {
            filterOn: null,
            viewLinks: [
                {
                    basePath: `${configManagementServiceAccountsPath}/:id`,
                    linkText: 'Configuration Management',
                    routeKey: 'configmanagement',
                },
            ],
        },
        SUBJECTS: {
            filterOn: null,
            viewLinks: [], // because search result id property value is not the id, but the name
        },
    };
}

// Given isRouteEnabled predicate function from useIsRouteEnabled hook,
// return copy of map with filter and view links only for routes that are enabled.
export function searchResultCategoryMapFilteredIsRouteEnabled(
    isRouteEnabled: IsRouteEnabled,
    isFeatureFlagEnabled: IsFeatureFlagEnabled
): SearchResultCategoryMap {
    const searchResultCategoryMap = getSearchResultCategoryMap(isFeatureFlagEnabled);
    const searchResultCategoryMapFiltered = cloneDeep(searchResultCategoryMap);

    Object.keys(searchResultCategoryMapFiltered).forEach((searchResultKey) => {
        const value = searchResultCategoryMapFiltered[searchResultKey];

        if (value.filterOn) {
            const filterLinks = value.filterOn.filterLinks.filter(({ routeKey }) =>
                isRouteEnabled(routeKey)
            );

            if (filterLinks.length !== 0) {
                value.filterOn.filterLinks = filterLinks;
            } else {
                value.filterOn = null;
            }
        }

        if (value.viewLinks.length !== 0) {
            value.viewLinks = value.viewLinks.filter(({ routeKey }) => isRouteEnabled(routeKey));
        }
    });

    return searchResultCategoryMapFiltered;
}

export type SearchNavCategory = 'SEARCH_UNSET' | SearchResultCategory;

export const searchNavMap: Record<SearchNavCategory, string> = {
    SEARCH_UNSET: 'All results',
    CLUSTERS: 'Clusters',
    DEPLOYMENTS: 'Deployments',
    IMAGES: 'Images',
    NAMESPACES: 'Namespaces',
    NODES: 'Nodes',
    POLICIES: 'Policies',
    POLICY_CATEGORIES: 'Policy categories',
    ROLES: 'Roles',
    ROLEBINDINGS: 'Role bindings',
    SECRETS: 'Secrets',
    SERVICE_ACCOUNTS: 'Service accounts',
    SUBJECTS: 'Users and groups',
    ALERTS: 'Violations',
};
