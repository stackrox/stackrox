import { PolicyCategory } from 'types/policy.proto';
import axios from './instance';
import { Empty } from './types';

const policyCategoriesUrl = '/v1/policycategories';

export function getPolicyCategory(id: string): Promise<PolicyCategory> {
    return axios
        .get<PolicyCategory>(`${policyCategoriesUrl}/${id}`)
        .then((response) => response.data);
}

/*
 * Although the request supports a search query string, UI does not need it.
 */
export function getPolicyCategories(): Promise<PolicyCategory[]> {
    return axios
        .get<{ categories: PolicyCategory[] }>(policyCategoriesUrl)
        .then((response) => response.data.categories);
}

/*
 * The id property of the argument has empty string value.
 * The id property of the response has unique value assigned by backend.
 */
export function postPolicyCategory(policyCategory: PolicyCategory): Promise<PolicyCategory> {
    return axios
        .post<PolicyCategory>(policyCategoriesUrl, policyCategory)
        .then((response) => response.data);
}

export function renamePolicyCategory(id: string, newCategoryName: string): Promise<PolicyCategory> {
    return axios
        .put<PolicyCategory>(policyCategoriesUrl, { id, newCategoryName })
        .then((response) => response.data);
}

export function deletePolicyCategory(id: string): Promise<Empty> {
    return axios.delete<Empty>(`${policyCategoriesUrl}/${id}`).then((response) => response.data);
}
