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
        const isAllCurrentlySelected = (statusesSelected ?? []).length === 0;
        const validStatuses = selections.filter((s) => s !== optionAll && isStatus(s));

        if (
            (selections.includes(optionAll) && !isAllCurrentlySelected) ||
            validStatuses.length === 0 ||
            validStatuses.length === statuses.length
        ) {
            setStatusesSelected(undefined);
            return;
        }

        setStatusesSelected(validStatuses);
    }

    const options = [
        <SelectOption key="All" value={optionAll}>
            All statuses
        </SelectOption>,
        <Divider key="Divider" />,
        ...statuses.map((status) => (
            <SelectOption key={status} value={status}>
                {getStatusText(status)}
            </SelectOption>
        )),
    ];

    return (
        <CheckboxSelect
            id="status-filter"
            selections={statusesSelected ?? [optionAll]}
            onChange={onSelect}
            ariaLabel="Status filter menu items"
            toggleAriaLabel="Status filter menu toggle"
            placeholderText="Filter by status"
            isDisabled={isDisabled}
        >
            {options}
        </CheckboxSelect>
    );
}

export default SearchFilterStatuses;
