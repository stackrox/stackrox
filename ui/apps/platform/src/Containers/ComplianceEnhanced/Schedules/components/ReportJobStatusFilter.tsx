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
    selectedStatuses: ReportJobStatus[];
    onChange: (checked: boolean, value: ReportJobStatus) => void;
};

function ReportJobStatusFilter({ selectedStatuses, onChange }: ReportJobStatusFilterProps) {
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
                <SelectOption
                    key={reportJobStatuses.PREPARING}
                    value={reportJobStatuses.PREPARING}
                    hasCheckbox
                    isSelected={selectedStatuses.includes(reportJobStatuses.PREPARING)}
                >
                    Preparing
                </SelectOption>
                <SelectOption
                    key={reportJobStatuses.WAITING}
                    value={reportJobStatuses.WAITING}
                    hasCheckbox
                    isSelected={selectedStatuses.includes(reportJobStatuses.WAITING)}
                >
                    Waiting
                </SelectOption>
                <SelectOption
                    key={reportJobStatuses.DOWNLOAD_GENERATED}
                    value={reportJobStatuses.DOWNLOAD_GENERATED}
                    hasCheckbox
                    isSelected={selectedStatuses.includes(reportJobStatuses.DOWNLOAD_GENERATED)}
                >
                    Download generated
                </SelectOption>
                <SelectOption
                    key={reportJobStatuses.PARTIAL_ERROR}
                    value={reportJobStatuses.PARTIAL_ERROR}
                    hasCheckbox
                    isSelected={selectedStatuses.includes(reportJobStatuses.PARTIAL_ERROR)}
                >
                    Partial download
                </SelectOption>
                <SelectOption
                    key={reportJobStatuses.EMAIL_DELIVERED}
                    value={reportJobStatuses.EMAIL_DELIVERED}
                    hasCheckbox
                    isSelected={selectedStatuses.includes(reportJobStatuses.EMAIL_DELIVERED)}
                >
                    Email delivered
                </SelectOption>
                <SelectOption
                    key={reportJobStatuses.ERROR}
                    value={reportJobStatuses.ERROR}
                    hasCheckbox
                    isSelected={selectedStatuses.includes(reportJobStatuses.ERROR)}
                >
                    Error
                </SelectOption>
            </SelectList>
        </CheckboxSelect>
    );
}

export default ReportJobStatusFilter;
