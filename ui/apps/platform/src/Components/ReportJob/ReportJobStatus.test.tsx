import React, { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import ReportJobStatus from './ReportJobStatus';

/*
  The following list enumerates possible states that can be checked:
    * 'Preparing': This state indicates the system is in the process of running.
    * 'Waiting': The system is idle, waiting for necessary input or resources to proceed.
    * 'Failure': An error occurred, causing the run to fail. Check for issues and try again.
    * 'Download - Not actionable / No permissions': The download is available but cannot be accessed or manipulated due to insufficient permissions.
    * 'Download - Actionable': The download is available and can be accessed or manipulated by the user.
    * 'Download - Deleted': The download was previously available but has since been removed or is no longer accessible.
    * 'Email - Delivered': The email has been successfully sent and delivered to the recipient.
    * 'Unknown': The current state cannot be determined or does not match any of the predefined states listed above. Typescript should safegaurd against this. This is mostly in case the API returns a different value
*/
describe('ReportJobStatus', () => {
    test('should display the status text and icon when preparing', () => {
        render(
            <ReportJobStatus
                reportStatus={{
                    runState: 'PREPARING',
                    reportRequestType: 'ON_DEMAND',
                    reportNotificationMethod: 'DOWNLOAD',
                    completedAt: '',
                    errorMsg: '',
                }}
                isDownloadAvailable={false}
                areDownloadActionsDisabled={false}
                onDownload={() => {}}
            />
        );

        const statusTextElement = screen.getByText('Preparing');
        const statusIconElement = screen.getByTitle('Report run is preparing');

        expect(statusTextElement).toBeInTheDocument();
        expect(statusIconElement).toBeInTheDocument();
    });

    test('should display the status text and icon when waiting', () => {
        render(
            <ReportJobStatus
                reportStatus={{
                    runState: 'WAITING',
                    reportRequestType: 'ON_DEMAND',
                    reportNotificationMethod: 'DOWNLOAD',
                    completedAt: '',
                    errorMsg: '',
                }}
                isDownloadAvailable={false}
                areDownloadActionsDisabled={false}
                onDownload={() => {}}
            />
        );

        const statusTextElement = screen.getByText('Waiting');
        const statusIconElement = screen.getByTitle('Report run is waiting');

        expect(statusTextElement).toBeInTheDocument();
        expect(statusIconElement).toBeInTheDocument();
    });

    test('should display the status text and icon when failed', async () => {
        render(
            <ReportJobStatus
                reportStatus={{
                    runState: 'FAILURE',
                    reportRequestType: 'ON_DEMAND',
                    reportNotificationMethod: 'DOWNLOAD',
                    completedAt: '',
                    errorMsg: 'This is an error message',
                }}
                isDownloadAvailable={false}
                areDownloadActionsDisabled={false}
                onDownload={() => {}}
            />
        );

        const statusTextElement = screen.getByText('Error');
        const statusIconElement = screen.getByRole('img', { name: 'Report run was unsuccessful' });

        expect(statusTextElement).toBeInTheDocument();
        expect(statusIconElement).toBeInTheDocument();

        // trigger the hover action
        fireEvent.mouseEnter(statusIconElement);

        const tooltipElement = await screen.findByText('This is an error message');

        expect(tooltipElement).toBeInTheDocument();

        // cleanup hover action
        fireEvent.mouseLeave(statusIconElement);
    });

    test('should display the status text and icon when download is available but not accessible', async () => {
        render(
            <ReportJobStatus
                reportStatus={{
                    runState: 'GENERATED',
                    reportRequestType: 'ON_DEMAND',
                    reportNotificationMethod: 'DOWNLOAD',
                    completedAt: '',
                    errorMsg: '',
                }}
                isDownloadAvailable
                areDownloadActionsDisabled
                onDownload={() => {}}
            />
        );

        const statusTextElement = screen.getByText('Ready for download');
        const statusIconElement = screen.getByRole('img', {
            name: 'Report download was successfully prepared',
        });

        expect(statusTextElement).toBeInTheDocument();
        expect(statusIconElement).toBeInTheDocument();

        const helpIconElement = screen.getByRole('img', {
            name: 'Permission limitations on download',
        });

        // trigger the hover action
        fireEvent.mouseEnter(helpIconElement);

        const tooltipElement = await screen.findByText(
            'Only the requestor of the download has the authority to access or remove it.'
        );

        expect(tooltipElement).toBeInTheDocument();

        // cleanup hover action
        fireEvent.mouseLeave(helpIconElement);
    });

    test('should display the status text and icon when download is available and accessible', () => {
        render(
            <ReportJobStatus
                reportStatus={{
                    runState: 'GENERATED',
                    reportRequestType: 'ON_DEMAND',
                    reportNotificationMethod: 'DOWNLOAD',
                    completedAt: '',
                    errorMsg: '',
                }}
                isDownloadAvailable
                areDownloadActionsDisabled={false}
                onDownload={() => {}}
            />
        );

        const statusTextElement = screen.getByText('Ready for download');
        const statusIconElement = screen.getByTitle('Report download was successfully prepared');

        expect(statusTextElement).toBeInTheDocument();
        expect(statusIconElement).toBeInTheDocument();
    });

    test('should display the status text and icon when download is deleted', async () => {
        render(
            <ReportJobStatus
                reportStatus={{
                    runState: 'GENERATED',
                    reportRequestType: 'ON_DEMAND',
                    reportNotificationMethod: 'DOWNLOAD',
                    completedAt: '',
                    errorMsg: '',
                }}
                isDownloadAvailable={false}
                areDownloadActionsDisabled={false}
                onDownload={() => {}}
            />
        );

        const statusTextElement = screen.getByText('Download deleted');
        const statusIconElement = screen.getByRole('img', {
            name: 'Report download was deleted',
        });

        expect(statusTextElement).toBeInTheDocument();
        expect(statusIconElement).toBeInTheDocument();

        const helpIconElement = screen.getByRole('img', {
            name: 'Download deletion explanation',
        });

        // trigger the hover action
        fireEvent.mouseEnter(helpIconElement);

        const tooltipElement = await screen.findByText(
            'The download was deleted. Please generate a new download if needed.'
        );

        expect(tooltipElement).toBeInTheDocument();

        // cleanup hover action
        fireEvent.mouseLeave(helpIconElement);
    });

    test('should display the status text and icon when email is sent', () => {
        render(
            <ReportJobStatus
                reportStatus={{
                    runState: 'DELIVERED',
                    reportRequestType: 'ON_DEMAND',
                    reportNotificationMethod: 'EMAIL',
                    completedAt: '',
                    errorMsg: '',
                }}
                isDownloadAvailable
                areDownloadActionsDisabled={false}
                onDownload={() => {}}
            />
        );

        const statusTextElement = screen.getByText('Successfully sent');
        const statusIconElement = screen.getByTitle('Report was successfully sent');

        expect(statusTextElement).toBeInTheDocument();
        expect(statusIconElement).toBeInTheDocument();
    });
});
