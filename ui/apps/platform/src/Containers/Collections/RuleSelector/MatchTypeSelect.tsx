import React, { ReactElement } from 'react';
import { Select } from '@patternfly/react-core';
import useToggle from 'hooks/useToggle';
import { ByLabelMatchType, ByNameMatchType, MatchType } from '../types';

export type MatchTypeSelectProps<T extends MatchType> = {
    onChange: (value: T) => void;
    selected: T;
    children: ReactElement[];
    isDisabled?: boolean;
};

function MatchTypeSelect<T extends MatchType>({
    onChange,
    selected,
    children,
    isDisabled = false,
}: MatchTypeSelectProps<T>) {
    const { isOn: isOpen, onToggle, toggleOff: closeSelect } = useToggle();

    function onSelect(_, value) {
        onChange(value);
        closeSelect();
    }

    return (
        <>
            <Select
                isOpen={isOpen}
                onToggle={onToggle}
                selections={selected}
                onSelect={onSelect}
                isDisabled={isDisabled}
            >
                {children}
            </Select>
        </>
    );
}

export function NameMatchTypeSelect(props: MatchTypeSelectProps<ByNameMatchType>) {
    return <MatchTypeSelect {...props} />;
}

export function LabelMatchTypeSelect(props: MatchTypeSelectProps<ByLabelMatchType>) {
    return <MatchTypeSelect {...props} />;
}
