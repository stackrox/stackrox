import React, { useState } from 'react';
import { Divider } from '@patternfly/react-core';
import { SelectOption, Select } from '@patternfly/react-core/deprecated';

import { isStatus, statuses } from 'services/DiscoveredClusterService';
import type { DiscoveredClusterStatus } from 'services/DiscoveredClusterService';

import { getStatusText } from './DiscoveredCluster';

const optionAll = 'All_statuses';

type SearchFilterStatusesProps = {
    statusesSelected: DiscoveredClusterStatus[] | undefined;
    isDisabled: boolean;
    setStatusesSelected: (statuses: DiscoveredClusterStatus[] | undefined) => void;
};

function SearchFilterStatuses({
    statusesSelected,
    isDisabled,
    setStatusesSelected,
}: SearchFilterStatusesProps) {
    const [isOpen, setIsOpen] = useState(false);

    function onSelect(_event, selection) {
        const previousStatuses = statusesSelected ?? [];
        if (isStatus(selection)) {
            setStatusesSelected(
                previousStatuses.includes(selection)
                    ? previousStatuses.filter((status) => status !== selection)
                    : [...previousStatuses, selection]
            );
        } else {
            setStatusesSelected(undefined);
        }
    }

    const options = statuses.map((status) => (
        <SelectOption key={status} value={status}>
            {getStatusText(status)}
        </SelectOption>
    ));
    options.push(
        <Divider key="Divider" />,
        <SelectOption key="All" value={optionAll}>
            All statuses
        </SelectOption>
    );

    return (
        <Select
            variant="checkbox"
            placeholderText="Filter by status"
            aria-label="Status filter menu items"
            toggleAriaLabel="Status filter menu toggle"
            onToggle={(_event, val) => setIsOpen(val)}
            onSelect={onSelect}
            selections={statusesSelected ?? optionAll}
            isDisabled={isDisabled}
            isOpen={isOpen}
        >
            {options}
        </Select>
    );
}

export default SearchFilterStatuses;
