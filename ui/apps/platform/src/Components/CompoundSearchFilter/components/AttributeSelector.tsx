import React from 'react';
import { SelectOption } from '@patternfly/react-core';

import { getEntityAttributes } from 'Components/CompoundSearchFilter/utils/utils';

import SimpleSelect from './SimpleSelect';
import type { SelectedEntity } from './EntitySelector';
import type { CompoundSearchFilterConfig } from '../types';

export type SelectedAttribute = string | undefined;
export type AttributeSelectorOnChange = (value: string | number | undefined) => void;

export type AttributeSelectorProps = {
    selectedEntity: SelectedEntity;
    selectedAttribute: SelectedAttribute;
    onChange: AttributeSelectorOnChange;
    config: CompoundSearchFilterConfig;
    menuToggleClassName?: string;
};

function AttributeSelector({
    selectedEntity = '',
    selectedAttribute = '',
    onChange,
    config,
    menuToggleClassName,
}: AttributeSelectorProps) {
    const entityAttributes = getEntityAttributes(config, selectedEntity);

    if (entityAttributes.length === 0) {
        return null;
    }

    return (
        <SimpleSelect
            menuToggleClassName={menuToggleClassName}
            value={selectedAttribute}
            onChange={onChange}
            ariaLabelMenu="compound search filter attribute selector menu"
            ariaLabelToggle="compound search filter attribute selector toggle"
        >
            {entityAttributes.map(({ displayName }) => {
                return (
                    <SelectOption key={displayName} value={displayName}>
                        {displayName}
                    </SelectOption>
                );
            })}
        </SimpleSelect>
    );
}

export default AttributeSelector;
