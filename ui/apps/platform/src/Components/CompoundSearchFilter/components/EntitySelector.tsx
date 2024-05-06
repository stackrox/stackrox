import React from 'react';
import { SelectOption } from '@patternfly/react-core';

import { CompoundSearchFilterConfig } from 'Components/CompoundSearchFilter/types';
import { getEntities } from 'Components/CompoundSearchFilter/utils/utils';

import SimpleSelect from './SimpleSelect';

export type EntitySelectorProps = {
    selectedEntity: string;
    onChange: (value) => void;
    config: Partial<CompoundSearchFilterConfig>;
};

function EntitySelector({ selectedEntity, onChange, config }: EntitySelectorProps) {
    const entities = getEntities(config);

    if (entities.length === 0) {
        return null;
    }

    return (
        <SimpleSelect
            id="compound-search-entity-selector"
            value={selectedEntity}
            onChange={onChange}
            ariaLabelMenu="compound search filter entity selector menu"
            ariaLabelToggle="compound search filter entity selector toggle"
        >
            {entities.map((entity) => {
                return (
                    <SelectOption key={entity} value={entity}>
                        {entity}
                    </SelectOption>
                );
            })}
        </SimpleSelect>
    );
}

export default EntitySelector;
