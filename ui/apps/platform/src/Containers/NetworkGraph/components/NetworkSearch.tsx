import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import { updateSearchFilter } from 'Components/CompoundSearchFilter/utils/utils';

import { useSearchFilter } from '../NetworkGraphURLStateContext';
import { searchFilterConfig } from './searchFilterConfig';

type NetworkSearchProps = {
    isDisabled: boolean;
};

function NetworkSearch({ isDisabled }: NetworkSearchProps) {
    const { searchFilter, setSearchFilter } = useSearchFilter();

    // updateSearchFilter merges the payload into the existing filter, so the Cluster/Namespace/
    // Deployment scope keys written by the scope selectors are preserved automatically.
    return (
        <CompoundSearchFilter
            config={searchFilterConfig}
            defaultEntity="Deployment"
            defaultAttribute="Label"
            searchFilter={searchFilter}
            isDisabled={isDisabled}
            onSearch={(payload) => setSearchFilter(updateSearchFilter(searchFilter, payload))}
        />
    );
}

export default NetworkSearch;
