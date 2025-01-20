import React, { ReactElement, useState } from 'react';
import { Flex, SearchInput, SelectOption } from '@patternfly/react-core';
import SimpleSelect from 'Components/CompoundSearchFilter/components/SimpleSelect';

export const matchTypes = ['Equals', 'Not'];

export type MatchType = (typeof matchTypes)[number];

export type IPMatchFilterResult = {
    matchType: MatchType;
    externalIP: string;
};

type IPMatchFilterProps = {
    onSearch: ({ matchType, externalIP }: IPMatchFilterResult) => void;
};

function ensureMatchType(value: unknown): MatchType {
    if (typeof value === 'string' && matchTypes.includes(value)) {
        return value;
    }
    return 'Equals';
}

function IPMatchFilter({ onSearch }: IPMatchFilterProps): ReactElement {
    const [matchType, setMatchType] = useState<MatchType>('Equals');
    const [externalIP, setExternalIP] = useState('');

    return (
        <Flex
            direction={{ default: 'row' }}
            flexWrap={{ default: 'nowrap' }}
            spaceItems={{ default: 'spaceItemsNone' }}
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
                placeholder="Find by external IP"
                value={externalIP}
                onChange={(_event, value) => setExternalIP(value)}
                onSearch={() => {
                    onSearch({ matchType, externalIP });
                }}
                onClear={() => setExternalIP('')}
            />
        </Flex>
    );
}

export default IPMatchFilter;
