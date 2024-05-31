import React, { useEffect, useState } from 'react';
import { Split } from '@patternfly/react-core';

import {
    CompoundSearchFilterConfig,
    SearchFilterAttribute,
    SearchFilterAttributeName,
    SearchFilterEntityName,
} from '../types';
import {
    ensureConditionNumber,
    ensureString,
    ensureStringArray,
    getDefaultAttribute,
    getDefaultEntity,
} from '../utils/utils';

import { conditionMap } from './ConditionNumber';
import EntitySelector, { SelectedEntity } from './EntitySelector';
import AttributeSelector, { SelectedAttribute } from './AttributeSelector';
import CompoundSearchFilterInputField, { InputFieldValue } from './CompoundSearchFilterInputField';

export type CompoundSearchFilterProps = {
    config: Partial<CompoundSearchFilterConfig>;
    defaultEntity?: SearchFilterEntityName;
    defaultAttribute?: SearchFilterAttributeName;
    onSearch: (searchKey: string, searchValue: string | string[]) => void;
};

function CompoundSearchFilter({
    config,
    defaultEntity,
    defaultAttribute,
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
            <AttributeSelector
                selectedEntity={selectedEntity}
                selectedAttribute={selectedAttribute}
                onChange={(value) => {
                    setSelectedAttribute(value as SearchFilterAttributeName);
                    setInputValue('');
                }}
                config={config}
            />
            <CompoundSearchFilterInputField
                selectedEntity={selectedEntity}
                selectedAttribute={selectedAttribute}
                value={inputValue}
                onChange={(value) => {
                    setInputValue(value);
                }}
                onSearch={(value) => {
                    if (selectedEntity && selectedAttribute) {
                        const entityObject = config[selectedEntity];
                        const attributeObject: SearchFilterAttribute =
                            entityObject?.attributes[selectedAttribute];
                        const { inputType } = attributeObject;

                        let result: string | string[] = '';

                        if (inputType === 'text') {
                            result = ensureString(value);
                        } else if (inputType === 'condition-number') {
                            const { condition, number } = ensureConditionNumber(value);
                            result = `${conditionMap[condition]}${number}`;
                        } else if (inputType === 'autocomplete') {
                            result = ensureString(value);
                        } else if (inputType === 'date-picker') {
                            result = ensureString(value);
                        } else if (inputType === 'select') {
                            const selection = ensureStringArray(value);
                            result = selection;
                        }

                        if ((Array.isArray(result) && result.length > 0) || result !== '') {
                            // eslint-disable-next-line no-alert
                            onSearch(attributeObject.searchTerm, result);
                        }
                    }
                }}
                config={config}
            />
        </Split>
    );
}

export default CompoundSearchFilter;
