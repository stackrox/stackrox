import React, { useState } from 'react';
import { Flex } from '@patternfly/react-core';

import { SelectOption } from '@patternfly/react-core/deprecated';
import { CompoundSearchFilterConfig } from '../types';

import EntitySelector from './EntitySelector';

export type CompoundSearchFilterProps = {
    config: CompoundSearchFilterConfig;
};

function CompoundSearchFilter({ config }: CompoundSearchFilterProps) {
    const entities = Object.keys(config);

    const [selectedEntity, setSelectedEntity] = useState(() => {
        return entities[0];
    });

    return (
        <Flex>
            {entities.length > 0 && (
                <EntitySelector
                    value={selectedEntity}
                    onChange={(value) => setSelectedEntity(value)}
                >
                    {entities.map((entity) => {
                        return (
                            <SelectOption key={entity} value={entity}>
                                {entity}
                            </SelectOption>
                        );
                    })}
                </EntitySelector>
            )}
        </Flex>
    );
}

export default CompoundSearchFilter;
