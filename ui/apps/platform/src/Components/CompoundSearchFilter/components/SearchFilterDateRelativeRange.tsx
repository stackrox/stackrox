import { useState } from 'react';
import { Button, TextInput, ToolbarItem } from '@patternfly/react-core';
import { ArrowRightIcon } from '@patternfly/react-icons';

import { serializeRelativeDateRange } from '../utils/utils';
import type { OnSearchCallback } from '../types';

export type SearchFilterDateRelativeRangeProps = {
    category: string;
    isDisabled?: boolean;
    onSearch: OnSearchCallback;
};

function SearchFilterDateRelativeRange({
    category,
    isDisabled = false,
    onSearch,
}: SearchFilterDateRelativeRangeProps) {
    const [minDays, setMinDays] = useState('');
    const [maxDays, setMaxDays] = useState('');

    function onApply() {
        const value = serializeRelativeDateRange(Number(minDays), Number(maxDays));
        if (minDays === '' || maxDays === '' || value === null) {
            return;
        }
        onSearch([{ action: 'APPEND', category, value }]);
        setMinDays('');
        setMaxDays('');
    }

    return (
        <>
            <ToolbarItem style={{ flexBasis: '6rem' }}>
                <TextInput
                    aria-label="Minimum days ago"
                    isDisabled={isDisabled}
                    value={minDays}
                    type="number"
                    min={0}
                    onChange={(_event, value) => setMinDays(value)}
                    placeholder="days"
                />
            </ToolbarItem>
            <ToolbarItem alignSelf="center">to</ToolbarItem>
            <ToolbarItem style={{ flexBasis: '6rem' }}>
                <TextInput
                    aria-label="Maximum days ago"
                    isDisabled={isDisabled}
                    value={maxDays}
                    type="number"
                    min={0}
                    onChange={(_event, value) => setMaxDays(value)}
                    placeholder="days"
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
