import React from 'react';
import { Divider, SelectOption } from '@patternfly/react-core';

import CheckboxSelect from 'Components/PatternFly/CheckboxSelect';
import { isStatus, statuses } from 'services/DiscoveredClusterService';
import type { DiscoveredClusterStatus } from 'services/DiscoveredClusterService';

import { getStatusText } from './DiscoveredCluster';

const optionAll = '##All Statuses##';

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
    function onSelect(selections: string[]) {
        const hadAllOption = (statusesSelected ?? []).length === 0;
        const isSelectAll = selections.includes(optionAll) && !hadAllOption;
        const validStatuses = selections.filter((s) => s !== optionAll && isStatus(s));

        if (isSelectAll || validStatuses.length === 0) {
            setStatusesSelected(undefined);
            return;
        }

        setStatusesSelected(validStatuses);
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
        <CheckboxSelect
            id="status-filter"
            selections={statusesSelected ?? [optionAll]}
            onChange={onSelect}
            ariaLabel="Status filter menu items"
            placeholderText="Filter by status"
            isDisabled={isDisabled}
        >
            {options}
        </CheckboxSelect>
    );
}

export default SearchFilterStatuses;
