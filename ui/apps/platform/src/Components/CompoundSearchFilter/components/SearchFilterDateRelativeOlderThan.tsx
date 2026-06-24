import { useState } from 'react';
import { Button, TextInput, ToolbarItem } from '@patternfly/react-core';
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
    const [days, setDays] = useState('');

    function onApply() {
        const value = serializeRelativeOlderThan(Number(days));
        if (days === '' || value === null) {
            return;
        }
        onSearch([{ action: 'APPEND', category, value }]);
        setDays('');
    }

    return (
        <>
            <ToolbarItem style={{ flexBasis: '6rem' }}>
                <TextInput
                    aria-label="Number of days"
                    isDisabled={isDisabled}
                    value={days}
                    type="number"
                    min={0}
                    onChange={(_event, value) => setDays(value)}
                    placeholder="days"
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
