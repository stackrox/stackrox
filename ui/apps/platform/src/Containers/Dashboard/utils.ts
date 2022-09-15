import { SearchFilter } from 'types/search';

/**
 * Given a searchFilter, determines whether or not a resource filter is applied to
 * the page.
 */
function isResourceScoped(searchFilter: SearchFilter): boolean {
    return Boolean(searchFilter.Cluster) || Boolean(searchFilter['Namespace ID']);
}

export default isResourceScoped;
