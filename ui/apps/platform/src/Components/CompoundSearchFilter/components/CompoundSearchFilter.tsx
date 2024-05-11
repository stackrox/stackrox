import React, { useEffect, useState } from 'react';
import { Flex } from '@patternfly/react-core';

import {
    CompoundSearchFilterConfig,
    SearchFilterAttributeName,
    SearchFilterEntityName,
} from '../types';
import { getDefaultAttribute, getDefaultEntity } from '../utils/utils';

import EntitySelector from './EntitySelector';
import AttributeSelector from './AttributeSelector';

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
    const [selectedEntity, setSelectedEntity] = useState(() => {
        return defaultEntity ?? getDefaultEntity(config);
    });
    const [selectedAttribute, setSelectedAttribute] = useState(() => {
        return defaultAttribute ?? getDefaultAttribute(getDefaultEntity(config), config);
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
                    setSelectedEntity(value);
                    const defaultAttribute = getDefaultAttribute(value, config);
                    setSelectedAttribute(defaultAttribute);
                }}
                config={config}
            />
            <AttributeSelector
                selectedEntity={selectedEntity}
                selectedAttribute={selectedAttribute}
                onChange={(value) => setSelectedAttribute(value)}
                config={config}
            />
        </Flex>
    );
}

export default CompoundSearchFilter;
