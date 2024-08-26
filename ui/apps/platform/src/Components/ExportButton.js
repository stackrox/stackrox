import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import onClickOutside from 'react-onclickoutside';
import { toast } from 'react-toastify';
import { Button } from '@patternfly/react-core';

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

function checkVulnMgmtSupport(page, type) {
    return (
        page === useCaseTypes.VULN_MANAGEMENT &&
        (type === entityTypes.CVE ||
            type === entityTypes.IMAGE_CVE ||
            type === entityTypes.NODE_CVE ||
            type === entityTypes.CLUSTER_CVE)
    );
}

function checkComplianceSupport(page, type) {
    return page === useCaseTypes.COMPLIANCE && Object.keys(queryParamMap).includes(type);
}
class ExportButton extends Component {
    static propTypes = {
        className: PropTypes.string,
        textClass: PropTypes.string,
        fileName: PropTypes.string,
        type: PropTypes.string,
        id: PropTypes.string,
        pdfId: PropTypes.string,
        tableOptions: PropTypes.shape({}),
        customCsvExportHandler: PropTypes.func,
        page: PropTypes.string,
        disabled: PropTypes.bool,
        isExporting: PropTypes.bool.isRequired,
        setIsExporting: PropTypes.func.isRequired,
    };

    static defaultProps = {
        className: 'btn btn-base h-10',
        textClass: null,
        fileName: 'compliance',
        type: null,
        id: '',
        pdfId: '',
        tableOptions: {},
        customCsvExportHandler: null,
        page: '',
        disabled: false,
    };

    constructor(props) {
        super(props);

        this.state = {
            toggleWidget: false,
            csvIsDownloading: false,
        };
    }

    handleClickOutside = () => this.setState({ toggleWidget: false });

    downloadCSVFile = () => {
        const { id, fileName, page, type, customCsvExportHandler } = this.props;
        const csvName = addBrandedTimestampToString(fileName);

        if (checkVulnMgmtSupport(page, type)) {
            if (!customCsvExportHandler) {
                throw new Error('A CSV export handler was not supplied to handle CSV export');
            }

            this.setState({ csvIsDownloading: true });
            customCsvExportHandler(csvName)
                .catch((err) => {
                    toast(`An error occurred while trying to export: ${err}`);
                })
                .finally(() => {
                    this.setState({ toggleWidget: false, csvIsDownloading: false });
                });
        } else {
            // otherwise, use legacy compliance CSV export
            let queryStr = '';
            let value = null;

            // Support for StandardId & ClusterId only
            if (queryParamMap[type]) {
                if (id) {
                    value = id;
                }
                queryStr = `${queryParamMap[type]}=${value}`;
            }

            downloadCSV(csvName, complianceDownloadUrl, queryStr);
        }
    };

    isCsvSupported = () => {
        const { page, type } = this.props;

        const isVulnMgmtSupportedPage = checkVulnMgmtSupport(page, type);
        const isComplianceSupportedPage = checkComplianceSupport(page, type);

        return isVulnMgmtSupportedPage || isComplianceSupportedPage;
    };

    renderContent = () => {
        const csvButtonText =
            this.props.type === entityTypes.CVE
                ? 'Download CVES as CSV'
                : 'Download Evidence as CSV';
        const { toggleWidget, csvIsDownloading } = this.state;
        if (!toggleWidget) {
            return null;
        }

        const headerText = this.props.fileName;

        const fileName = addBrandedTimestampToString(headerText);

        const wrapperClass = !!this.props.pdfId && !this.isCsvSupported() ? 'min-w-64' : '';

        return !!this.props.pdfId || this.isCsvSupported() ? (
            <div className={`absolute right-0 z-20 flex flex-col text-base-600 ${wrapperClass}`}>
                <div className="arrow-up self-end mr-5" />
                <ul
                    className="bg-base-100 rounded"
                    style={{
                        borderColor: 'var(--pf-v5-global--primary-color--100)',
                        borderWidth: 2,
                    }}
                >
                    <li className="p-4 border-b">
                        <div className="flex">
                            {!!this.props.pdfId && (
                                <WorkflowPDFExportButton
                                    id={this.props.pdfId}
                                    className={`min-w-48 ${
                                        this.isCsvSupported() ? 'mr-2' : 'w-full'
                                    }`}
                                    tableOptions={this.props.tableOptions}
                                    fileName={fileName}
                                    pdfTitle={headerText}
                                    isExporting={this.props.isExporting}
                                    setIsExporting={this.props.setIsExporting}
                                />
                            )}
                            {this.isCsvSupported() && (
                                <Button
                                    variant="primary"
                                    onClick={this.downloadCSVFile}
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
    };

    toggleWidget = () => {
        this.setState(({ toggleWidget }) => ({ toggleWidget: !toggleWidget }));
    };

    render() {
        return (
            <div className="relative pl-2">
                <ButtonClassic
                    className={this.props.className}
                    disabled={this.props.disabled}
                    text="Export"
                    textCondensed="Export"
                    textClass={this.props.textClass}
                    icon={<Icon.FileText size="14" className="mx-1 lg:ml-1 lg:mr-3" />}
                    onClick={this.toggleWidget}
                />
                {this.renderContent()}
            </div>
        );
    }
}

export default onClickOutside(ExportButton);
