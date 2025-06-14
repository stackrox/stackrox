import React from 'react';
import { Button } from '@patternfly/react-core';
import { SearchFilter } from 'types/search';

export type SnoozedCveToggleButtonProps = {
    searchFilter: SearchFilter;
    setSearchFilter: (searchFilter: SearchFilter) => void;
    snoozedCveCount: number | undefined;
};

function SnoozedCveToggleButton({
    searchFilter,
    setSearchFilter,
    snoozedCveCount,
}: SnoozedCveToggleButtonProps) {
    const isSnoozeFilterActive = searchFilter['CVE Snoozed']?.[0] === 'true';
    const buttonText = isSnoozeFilterActive ? 'Show observed CVEs' : 'Show snoozed CVEs';
    const showCountBadge =
        typeof snoozedCveCount === 'number' && snoozedCveCount > 0 && !isSnoozeFilterActive;
    const badgeCount = showCountBadge ? { isRead: true, count: snoozedCveCount } : undefined;

    function toggleSnoozeFilter() {
        const nextFilter = { ...searchFilter };
        if (isSnoozeFilterActive) {
            delete nextFilter['CVE Snoozed'];
        } else {
            nextFilter['CVE Snoozed'] = ['true'];
        }
        setSearchFilter(nextFilter);
    }

    return (
        <Button variant="secondary" onClick={toggleSnoozeFilter} countOptions={badgeCount}>
            {buttonText}
        </Button>
    );
}

export default SnoozedCveToggleButton;
