import React from 'react';
import { SelectList, SelectOption } from '@patternfly/react-core';
import { FilterIcon } from '@patternfly/react-icons';

import CheckboxSelect from 'Components/CheckboxSelect';
import { ValueOf } from 'utils/type.utils';

export const reportJobStatuses = {
    WAITING: 'WAITING',
    PREPARING: 'PREPARING',
    DOWNLOAD_GENERATED: 'DOWNLOAD_GENERATED',
    EMAIL_DELIVERED: 'EMAIL_DELIVERED',
    ERROR: 'ERROR',
    PARTIAL_ERROR: 'PARTIAL_ERROR',
} as const;

export type ReportJobStatus = ValueOf<typeof reportJobStatuses>;

export const reportJobStatusLabels: Record<ReportJobStatus, string> = {
    WAITING: 'Waiting',
    PREPARING: 'Preparing',
    DOWNLOAD_GENERATED: 'Download generated',
    EMAIL_DELIVERED: 'Email delivered',
    ERROR: 'Error',
    PARTIAL_ERROR: 'Partial error',
};

function isReportJobStatus(value: string): value is ReportJobStatus {
    return value in reportJobStatuses;
}

/**
 * Ensures that the given search filter value is converted to an array of valid report job status values.
 *
 * Example:
 * ensureReportJobStatuses(["WAITING", "PREPARING"]);  // returns ["WAITING", "PREPARING"]
 * ensureReportJobStatuses("WAITING");                 // returns [] (since input is not an array)
 * ensureReportJobStatuses(undefined);                 // returns []
 *
 * @param searchFilterValue - The input value, which can be a string, an array of strings, or undefined.
 * @returns {ReportJobStatus[]} - If the input is an array of strings, it filters the values that match valid "ReportJobStatus"s
 *                         and returns them as an array.
 *                         If the input is not an array or undefined, it returns an empty array.
 */
export function ensureReportJobStatuses(
    searchFilterValue: string | string[] | undefined
): ReportJobStatus[] {
    if (Array.isArray(searchFilterValue)) {
        return searchFilterValue.filter((value) => isReportJobStatus(value));
    }
    return [];
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
