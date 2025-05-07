import React, { useState } from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import ReportJobStatusFilter, {
    ensureReportJobStatuses,
    ReportJobStatus,
} from './ReportJobStatusFilter';

const getCheckboxOption = (name: string) => {
    return screen.getByRole('checkbox', {
        name,
    });
};

const ReportJobsStatusFilterWrapper = ({
    defaultOptions,
}: {
    defaultOptions: ReportJobStatus[];
}) => {
    const [selectedStatuses, setSelectedStatuses] = useState<ReportJobStatus[]>(defaultOptions);
    const onChange = (_checked: boolean, value: ReportJobStatus) => {
        const isStatusIncluded = selectedStatuses.includes(value);
        const newStatuses = isStatusIncluded
            ? selectedStatuses.filter((status) => status !== value)
            : [...selectedStatuses, value];
        setSelectedStatuses(newStatuses);
    };

    return (
        <ReportJobStatusFilter
            availableStatuses={[
                'WAITING',
                'PREPARING',
                'DOWNLOAD_GENERATED',
                'EMAIL_DELIVERED',
                'ERROR',
            ]}
            selectedStatuses={selectedStatuses}
            onChange={onChange}
        />
    );
};

describe('ReportJobStatusFilter', () => {
    test('should show multiple selected options by default', async () => {
        render(<ReportJobsStatusFilterWrapper defaultOptions={['PREPARING', 'ERROR']} />);

        const reportJobStatusFilterButton = screen.getByRole('button', {
            name: 'Report job status',
        });

        expect(reportJobStatusFilterButton).toBeInTheDocument();

        await userEvent.click(reportJobStatusFilterButton);

        const checkboxOptionForPreparing = getCheckboxOption('Preparing');
        const checkboxOptionForWaiting = getCheckboxOption('Waiting');
        const checkboxOptionForDownloadGenerated = getCheckboxOption('Download generated');
        const checkboxOptionFoEmailDelivered = getCheckboxOption('Email delivered');
        const checkboxOptionForError = getCheckboxOption('Error');

        expect(checkboxOptionForPreparing).toBeChecked();
        expect(checkboxOptionForWaiting).not.toBeChecked();
        expect(checkboxOptionForDownloadGenerated).not.toBeChecked();
        expect(checkboxOptionFoEmailDelivered).not.toBeChecked();
        expect(checkboxOptionForError).toBeChecked();
    });

    test('should select multiple options', async () => {
        render(<ReportJobsStatusFilterWrapper defaultOptions={[]} />);

        const reportJobStatusFilterButton = screen.getByRole('button', {
            name: 'Report job status',
        });

        expect(reportJobStatusFilterButton).toBeInTheDocument();

        await userEvent.click(reportJobStatusFilterButton);

        const checkboxOptionForPreparing = getCheckboxOption('Preparing');
        const checkboxOptionForWaiting = getCheckboxOption('Waiting');
        const checkboxOptionForDownloadGenerated = getCheckboxOption('Download generated');
        const checkboxOptionForEmailDelivered = getCheckboxOption('Email delivered');
        const checkboxOptionForError = getCheckboxOption('Error');

        await userEvent.click(checkboxOptionForPreparing);
        await userEvent.click(checkboxOptionForError);

        expect(checkboxOptionForPreparing).toBeChecked();
        expect(checkboxOptionForWaiting).not.toBeChecked();
        expect(checkboxOptionForDownloadGenerated).not.toBeChecked();
        expect(checkboxOptionForEmailDelivered).not.toBeChecked();
        expect(checkboxOptionForError).toBeChecked();
    });

    // Tests for the "ensureReportJobStatuses" helper function
    describe('ensureReportJobStatuses', () => {
        test('should filter out all incorrect values', async () => {
            expect(ensureReportJobStatuses('')).toEqual([]);
            expect(ensureReportJobStatuses(['TEST', 'BLAH', 'LOADING'])).toEqual([]);
        });

        test('should filter to show all correct values', async () => {
            expect(
                ensureReportJobStatuses([
                    'PREPARING',
                    'WAITING',
                    'DOWNLOAD_GENERATED',
                    'EMAIL_DELIVERED',
                    'ERROR',
                ])
            ).toEqual(['PREPARING', 'WAITING', 'DOWNLOAD_GENERATED', 'EMAIL_DELIVERED', 'ERROR']);
        });
    });
});
