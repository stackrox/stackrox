import React, { ReactElement, useState } from 'react';
import { Flex, SearchInput, SelectOption } from '@patternfly/react-core';

import SimpleSelect from 'Components/CompoundSearchFilter/components/SimpleSelect';
import { onURLSearch } from 'Components/CompoundSearchFilter/utils/utils';
import { SetSearchFilter } from 'hooks/useURLSearch';
import { SearchFilter } from 'types/search';
import { isValidCidrBlock } from 'utils/urlUtils';

import { EXTERNAL_SOURCE_ADDRESS_QUERY } from '../NetworkGraph.constants';

export const matchTypes = ['Equals', 'Not'];

export type MatchType = (typeof matchTypes)[number];

export type IPMatchFilterResult = {
    matchType: MatchType;
    externalIP: string;
};

type IPMatchFilterProps = {
    searchFilter: SearchFilter;
    setSearchFilter: SetSearchFilter;
};

function ensureMatchType(value: unknown): MatchType {
    if (typeof value === 'string' && matchTypes.includes(value)) {
        return value;
    }
    return 'Equals';
}

function IPMatchFilter({ searchFilter, setSearchFilter }: IPMatchFilterProps): ReactElement {
    const [matchType, setMatchType] = useState<MatchType>('Equals');
    const [externalIP, setExternalIP] = useState('');

    function handleClear() {
        setExternalIP('');
    }

    function handleSearch(ipAddress: string) {
        // this will only work if ipv4 is reported, will need to check if ipv4 or ipv6 and add /128 for ipv6
        const searchValue = isValidCidrBlock(`${ipAddress}/32`) ? `${ipAddress}/32` : ipAddress;

        onURLSearch(searchFilter, setSearchFilter, {
            action: 'ADD',
            category: EXTERNAL_SOURCE_ADDRESS_QUERY,
            value: searchValue,
        });

        setExternalIP('');
    }

    return (
        <Flex
            direction={{ default: 'row' }}
            flexWrap={{ default: 'nowrap' }}
            spaceItems={{ default: 'spaceItemsNone' }}
            className="pf-v5-u-w-100"
        >
            <SimpleSelect
                menuToggleClassName="pf-v5-u-flex-shrink-0"
                value={matchType}
                onChange={(value) => setMatchType(ensureMatchType(value))}
                ariaLabelMenu="external ip comparison selector menu"
                ariaLabelToggle="external ip comparison selector toggle"
            >
                <SelectOption key="Equals" value="Equals">
                    Equals
                </SelectOption>
                <SelectOption key="Not" value="Not">
                    Not
                </SelectOption>
            </SimpleSelect>
            <SearchInput
                placeholder="Find by IP or IP/CIDR"
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
