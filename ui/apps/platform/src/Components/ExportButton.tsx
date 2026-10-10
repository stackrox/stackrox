import { useState } from 'react';
import { FileText } from 'react-feather';
import { toast } from 'react-toastify';
import { Button, Popover } from '@patternfly/react-core';

import downloadCSV from 'services/CSVDownloadService';
import WorkflowPDFExportButton from 'Components/WorkflowPDFExportButton';
import ButtonClassic from 'Components/Button';
import useCaseTypes from 'constants/useCaseTypes';
import entityTypes from 'constants/entityTypes';
import { addBrandedTimestampToString } from 'utils/dateUtils';

const queryParamMap = {
    [entityTypes.CLUSTER]: 'clusterId',
    [entityTypes.STANDARD]: 'standardId',
    ALL: '',
};

const complianceDownloadUrl = '/api/compliance/export/csv';

function checkVulnMgmtSupport(page: string, type: string | null) {
    return (
        page === useCaseTypes.VULN_MANAGEMENT &&
        (type === entityTypes.CVE ||
            type === entityTypes.IMAGE_CVE ||
            type === entityTypes.NODE_CVE ||
            type === entityTypes.CLUSTER_CVE)
    );
}

function checkComplianceSupport(page: string, type: string | null) {
    return page === useCaseTypes.COMPLIANCE && type !== null && type in queryParamMap;
}

type ExportButtonProps = {
    className?: string;
    textClass?: string | null;
    fileName?: string;
    type?: string | null;
    id?: string;
    pdfId?: string;
    tableOptions?: Record<string, unknown>;
    customCsvExportHandler?: ((fileName: string) => Promise<void>) | null;
    page?: string;
    disabled?: boolean;
    isExporting: boolean;
    setIsExporting: (isExporting: boolean) => void;
};

function ExportButton({
    className = 'btn btn-base h-10',
    textClass = null,
    fileName = 'compliance',
    type = null,
    id = '',
    pdfId = '',
    tableOptions = {},
    customCsvExportHandler = null,
    page = '',
    disabled = false,
    isExporting,
    setIsExporting,
}: ExportButtonProps) {
    const [csvIsDownloading, setCsvIsDownloading] = useState(false);

    function isCsvSupported() {
        const isVulnMgmtSupportedPage = checkVulnMgmtSupport(page, type);
        const isComplianceSupportedPage = checkComplianceSupport(page, type);
        return isVulnMgmtSupportedPage || isComplianceSupportedPage;
    }

    function downloadCSVFile() {
        const csvName = addBrandedTimestampToString(fileName);

        if (checkVulnMgmtSupport(page, type)) {
            if (!customCsvExportHandler) {
                throw new Error('A CSV export handler was not supplied to handle CSV export');
            }

            setCsvIsDownloading(true);
            customCsvExportHandler(csvName)
                .catch((err) => {
                    toast(`An error occurred while trying to export: ${err}`);
                })
                .finally(() => {
                    setCsvIsDownloading(false);
                });
        } else {
            // otherwise, use legacy compliance CSV export
            let queryStr = '';
            let value = null;

            // Support for StandardId & ClusterId only
            if (type && queryParamMap[type as keyof typeof queryParamMap]) {
                if (id) {
                    value = id;
                }
                queryStr = `${queryParamMap[type as keyof typeof queryParamMap]}=${value}`;
            }

            downloadCSV(csvName, complianceDownloadUrl, queryStr).catch(() => {
                // Error handling is done inside the service
            });
        }
    }

    const csvButtonText =
        type === entityTypes.CVE ? 'Download CVES as CSV' : 'Download Evidence as CSV';
    const headerText = fileName;
    const pdfFileName = addBrandedTimestampToString(headerText);
    const wrapperClass = pdfId && !isCsvSupported() ? 'min-w-64' : '';

    const hasContent = pdfId || isCsvSupported();

    const popoverContent = hasContent ? (
        <div className={`flex flex-col text-base-600 ${wrapperClass}`}>
            <ul
                className="bg-base-100 rounded"
                style={{
                    borderColor: 'var(--pf-v5-global--primary-color--100)',
                    borderWidth: 2,
                }}
            >
                <li className="p-4 border-b">
                    <div className="flex">
                        {pdfId && (
                            <WorkflowPDFExportButton
                                id={pdfId}
                                className={`min-w-48 ${isCsvSupported() ? 'mr-2' : 'w-full'}`}
                                tableOptions={tableOptions}
                                fileName={pdfFileName}
                                pdfTitle={headerText}
                                isExporting={isExporting}
                                setIsExporting={setIsExporting}
                            />
                        )}
                        {isCsvSupported() && (
                            <Button
                                variant="primary"
                                onClick={downloadCSVFile}
                                isLoading={csvIsDownloading}
                            >
                                {csvButtonText}
                            </Button>
                        )}
                    </div>
                </li>
                <li className="hidden">
                    <span>or share to</span>
                    <div>Slack</div>
                </li>
            </ul>
        </div>
    ) : null;

    if (!hasContent) {
        return (
            <div className="relative pl-2">
                <ButtonClassic
                    className={className}
                    disabled={disabled}
                    text="Export"
                    textCondensed="Export"
                    textClass={textClass}
                    icon={<FileText size="14" className="mx-1 lg:ml-1 lg:mr-3" />}
                    onClick={() => {}}
                />
            </div>
        );
    }

    return (
        <div className="relative pl-2">
            <Popover
                aria-label="Export options"
                hasNoPadding
                hasAutoWidth
                showClose={false}
                position="bottom-end"
                bodyContent={popoverContent}
            >
                <ButtonClassic
                    className={className}
                    disabled={disabled}
                    text="Export"
                    textCondensed="Export"
                    textClass={textClass}
                    icon={<FileText size="14" className="mx-1 lg:ml-1 lg:mr-3" />}
                />
            </Popover>
        </div>
    );
}

export default ExportButton;
