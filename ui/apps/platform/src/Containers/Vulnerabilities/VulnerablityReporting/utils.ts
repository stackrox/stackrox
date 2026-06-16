import type { ReportStatus } from 'types/reportJob';

export function getReportStatusText(
    reportStatus: ReportStatus | null,
    isDownloadAvailable: boolean
): string {
    let statusText = '-';

    const isDownload = reportStatus?.reportNotificationMethod === 'DOWNLOAD';
    const isEmail = reportStatus?.reportNotificationMethod === 'EMAIL';

    if (reportStatus?.runState === 'PREPARING') {
        statusText = 'Preparing';
    } else if (reportStatus?.runState === 'WAITING') {
        statusText = 'Waiting';
    } else if (reportStatus?.runState === 'FAILURE' && isEmail) {
        statusText = 'Email attempted';
    } else if (reportStatus?.runState === 'FAILURE' && isDownload) {
        statusText = 'Failed to generate download';
    } else if (!isDownload && reportStatus?.runState === 'DELIVERED') {
        statusText = 'Emailed';
    } else if (isDownload && isDownloadAvailable) {
        statusText = 'Download prepared';
    } else if (isDownload && !isDownloadAvailable) {
        statusText = 'Download deleted';
    } else if (reportStatus?.runState === 'FAILURE') {
        statusText = 'Error';
    }

    return statusText;
}
