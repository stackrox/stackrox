import React, { useEffect, useState } from 'react';
import { Split } from '@patternfly/react-core';

import {
    CompoundSearchFilterConfig,
    SearchFilterAttribute,
    SearchFilterAttributeName,
    SearchFilterEntityName,
} from '../types';
import { getDefaultAttribute, getDefaultEntity } from '../utils/utils';

import EntitySelector, { SelectedEntity } from './EntitySelector';
import AttributeSelector, { SelectedAttribute } from './AttributeSelector';
import CompoundSearchFilterInputField, { InputFieldValue } from './CompoundSearchFilterInputField';
import { ConditionNumberProps, conditionMap } from './ConditionNumber';

export type CompoundSearchFilterProps = {
    config: Partial<CompoundSearchFilterConfig>;
    defaultEntity?: SearchFilterEntityName;
    defaultAttribute?: SearchFilterAttributeName;
    onSearch: (searchKey: string, searchValue: string) => void;
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
                    // @TODO: Add search filter value to URL search
                    if (selectedEntity && selectedAttribute) {
                        const entityObject = config[selectedEntity];
                        const attributeObject: SearchFilterAttribute =
                            entityObject?.attributes[selectedAttribute];
                        const { inputType } = attributeObject;
                        let result = '';
                        if (inputType === 'text') {
                            result = value as string;
                        } else if (inputType === 'condition-number') {
                            const { condition, number } = value as ConditionNumberProps['value'];
                            result = `${conditionMap[condition]}${number}`;
                        } else if (inputType === 'autocomplete') {
                            result = value as string;
                        } else if (inputType === 'date-picker') {
                            result = value as string;
                        } else if (inputType === 'select') {
                            const selection = value as string[];
                            result = selection.join(',');
                        }
                        if (result !== '') {
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
