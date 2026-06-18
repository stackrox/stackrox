import { useState } from 'react';
import { Button, NumberInput, ToolbarItem } from '@patternfly/react-core';
import { ArrowRightIcon } from '@patternfly/react-icons';

import { serializeRelativeOlderThan } from '../utils/utils';
import type { OnSearchCallback } from '../types';

export type SearchFilterDateRelativeOlderThanProps = {
    category: string;
    isDisabled?: boolean;
    onSearch: OnSearchCallback;
};

function SearchFilterDateRelativeOlderThan({
    category,
    isDisabled = false,
    onSearch,
}: SearchFilterDateRelativeOlderThanProps) {
    const [days, setDays] = useState(0);

    function updateDays(newValue: number) {
        setDays(Math.max(0, Math.floor(newValue)));
    }

    function onApply() {
        const value = serializeRelativeOlderThan(days);
        if (value !== null) {
            onSearch([{ action: 'APPEND', category, value }]);
            setDays(0);
        }
    }

    return (
        <>
            <ToolbarItem>
                <NumberInput
                    inputAriaLabel="Number of days"
                    isDisabled={isDisabled}
                    value={days}
                    min={0}
                    onChange={(event) => {
                        const parsed = Number(event.currentTarget.value);
                        updateDays(Number.isNaN(parsed) ? 0 : parsed);
                    }}
                    onMinus={() => updateDays(days - 1)}
                    onPlus={() => updateDays(days + 1)}
                    minusBtnAriaLabel="Decrease days"
                    plusBtnAriaLabel="Increase days"
                />
            </ToolbarItem>
            <ToolbarItem>
                <Button
                    icon={<ArrowRightIcon />}
                    variant="control"
                    aria-label="Apply relative date filter"
                    onClick={onApply}
                />
            </ToolbarItem>
        </>
    );
}

export default SearchFilterDateRelativeOlderThan;
