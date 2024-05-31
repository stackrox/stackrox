import React, { useEffect, useState } from 'react';
import { Split } from '@patternfly/react-core';

import { DeepPartial } from 'utils/type.utils';
import { SearchFilter } from 'types/search';
import {
    CompoundSearchFilterConfig,
    SearchFilterAttributeName,
    SearchFilterEntityName,
} from '../types';
import { getDefaultAttribute, getDefaultEntity, getEntityAttributeNames } from '../utils/utils';

import EntitySelector, { SelectedEntity } from './EntitySelector';
import AttributeSelector, { SelectedAttribute } from './AttributeSelector';
import CompoundSearchFilterInputField, { InputFieldValue } from './CompoundSearchFilterInputField';

export type OnSearchPayload = {
    action: 'ADD' | 'REMOVE';
    category: string;
    value: string;
};

export type CompoundSearchFilterProps = {
    config: DeepPartial<CompoundSearchFilterConfig>;
    defaultEntity?: SearchFilterEntityName;
    defaultAttribute?: SearchFilterAttributeName;
    searchFilter: SearchFilter;
    onSearch: ({ action, category, value }: OnSearchPayload) => void;
};

function CompoundSearchFilter({
    config,
    defaultEntity,
    defaultAttribute,
    searchFilter,
    onSearch,
}: CompoundSearchFilterProps) {
    const [selectedEntity, setSelectedEntity] = useState<SelectedEntity>(() => {
        if (defaultEntity) {
            return defaultEntity;
        }
        return getDefaultEntity(config);
    });

    const [selectedAttribute, setSelectedAttribute] = useState<SelectedAttribute>(() => {
        if (defaultAttribute) {
            return defaultAttribute;
        }
        const defaultEntity = getDefaultEntity(config);
        if (!defaultEntity) {
            return undefined;
        }
        return getDefaultAttribute(defaultEntity, config);
    });

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

    const hasMultipleAttributes = selectedEntity
        ? getEntityAttributeNames(selectedEntity, config).length > 1
        : false;

    return (
        <Split className="pf-v5-u-flex-grow-1">
            <EntitySelector
                selectedEntity={selectedEntity}
                onChange={(value) => {
                    setSelectedEntity(value as SearchFilterEntityName);
                    const defaultAttribute = getDefaultAttribute(
                        value as SearchFilterEntityName,
                        config
                    );
                    setSelectedAttribute(defaultAttribute);
                    setInputValue('');
                }}
                config={config}
            />
            {hasMultipleAttributes && (
                <AttributeSelector
                    selectedEntity={selectedEntity}
                    selectedAttribute={selectedAttribute}
                    onChange={(value) => {
                        setSelectedAttribute(value as SearchFilterAttributeName);
                        setInputValue('');
                    }}
                    config={config}
                />
            )}
            <CompoundSearchFilterInputField
                selectedEntity={selectedEntity}
                selectedAttribute={selectedAttribute}
                value={inputValue}
                onChange={(value) => {
                    setInputValue(value);
                }}
                searchFilter={searchFilter}
                onSearch={onSearch}
                config={config}
            />
        </Split>
    );
}

export default CompoundSearchFilter;
