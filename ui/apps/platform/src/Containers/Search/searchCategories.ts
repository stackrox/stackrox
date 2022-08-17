import { SearchResultCategory } from 'services/SearchService';
import { ResourceName } from 'types/roleResources';
import {
    clustersBasePath,
    configManagementRolesPath,
    configManagementSecretsPath,
    configManagementServiceAccountsPath,
    policiesBasePath,
    riskBasePath,
    violationsBasePath,
    vulnManagementImagesPath,
    vulnManagementNamespacesPath,
    vulnManagementNodesPath,
} from 'routePaths';

type SearchResultCategoryDescriptor = {
    resourceName: ResourceName;
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
};

const filterOnRisk = {
    basePath: riskBasePath,
    linkText: 'Risk',
};

const filterOnViolations = {
    basePath: violationsBasePath,
    linkText: 'Violations',
};

// prettier-ignore
export const searchResultCategoryMap: Record<
    SearchResultCategory,
    SearchResultCategoryDescriptor
> = {
    ALERTS: {
        resourceName: 'Alert',
        filterOn: null,
        viewLinks: [
            {
                basePath: violationsBasePath,
                linkText: 'Violations',
            },
        ],
    },
    CLUSTERS: {
        resourceName: 'Cluster',
        filterOn: null,
        viewLinks: [
            {
                basePath: clustersBasePath,
                linkText: 'Clusters',
            },
        ],
    },
    DEPLOYMENTS: {
        resourceName: 'Deployment',
        filterOn: {
            filterCategory: 'Deployment',
            filterLinks: [filterOnViolations],
        },
        viewLinks: [
            {
                basePath: riskBasePath,
                linkText: 'Risk',
            },
        ],
    },
    IMAGES: {
        resourceName: 'Image',
        filterOn: {
            filterCategory: 'Image',
            filterLinks: [filterOnViolations, filterOnRisk],
        },
        viewLinks: [
            {
                basePath: vulnManagementImagesPath,
                linkText: 'Images',
            },
        ],
    },
    NAMESPACES: {
        resourceName: 'Namespace',
        filterOn: null,
        viewLinks: [
            {
                basePath: vulnManagementNamespacesPath,
                linkText: 'Vulnerability Management',
            },
        ],
    },
    NODES: {
        resourceName: 'Node',
        filterOn: null,
        viewLinks: [
            {
                basePath: vulnManagementNodesPath,
                linkText: 'Vulnerability Management',
            },
        ],
    },
    POLICIES: {
        resourceName: 'Policy',
        filterOn: {
            filterCategory: 'Policy',
            filterLinks: [filterOnViolations],
        },
        viewLinks: [
            {
                basePath: policiesBasePath,
                linkText: 'Policies',
            },
        ],
    },
    ROLES: {
        resourceName: 'K8sRole',
        filterOn: null,
        viewLinks: [
            {
                basePath: configManagementRolesPath,
                linkText: 'Configuration Management',
            },
        ],
    },
    ROLEBINDINGS: {
        resourceName: 'K8sRoleBinding',
        filterOn: null,
        viewLinks: [],
    },
    SECRETS: {
        resourceName: 'Secret',
        filterOn: {
            filterCategory: 'Secret',
            filterLinks: [filterOnRisk],
        },
        viewLinks: [
            {
                basePath: configManagementSecretsPath,
                linkText: 'Configuration Management',
            },
        ],
    },
    SERVICE_ACCOUNTS: {
        resourceName: 'ServiceAccount',
        filterOn: null,
        viewLinks: [
            {
                basePath: configManagementServiceAccountsPath,
                linkText: 'Configuration Management',
            },
        ],
    },
    SUBJECTS: {
        resourceName: 'K8sSubject',
        filterOn: null,
        viewLinks: [], // because search result id property value is not the id, but the name
    },
};

export type SearchNavCategory = 'SEARCH_UNSET' | SearchResultCategory;

export const searchNavMap: Record<SearchNavCategory, string> = {
    SEARCH_UNSET: 'All results',
    CLUSTERS: 'Clusters',
    DEPLOYMENTS: 'Deployments',
    IMAGES: 'Images',
    NAMESPACES: 'Namespaces',
    NODES: 'Nodes',
    POLICIES: 'Policies',
    ROLES: 'Roles',
    ROLEBINDINGS: 'Role bindings',
    SECRETS: 'Secrets',
    SERVICE_ACCOUNTS: 'Service accounts',
    SUBJECTS: 'Users and groups',
    ALERTS: 'Violations',
};
