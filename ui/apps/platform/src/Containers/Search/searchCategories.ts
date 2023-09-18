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
} from 'routePaths';

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

// prettier-ignore
const searchResultCategoryMap: SearchResultCategoryMap = {
    ALERTS: {
        filterOn: null,
        viewLinks: [
            {
                basePath: violationsBasePath,
                linkText: 'Violations',
                routeKey: 'violations',
            },
        ],
    },
    CLUSTERS: {
        filterOn: null,
        viewLinks: [
            {
                basePath: clustersBasePath,
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
                basePath: riskBasePath,
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
                basePath: vulnManagementImagesPath,
                linkText: 'Images',
                routeKey: 'vulnerability-management',
            },
        ],
    },
    NAMESPACES: {
        filterOn: null,
        viewLinks: [
            {
                basePath: vulnManagementNamespacesPath,
                linkText: 'Vulnerability Management',
                routeKey: 'vulnerability-management',
            },
        ],
    },
    NODES: {
        filterOn: null,
        viewLinks: [
            {
                basePath: vulnManagementNodesPath,
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
                basePath: policiesBasePath,
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
                basePath: configManagementRolesPath,
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
                basePath: configManagementSecretsPath,
                linkText: 'Configuration Management',
                routeKey: 'configmanagement',
            },
        ],
    },
    SERVICE_ACCOUNTS: {
        filterOn: null,
        viewLinks: [
            {
                basePath: configManagementServiceAccountsPath,
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

// Given isRouteEnabled predicate function from useIsRouteEnabled hook,
// return copy of map with filter and view links only for routes that are enabled.
export function searchResultCategoryMapFilteredIsRouteEnabled(
    isRouteEnabled: IsRouteEnabled
): SearchResultCategoryMap {
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
