import type { ReactElement } from 'react';
import SelectSingle from 'Components/SelectSingle/SelectSingle';
import type { ByLabelMatchType, ByNameMatchType, MatchType } from '../types';

export type MatchTypeSelectProps<T extends MatchType> = {
    onChange: (value: T) => void;
    selected: T;
    children: ReactElement[];
    isDisabled?: boolean;
    id?: string;
};

function MatchTypeSelect<T extends MatchType>({
    onChange,
    selected,
    children,
    isDisabled = false,
    id = 'match-type-select',
}: MatchTypeSelectProps<T>) {
    function onSelect(_id: string, value: string) {
        onChange(value as T);
    }

    return (
        <SelectSingle id={id} value={selected} handleSelect={onSelect} isDisabled={isDisabled}>
            {children}
        </SelectSingle>
    );
}

export function NameMatchTypeSelect(props: MatchTypeSelectProps<ByNameMatchType>) {
    return <MatchTypeSelect {...props} />;
}

export function LabelMatchTypeSelect(props: MatchTypeSelectProps<ByLabelMatchType>) {
    return <MatchTypeSelect {...props} />;
}
