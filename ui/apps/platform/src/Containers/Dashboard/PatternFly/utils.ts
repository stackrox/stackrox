import { SearchFilter } from 'types/search';

function isResourceScoped(searchFilter: SearchFilter): boolean {
    return Boolean(searchFilter.Cluster) || Boolean(searchFilter['Namespace ID']);
}

export default isResourceScoped;
