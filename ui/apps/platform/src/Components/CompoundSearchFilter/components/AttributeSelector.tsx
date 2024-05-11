import React from 'react';
import { SelectOption } from '@patternfly/react-core';

import {
    CompoundSearchFilterConfig,
    SearchFilterEntityName,
} from 'Components/CompoundSearchFilter/types';
import { getEntityAttributes } from 'Components/CompoundSearchFilter/utils/utils';

import SimpleSelect from './SimpleSelect';

export type AttributeSelectorProps = {
    selectedEntity: SearchFilterEntityName;
    selectedAttribute: string;
    onChange: (value: string | number | undefined) => void;
    config: Partial<CompoundSearchFilterConfig>;
};

function AttributeSelector({
    selectedEntity,
    selectedAttribute,
    onChange,
    config,
}: AttributeSelectorProps) {
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
