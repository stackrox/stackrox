import { SelectOption } from '@patternfly/react-core';

import SimpleSelect from './SimpleSelect';
import type { CompoundSearchFilterAttribute } from '../types';

export type SelectedAttribute = string | undefined;
export type AttributeSelectorOnChange = (value: string | number | undefined) => void;

export type AttributeSelectorProps = {
    attributes: CompoundSearchFilterAttribute[];
    attribute: CompoundSearchFilterAttribute;
    onChange: AttributeSelectorOnChange;
    menuToggleClassName?: string;
};

function AttributeSelector({
    attributes,
    attribute,
    onChange,
    menuToggleClassName,
}: AttributeSelectorProps) {
    return (
        <SimpleSelect
            menuToggleClassName={menuToggleClassName}
            value={attribute.displayName}
            onChange={onChange}
            ariaLabelMenu="compound search filter attribute selector menu"
            ariaLabelToggle="compound search filter attribute selector toggle"
        >
            {attributes.map(({ displayName }) => {
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
