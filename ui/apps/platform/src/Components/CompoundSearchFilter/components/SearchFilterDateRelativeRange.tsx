import { useState } from 'react';
import { Button, NumberInput, ToolbarItem } from '@patternfly/react-core';
import { ArrowRightIcon } from '@patternfly/react-icons';

import { serializeRelativeDateRange } from '../utils/utils';
import type { OnSearchCallback } from '../types';

export type SearchFilterDateRelativeRangeProps = {
    category: string;
    isDisabled?: boolean;
    onSearch: OnSearchCallback;
};

function normalizeInput(value: number): number {
    return Math.max(0, Math.floor(Number.isNaN(value) ? 0 : value));
}

function SearchFilterDateRelativeRange({
    category,
    isDisabled = false,
    onSearch,
}: SearchFilterDateRelativeRangeProps) {
    const [minDays, setMinDays] = useState(0);
    const [maxDays, setMaxDays] = useState(0);

    function updateMin(newValue: number) {
        setMinDays(normalizeInput(newValue));
    }

    function updateMax(newValue: number) {
        setMaxDays(normalizeInput(newValue));
    }

    function onApply() {
        const value = serializeRelativeDateRange(minDays, maxDays);
        if (value !== null) {
            onSearch([{ action: 'APPEND', category, value }]);
            setMinDays(0);
            setMaxDays(0);
        }
    }

    return (
        <>
            <ToolbarItem>
                <NumberInput
                    inputAriaLabel="Minimum days ago"
                    isDisabled={isDisabled}
                    value={minDays}
                    min={0}
                    onChange={(event) => {
                        updateMin(Number(event.currentTarget.value));
                    }}
                    onMinus={() => updateMin(minDays - 1)}
                    onPlus={() => updateMin(minDays + 1)}
                    minusBtnAriaLabel="Decrease minimum days"
                    plusBtnAriaLabel="Increase minimum days"
                />
            </ToolbarItem>
            <ToolbarItem alignSelf="center">to</ToolbarItem>
            <ToolbarItem>
                <NumberInput
                    inputAriaLabel="Maximum days ago"
                    isDisabled={isDisabled}
                    value={maxDays}
                    min={0}
                    onChange={(event) => {
                        updateMax(Number(event.currentTarget.value));
                    }}
                    onMinus={() => updateMax(maxDays - 1)}
                    onPlus={() => updateMax(maxDays + 1)}
                    minusBtnAriaLabel="Decrease maximum days"
                    plusBtnAriaLabel="Increase maximum days"
                />
            </ToolbarItem>
            <ToolbarItem>
                <Button
                    icon={<ArrowRightIcon />}
                    variant="control"
                    aria-label="Apply relative date range filter"
                    onClick={onApply}
                />
            </ToolbarItem>
        </>
    );
}

export default SearchFilterDateRelativeRange;
