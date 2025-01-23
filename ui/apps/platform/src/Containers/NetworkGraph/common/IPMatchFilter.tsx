import React, { ReactElement } from 'react';
import { Flex, SearchInput, SelectOption } from '@patternfly/react-core';
import SimpleSelect from 'Components/CompoundSearchFilter/components/SimpleSelect';

export const matchTypes = ['Equals', 'Not'];

export type MatchType = (typeof matchTypes)[number];

export type IPMatchFilterResult = {
    matchType: MatchType;
    externalIP: string;
};

type IPMatchFilterProps = {
    filter: IPMatchFilterResult;
    onChange: ({ matchType, externalIP }: IPMatchFilterResult) => void;
    onSearch: ({ matchType, externalIP }: IPMatchFilterResult) => void;
    onClear: () => void;
};

function ensureMatchType(value: unknown): MatchType {
    if (typeof value === 'string' && matchTypes.includes(value)) {
        return value;
    }
    return 'Equals';
}

function IPMatchFilter({ filter, onChange, onSearch, onClear }: IPMatchFilterProps): ReactElement {
    return (
        <Flex
            direction={{ default: 'row' }}
            flexWrap={{ default: 'nowrap' }}
            spaceItems={{ default: 'spaceItemsNone' }}
        >
            <SimpleSelect
                menuToggleClassName="pf-v5-u-flex-shrink-0"
                value={filter.matchType}
                onChange={(value) => {
                    onChange({ ...filter, matchType: ensureMatchType(value) });
                }}
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
                value={filter.externalIP}
                onChange={(_event, value) => {
                    onChange({ ...filter, externalIP: value });
                }}
                onSearch={() => {
                    onSearch({ ...filter });
                }}
                onClear={onClear}
            />
        </Flex>
    );
}

export default IPMatchFilter;
