import { SelectOption, ToolbarItem } from '@patternfly/react-core';

import SimpleSelect from './SimpleSelect';
import type { CompoundSearchFilterConfig, CompoundSearchFilterEntity } from '../types';

export type SelectedEntity = string | undefined;
export type EntitySelectorOnChange = (value: string | number | undefined) => void;

export type EntitySelectorProps = {
    entity: CompoundSearchFilterEntity;
    onChange: EntitySelectorOnChange;
    config: CompoundSearchFilterConfig;
    menuToggleClassName?: string;
};

function EntitySelector({ entity, onChange, config, menuToggleClassName }: EntitySelectorProps) {
    return (
        <ToolbarItem>
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
        </ToolbarItem>
    );
}

export default EntitySelector;
