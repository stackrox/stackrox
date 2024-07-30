import React from 'react';
import { SelectOption } from '@patternfly/react-core';

import { getEntities } from 'Components/CompoundSearchFilter/utils/utils';

import SimpleSelect from './SimpleSelect';
import { CompoundSearchFilterConfig } from '../types';

export type SelectedEntity = string | undefined;
export type EntitySelectorOnChange = (value: string | number | undefined) => void;

export type EntitySelectorProps = {
    selectedEntity: SelectedEntity;
    onChange: EntitySelectorOnChange;
    config: CompoundSearchFilterConfig;
    menuToggleClassName?: string;
};

function EntitySelector({
    selectedEntity,
    onChange,
    config,
    menuToggleClassName,
}: EntitySelectorProps) {
    const entities = getEntities(config);

    if (entities.length === 0) {
        return null;
    }

    const displayName = selectedEntity ? config[selectedEntity]?.displayName : undefined;

    return (
        <SimpleSelect
            menuToggleClassName={menuToggleClassName}
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
