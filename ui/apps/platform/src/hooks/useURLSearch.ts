import { SearchFilter } from 'types/search';
import useURLParameter from './useURLParameter';

function useURLSearch(keyPrefix = 'search') {
    const [searchFilter, setSearchFilter] = useURLParameter<SearchFilter>(keyPrefix, {});
    return { searchFilter, setSearchFilter };
}

export default useURLSearch;
