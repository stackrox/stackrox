import useURLParameter from './useURLParameter';

function useURLSearch(keyPrefix = 'search') {
    const [searchFilter, setSearchFilter] = useURLParameter(keyPrefix, {});
    return { searchFilter, setSearchFilter };
}

export default useURLSearch;
