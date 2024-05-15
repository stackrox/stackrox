import React from 'react';
import { SelectOption } from '@patternfly/react-core';

import { CompoundSearchFilterConfig } from 'Components/CompoundSearchFilter/types';
import { getEntityAttributes } from 'Components/CompoundSearchFilter/utils/utils';

import SimpleSelect from './SimpleSelect';
import { SelectedEntity } from './EntitySelector';

export type SelectedAttribute = string | undefined;
export type AttributeSelectorOnChange = (value: string | number | undefined) => void;

export type AttributeSelectorProps = {
    selectedEntity: SelectedEntity;
    selectedAttribute: SelectedAttribute;
    onChange: AttributeSelectorOnChange;
    config: Partial<CompoundSearchFilterConfig>;
};

function AttributeSelector({
    selectedEntity,
    selectedAttribute,
    onChange,
    config,
}: AttributeSelectorProps) {
    if (!selectedEntity) {
        return null;
    }

    const entityAttributes = getEntityAttributes(selectedEntity, config);

    if (entityAttributes.length === 0) {
        return null;
    }

    return (
        <SimpleSelect
            value={selectedAttribute}
            onChange={onChange}
            ariaLabelMenu="compound search filter attribute selector menu"
            ariaLabelToggle="compound search filter attribute selector toggle"
        >
            {entityAttributes.map((attribute) => {
                return (
                    <SelectOption key={attribute.displayName} value={attribute.displayName}>
                        {attribute.displayName}
                    </SelectOption>
                );
            })}
        </SimpleSelect>
    );
}

export default AttributeSelector;
