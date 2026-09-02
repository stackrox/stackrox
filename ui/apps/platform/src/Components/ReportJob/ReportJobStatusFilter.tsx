import { SelectList, SelectOption } from '@patternfly/react-core';
import { FilterIcon } from '@patternfly/react-icons';

import CheckboxSelect from 'Components/CheckboxSelect';
import { reportJobStatusLabels, reportJobStatuses } from './types';
import type { ReportJobStatus } from './types';

export function isReportJobStatus(value: string): value is ReportJobStatus {
    return value in reportJobStatuses;
}

export type ReportJobStatusFilterProps = {
    availableStatuses: ReportJobStatus[];
    selectedStatuses: ReportJobStatus[];
    onChange: (checked: boolean, value: ReportJobStatus) => void;
};

function ReportJobStatusFilter({
    availableStatuses,
    selectedStatuses,
    onChange,
}: ReportJobStatusFilterProps) {
    function onChangeHandler(checked: boolean, value: string) {
        if (!isReportJobStatus(value)) {
            return;
        }
        onChange(checked, value);
    }

    return (
        <CheckboxSelect
            ariaLabelMenu="Report job status select menu"
            toggleLabel="Report job status"
            toggleIcon={<FilterIcon />}
            selection={selectedStatuses}
            onChange={onChangeHandler}
        >
            <SelectList>
                {availableStatuses.map((status) => {
                    return (
                        <SelectOption
                            key={status}
                            value={status}
                            hasCheckbox
                            isSelected={selectedStatuses.includes(status)}
                        >
                            {reportJobStatusLabels[status]}
                        </SelectOption>
                    );
                })}
            </SelectList>
        </CheckboxSelect>
    );
}

export default ReportJobStatusFilter;
