import { useState } from 'react';
import { Flex } from '@patternfly/react-core';

import type { SearchFilter } from 'types/search';
import { ensureString } from 'utils/ensure';
import { isOnSearchPayload } from '../types';
import type { CompoundSearchFilterConfig, OnSearchCallback } from '../types';
import {
    getAttributeFromEntity,
    getEntityFromConfig,
    payloadItemFiltererForUpdating,
} from '../utils/utils';

import EntitySelector from './EntitySelector';
import type { SelectedEntity } from './EntitySelector';
import AttributeSelector from './AttributeSelector';
import type { SelectedAttribute } from './AttributeSelector';
import CompoundSearchFilterInputField from './CompoundSearchFilterInputField';

export type CompoundSearchFilterProps = {
    config: CompoundSearchFilterConfig;
    defaultEntity?: string;
    defaultAttribute?: string;
    searchFilter: SearchFilter;
    additionalContextFilter?: SearchFilter;
    onSearch: OnSearchCallback;
};

function CompoundSearchFilter({
    config,
    defaultEntity,
    defaultAttribute = 'Name',
    searchFilter,
    additionalContextFilter,
    onSearch,
}: CompoundSearchFilterProps) {
    const [selectedEntity, setSelectedEntity] = useState<SelectedEntity>(undefined);
    const [selectedAttribute, setSelectedAttribute] = useState<SelectedAttribute>(undefined);

    const entity = getEntityFromConfig(config, selectedEntity, defaultEntity);
    const attribute = getAttributeFromEntity(entity, selectedAttribute, defaultAttribute);

    return (
        <Flex
            direction={{ default: 'row' }}
            spaceItems={{ default: 'spaceItemsNone' }}
            flexWrap={{ default: 'nowrap' }}
            className="pf-v5-u-w-100"
        >
            {entity && (
                <EntitySelector
                    menuToggleClassName="pf-v5-u-flex-shrink-0"
                    entity={entity}
                    onChange={(value) => {
                        setSelectedEntity(ensureString(value));
                        setSelectedAttribute(undefined);
                    }}
                    config={config}
                />
            )}
            {entity &&
                Array.isArray(entity.attributes) &&
                entity.attributes.length !== 0 &&
                attribute && (
                    <AttributeSelector
                        menuToggleClassName="pf-v5-u-flex-shrink-0"
                        attributes={entity.attributes}
                        attribute={attribute}
                        onChange={(value) => {
                            setSelectedAttribute(ensureString(value));
                        }}
                    />
                )}
            {entity && attribute && (
                <CompoundSearchFilterInputField
                    // Change in key causes React to instantiate a new input element,
                    // which has side effect to clear input state if same type as previous element.
                    key={`${entity.displayName} ${attribute.displayName}`}
                    entity={entity}
                    attribute={attribute}
                    searchFilter={searchFilter}
                    additionalContextFilter={additionalContextFilter}
                    onSearch={(payload) => {
                        // TODO What is pro and con for search filter input field to prevent empty string and filter?
                        const payloadFiltered = payload.filter((payloadItem) =>
                            payloadItemFiltererForUpdating(searchFilter, payloadItem)
                        );

                        if (isOnSearchPayload(payloadFiltered)) {
                            onSearch(payloadFiltered);
                        }
                    }}
                />
            )}
        </Flex>
    );
}

export default CompoundSearchFilter;
