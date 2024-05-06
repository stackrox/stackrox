import React, { useState } from 'react';
import { Flex } from '@patternfly/react-core';

import { CompoundSearchFilterConfig } from '../types';

import EntitySelector from './EntitySelector';
import AttributeSelector from './AttributeSelector';
import { getDefaultAttribute, getDefaultEntity } from '../utils/utils';

export type CompoundSearchFilterProps = {
    config: Partial<CompoundSearchFilterConfig>;
};

function CompoundSearchFilter({ config }: CompoundSearchFilterProps) {
    const [selectedEntity, setSelectedEntity] = useState(() => {
        return getDefaultEntity(config);
    });
    const [selectedAttribute, setSelectedAttribute] = useState(() => {
        const defaultEntity = getDefaultEntity(config);
        const defaultAttribute = getDefaultAttribute(defaultEntity, config);
        return defaultAttribute;
    });

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
