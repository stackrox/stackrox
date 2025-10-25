import { SelectOption } from '@patternfly/react-core';

import { getEntity } from 'Components/CompoundSearchFilter/utils/utils';

import SimpleSelect from './SimpleSelect';
import type { CompoundSearchFilterConfig } from '../types';

export type SelectedEntity = string | undefined;
export type EntitySelectorOnChange = (value: string | number | undefined) => void;

export type EntitySelectorProps = {
    selectedEntity: SelectedEntity;
    onChange: EntitySelectorOnChange;
    config: CompoundSearchFilterConfig;
    menuToggleClassName?: string;
};

function EntitySelector({
    selectedEntity = '',
    onChange,
    config,
    menuToggleClassName,
}: EntitySelectorProps) {
    const entity = getEntity(config, selectedEntity);

    if (!entity) {
        return null;
    }

    return (
        <SimpleSelect
            menuToggleClassName={menuToggleClassName}
            value={entity.displayName}
            onChange={onChange}
            ariaLabelMenu="compound search filter entity selector menu"
            ariaLabelToggle="compound search filter entity selector toggle"
        >
            {config.map(({ displayName }) => {
                return (
                    <SelectOption key={displayName} value={displayName}>
                        {displayName}
                    </SelectOption>
                );
            })}
        </SimpleSelect>
    );
}

export default EntitySelector;
