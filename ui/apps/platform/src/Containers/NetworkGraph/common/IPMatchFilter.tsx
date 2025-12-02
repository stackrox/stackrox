import { useState } from 'react';
import type { ReactElement } from 'react';
import { Flex, SearchInput } from '@patternfly/react-core';

import { updateSearchFilter } from 'Components/CompoundSearchFilter/utils/utils';
import type { GenericSearchFilterAttribute } from 'Components/CompoundSearchFilter/types';
import type { SetSearchFilter } from 'hooks/useURLSearch';
import type { SearchFilter } from 'types/search';
import { isValidCidrBlock } from 'utils/urlUtils';

export const matchTypes = ['Equals', 'Not'];

export type MatchType = (typeof matchTypes)[number];

type IPMatchFilterProps = {
    attribute: GenericSearchFilterAttribute;
    searchFilter: SearchFilter;
    setSearchFilter: SetSearchFilter;
};

function IPMatchFilter({
    attribute,
    searchFilter,
    setSearchFilter,
}: IPMatchFilterProps): ReactElement {
    const [externalIP, setExternalIP] = useState('');

    const { filterChipLabel, searchTerm: category } = attribute;
    const textLabel = `Filter results by ${filterChipLabel}`; // consistent with SearchFilterText

    function handleClear() {
        setExternalIP('');
    }

    function handleSearch(ipAddress: string) {
        // this will only work if ipv4 is reported, will need to check if ipv4 or ipv6 and add /128 for ipv6
        const searchValue = isValidCidrBlock(`${ipAddress}/32`) ? `${ipAddress}/32` : ipAddress;

        setSearchFilter(
            updateSearchFilter(searchFilter, [
                {
                    action: 'APPEND',
                    category,
                    value: searchValue,
                },
            ])
        );

        setExternalIP('');
    }

    return (
        <Flex
            direction={{ default: 'row' }}
            flexWrap={{ default: 'nowrap' }}
            spaceItems={{ default: 'spaceItemsNone' }}
            className="pf-v6-u-w-100"
        >
            <SearchInput
                aria-label={textLabel}
                placeholder={textLabel}
                value={externalIP}
                onChange={(_event, value) => setExternalIP(value)}
                onSearch={() => {
                    handleSearch(externalIP);
                }}
                onClear={handleClear}
            />
        </Flex>
    );
}

export default IPMatchFilter;
