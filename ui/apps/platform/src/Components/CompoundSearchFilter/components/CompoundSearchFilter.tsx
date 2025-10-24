import { useEffect, useState } from 'react';
import { Flex } from '@patternfly/react-core';

import type { SearchFilter } from 'types/search';
import { ensureString } from 'utils/ensure';
import type { CompoundSearchFilterConfig, OnSearchPayload } from '../types';
import { getDefaultAttributeName, getDefaultEntityName } from '../utils/utils';

import EntitySelector from './EntitySelector';
import type { SelectedEntity } from './EntitySelector';
import AttributeSelector from './AttributeSelector';
import type { SelectedAttribute } from './AttributeSelector';
import CompoundSearchFilterInputField from './CompoundSearchFilterInputField';
import type { InputFieldValue } from './CompoundSearchFilterInputField';

export type CompoundSearchFilterProps = {
    config: CompoundSearchFilterConfig;
    defaultEntity?: string;
    defaultAttribute?: string;
    searchFilter: SearchFilter;
    additionalContextFilter?: SearchFilter;
    onSearch: ({ action, category, value }: OnSearchPayload) => void;
};

function CompoundSearchFilter({
    config,
    defaultEntity,
    defaultAttribute,
    searchFilter,
    additionalContextFilter,
    onSearch,
}: CompoundSearchFilterProps) {
    const [selectedEntity, setSelectedEntity] = useState<SelectedEntity>(() => {
        if (defaultEntity) {
            return defaultEntity;
        }
        return getDefaultEntityName(config);
    });

    const [selectedAttribute, setSelectedAttribute] = useState<SelectedAttribute>(() => {
        if (defaultAttribute) {
            return defaultAttribute;
        }
        const defaultEntityName = getDefaultEntityName(config);
        if (!defaultEntityName) {
            return undefined;
        }
        return getDefaultAttributeName(config, defaultEntityName);
    });

    // If the selected entity/attribute is not in the config, use the default entity. This handles the case where the search config
    // changes at runtime while the removed entity is still selected.
    const entityConfig = config.find((entity) => entity.displayName === selectedEntity);
    const currentEntity = entityConfig ? selectedEntity : getDefaultEntityName(config);
    const currentAttribute = entityConfig?.attributes.find(
        ({ displayName }) => displayName === selectedAttribute
    )
        ? selectedAttribute
        : getDefaultAttributeName(config, currentEntity ?? '');

    const [inputValue, setInputValue] = useState<InputFieldValue>('');

    useEffect(() => {
        if (defaultEntity) {
            setSelectedEntity(defaultEntity);
        }
    }, [defaultEntity]);

    useEffect(() => {
        if (defaultAttribute) {
            setSelectedAttribute(defaultAttribute);
        }
    }, [defaultAttribute]);

    return (
        <Flex
            direction={{ default: 'row' }}
            spaceItems={{ default: 'spaceItemsNone' }}
            flexWrap={{ default: 'nowrap' }}
            className="pf-v5-u-w-100"
        >
            <EntitySelector
                menuToggleClassName="pf-v5-u-flex-shrink-0"
                selectedEntity={currentEntity}
                onChange={(value) => {
                    const entityName = ensureString(value);
                    const defaultAttributeName = getDefaultAttributeName(config, entityName);
                    setSelectedEntity(entityName);
                    setSelectedAttribute(defaultAttributeName);
                    setInputValue('');
                }}
                config={config}
            />
            <AttributeSelector
                menuToggleClassName="pf-v5-u-flex-shrink-0"
                selectedEntity={currentEntity}
                selectedAttribute={currentAttribute}
                onChange={(value) => {
                    setSelectedAttribute(ensureString(value));
                    setInputValue('');
                }}
                config={config}
            />
            <CompoundSearchFilterInputField
                selectedEntity={currentEntity}
                selectedAttribute={currentAttribute}
                value={inputValue}
                onChange={(value) => {
                    setInputValue(value);
                }}
                searchFilter={searchFilter}
                additionalContextFilter={additionalContextFilter}
                onSearch={(payload) => {
                    const { action, category, value } = payload;
                    const shouldSearch =
                        (action === 'ADD' &&
                            value !== '' &&
                            !searchFilter?.[category]?.includes(value)) ||
                        (action === 'REMOVE' && value !== '');

                    if (shouldSearch) {
                        onSearch(payload);
                    }
                }}
                config={config}
            />
        </Flex>
    );
}

export default CompoundSearchFilter;
