import React, { ReactElement } from 'react';
import { Select, SelectOption } from '@patternfly/react-core/deprecated';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';

export type EntitySelectorProps = {
    value: string;
    onChange: (value) => void;
    children: ReactElement<typeof SelectOption>[];
};

function EntitySelector({ value, onChange, children }: EntitySelectorProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();

    function onSelect(e, selection) {
        onChange(selection);
        closeSelect();
    }

    return (
        <Select
            variant="single"
            toggleAriaLabel="compound search filter entity selector toggle"
            aria-label="compound search filter entity selector items"
            onToggle={(_e, v) => onToggle(v)}
            onSelect={onSelect}
            selections={value}
            isOpen={isOpen}
            className="pf-v5-u-flex-basis-0"
        >
            {children}
        </Select>
    );
}

export default EntitySelector;
