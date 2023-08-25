import qs from 'qs';
import { SearchEntry } from 'types/search';
import axios from './instance';
import { CancellableRequest, makeCancellableAxiosRequest } from './cancellationUtils';

const baseUrl = '/v1/search';
const autoCompleteURL = `${baseUrl}/autocomplete`;

// Add strings in alphabetical order with the exception of SEARCH_UNSET.
// Semicolon on separate line following the strings prevents an extra changed line to add a string at the end.
// prettier-ignore
export type SearchCategory =
    | 'SEARCH_UNSET'
    | 'ACTIVE_COMPONENT'
    | 'ALERTS'
    | 'CLUSTER_HEALTH'
    | 'CLUSTER_VULN_EDGE'
    | 'CLUSTER_VULNERABILITIES'
    | 'CLUSTERS'
    | 'COMPLIANCE'
    | 'COMPLIANCE_CONTROL'
    | 'COMPLIANCE_CONTROL_GROUP'
    | 'COMPLIANCE_DOMAIN'
    | 'COMPLIANCE_METADATA'
    | 'COMPLIANCE_RESULTS'
    | 'COMPLIANCE_STANDARD'
    | 'COMPONENT_VULN_EDGE'
    | 'DEPLOYMENTS'
    | 'IMAGE_COMPONENTS'
    | 'IMAGE_COMPONENT_EDGE'
    | 'IMAGE_VULN_EDGE'
    | 'IMAGE_VULNERABILITIES'
    | 'IMAGES'
    | 'NAMESPACES'
    | 'NETWORK_BASELINE'
    | 'NETWORK_ENTITY'
    | 'NETWORK_POLICIES'
    | 'NODE_COMPONENT_CVE_EDGE'
    | 'NODE_COMPONENT_EDGE'
    | 'NODE_COMPONENTS'
    | 'NODE_VULN_EDGE'
    | 'NODE_VULNERABILITIES'
    | 'NODES'
    | 'PODS'
    | 'POLICIES'
    | 'POLICY_CATEGORIES'
    | 'PROCESS_BASELINE_RESULTS'
    | 'PROCESS_BASELINES'
    | 'PROCESS_INDICATORS'
    | 'REPORT_CONFIGURATIONS'
    | 'RISKS'
    | 'ROLEBINDINGS'
    | 'ROLES'
    | 'SECRETS'
    | 'SERVICE_ACCOUNTS'
    | 'SUBJECTS'
    | 'VULN_REQUEST'
    | 'VULNERABILITIES'
    ;

// RawSearchRequest is used to scope a given search in a specific category.
// The search categories could be deployments, policies, images etc.
export type RawSearchRequest = {
    query: string;
    categories?: SearchCategory[];
};

export type SearchResultCategory =
    | 'ALERTS'
    | 'CLUSTERS'
    | 'DEPLOYMENTS'
    | 'IMAGES'
    | 'NAMESPACES'
    | 'NODES'
    | 'POLICIES'
    | 'POLICY_CATEGORIES'
    | 'ROLES'
    | 'ROLEBINDINGS'
    | 'SECRETS'
    | 'SERVICE_ACCOUNTS'
    | 'SUBJECTS';

export type SearchResult = {
    id: string;
    name: string;
    category: SearchResultCategory;
    fieldToMatches: Record<string, SearchResultMatches>;
    score: number; // double
    // Location is intended to be a unique, yet human readable,
    // identifier for the result. For example, for a deployment,
    // the location will be "$cluster_name/$namespace/$deployment_name.
    // It is displayed in the UI in the global search results, underneath
    // the name for each result.
    location: string;
};

export type SearchResultMatches = {
    values: string[];
};

export type SearchCategoryCount = {
    category: SearchResultCategory;
    count: string; // int64
};

export type SearchResponse = {
    results: SearchResult[];
    counts: SearchCategoryCount[];
};

type SearchOptionsResponse = { options: string[] };
type AutocompleteResponse = { values: string[] };

/**
 * Fetches search options
 */
export function fetchOptions(query = ''): Promise<SearchEntry[]> {
    return axios.get<SearchOptionsResponse>(`${baseUrl}/metadata/options?${query}`).then(
        (response) =>
            response?.data?.options?.map((option) => ({
                value: `${option}:`,
                label: `${option}:`,
                type: 'categoryOption',
            })) ?? []
    );
}

/*
 * Get search options for category.
 */
export function getSearchOptionsForCategory(
    searchCategory?: SearchCategory
): CancellableRequest<string[]> {
    const queryString = searchCategory ? `categories=${searchCategory}` : '';
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<SearchOptionsResponse>(`${baseUrl}/metadata/options?${queryString}`, {
                signal,
            })
            .then((response) => response?.data?.options ?? [])
    );
}

/**
 * Fetches search results
 */
export function fetchGlobalSearchResults(
    rawSearchRequest: RawSearchRequest
): Promise<SearchResponse> {
    const params = qs.stringify(rawSearchRequest, { arrayFormat: 'repeat' });
    return axios.get<SearchResponse>(`${baseUrl}?${params}`).then((response) => response.data);
}

// Fetches the autocomplete response.
export function fetchAutoCompleteResults(rawSearchRequest: RawSearchRequest): Promise<string[]> {
    const params = qs.stringify(rawSearchRequest, { arrayFormat: 'repeat' });
    return axios
        .get<AutocompleteResponse>(`${autoCompleteURL}?${params}`)
        .then((response) => response?.data?.values || []);
}
