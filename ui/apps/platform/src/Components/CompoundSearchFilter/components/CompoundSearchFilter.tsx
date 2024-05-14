import React, { useEffect, useState } from 'react';
import { Flex } from '@patternfly/react-core';

import {
    CompoundSearchFilterConfig,
    SearchFilterAttributeName,
    SearchFilterEntityName,
} from '../types';
import { getDefaultAttribute, getDefaultEntity } from '../utils/utils';

import EntitySelector, { SelectedEntity } from './EntitySelector';
import AttributeSelector, { SelectedAttribute } from './AttributeSelector';

export type CompoundSearchFilterProps = {
    config: Partial<CompoundSearchFilterConfig>;
    defaultEntity?: SearchFilterEntityName;
    defaultAttribute?: SearchFilterAttributeName;
};

function CompoundSearchFilter({
    config,
    defaultEntity,
    defaultAttribute,
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
        <Flex spaceItems={{ default: 'spaceItemsNone' }}>
            <EntitySelector
                selectedEntity={selectedEntity}
                onChange={(value) => {
                    setSelectedEntity(value as SearchFilterEntityName);
                    const defaultAttribute = getDefaultAttribute(
                        value as SearchFilterEntityName,
                        config
                    );
                    setSelectedAttribute(defaultAttribute);
                }}
                config={config}
            />
            <AttributeSelector
                selectedEntity={selectedEntity}
                selectedAttribute={selectedAttribute}
                onChange={(value) => setSelectedAttribute(value as SearchFilterAttributeName)}
                config={config}
            />
        </Flex>
    );
}

export default CompoundSearchFilter;
