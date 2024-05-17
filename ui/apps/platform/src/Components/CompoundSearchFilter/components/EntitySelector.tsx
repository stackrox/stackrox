import React from 'react';
import { SelectOption } from '@patternfly/react-core';

import {
    CompoundSearchFilterConfig,
    SearchFilterEntityName,
} from 'Components/CompoundSearchFilter/types';
import { getEntities } from 'Components/CompoundSearchFilter/utils/utils';

import SimpleSelect from './SimpleSelect';

export type SelectedEntity = SearchFilterEntityName | undefined;
export type EntitySelectorOnChange = (value: string | number | undefined) => void;

export type EntitySelectorProps = {
    selectedEntity: SelectedEntity;
    onChange: EntitySelectorOnChange;
    config: Partial<CompoundSearchFilterConfig>;
};

function EntitySelector({ selectedEntity, onChange, config }: EntitySelectorProps) {
    const entities = getEntities(config);

    if (entities.length === 0) {
        return null;
    }

    const displayName = selectedEntity ? config[selectedEntity]?.displayName : undefined;

    return (
        <SimpleSelect
            value={displayName}
            onChange={onChange}
            ariaLabelMenu="compound search filter entity selector menu"
            ariaLabelToggle="compound search filter entity selector toggle"
        >
            {entities.map((entity) => {
                const displayName = config[entity]?.displayName;
                return (
                    <SelectOption key={entity} value={entity}>
                        {displayName}
                    </SelectOption>
                );
            })}
        </SimpleSelect>
    );
}

export default EntitySelector;
