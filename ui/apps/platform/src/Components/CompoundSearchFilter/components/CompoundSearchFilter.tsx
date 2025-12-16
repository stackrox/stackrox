import { useEffect, useState } from 'react';
import { Flex } from '@patternfly/react-core';

import type { SearchFilter } from 'types/search';
import { ensureString } from 'utils/ensure';
import { isOnSearchPayload } from '../types';
import type { CompoundSearchFilterConfig, OnSearchCallback } from '../types';
import {
    getAttribute,
    getDefaultAttributeName,
    getDefaultEntityName,
    getEntity,
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

    const entity = getEntity(config, currentEntity ?? '');
    const attribute = getAttribute(config, currentEntity ?? '', currentAttribute ?? '');

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
                }}
                config={config}
            />
            <AttributeSelector
                menuToggleClassName="pf-v5-u-flex-shrink-0"
                selectedEntity={currentEntity}
                selectedAttribute={currentAttribute}
                onChange={(value) => {
                    setSelectedAttribute(ensureString(value));
                }}
                config={config}
            />
            {entity && attribute && (
                <CompoundSearchFilterInputField
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
