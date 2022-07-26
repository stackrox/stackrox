import {
    configManagementPath,
    networkBasePath,
    policiesBasePath,
    riskBasePath,
    violationsBasePath,
    vulnManagementImagesPath,
} from 'routePaths';
import { SearchCategory } from 'services/SearchService';

type SearchCategoryDescriptor = {
    filterCategory: string; // label and value in SearchEntry object which has type: 'categoryOption'
    filterOn: SearchLinkDescriptor[];
    viewOn: SearchLinkDescriptor[];
};

/*
 * A filter link appends ?queryString which includes filterCategory and name from SearchResult.
 * A view link appends /id from SearchResult.
 */
type SearchLinkDescriptor = {
    basePath: string;
    linkText: string;
};

const searchCategoryDescriptorMap: Partial<Record<SearchCategory, SearchCategoryDescriptor>> = {
    ALERTS: {
        filterCategory: 'Policy',
        filterOn: [],
        viewOn: [
            {
                basePath: violationsBasePath,
                linkText: 'Violations',
            },
        ],
    },
    DEPLOYMENTS: {
        filterCategory: 'Deployment',
        filterOn: [
            {
                basePath: violationsBasePath,
                linkText: 'Violations',
            },
            {
                basePath: networkBasePath,
                linkText: 'Network',
            },
        ],
        viewOn: [
            {
                basePath: riskBasePath,
                linkText: 'Risk',
            },
        ],
    },
    IMAGES: {
        filterCategory: 'Image',
        filterOn: [
            {
                basePath: riskBasePath,
                linkText: 'Risk',
            },
            {
                basePath: violationsBasePath,
                linkText: 'Violations',
            },
        ],
        viewOn: [
            {
                basePath: vulnManagementImagesPath,
                linkText: 'Images',
            },
        ],
    },
    POLICIES: {
        filterCategory: 'Policy',
        filterOn: [
            {
                basePath: violationsBasePath,
                linkText: 'Violations',
            },
        ],
        viewOn: [
            {
                basePath: policiesBasePath,
                linkText: 'Policies',
            },
        ],
    },
    SECRETS: {
        filterCategory: 'Secret',
        filterOn: [
            {
                basePath: riskBasePath,
                linkText: 'Risk',
            },
        ],
        viewOn: [
            {
                basePath: `${configManagementPath}/secrets`,
                linkText: 'Secrets',
            },
        ],
    },
};

export default searchCategoryDescriptorMap;
