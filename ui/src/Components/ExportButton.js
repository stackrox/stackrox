import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import onClickOutside from 'react-onclickoutside';

import downloadCsv from 'services/ComplianceDownloadService';
import PDFExportButton from 'Components/PDFExportButton';
import Button from 'Components/Button';
import useCaseTypes from 'constants/useCaseTypes';
import entityTypes from 'constants/entityTypes';
import { addBrandedTimestampToString } from 'utils/dateUtils';

const btnClassName =
    'btn border-primary-600 bg-primary-600 text-base-100 w-48 hover:bg-primary-700 hover:border-primary-700';
const queryParamMap = {
    CLUSTER: 'clusterId',
    STANDARD: 'standardId',
    ALL: ''
};

const complianceDownloadUrl = '/api/compliance/export/csv';

function checkVulnMgmtSupport(page, type) {
    return page === useCaseTypes.VULN_MANAGEMENT && type === entityTypes.CVE;
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
        disabled: PropTypes.bool
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
        disabled: false
    };

    state = {
        toggleWidget: false
    };

    handleClickOutside = () => this.setState({ toggleWidget: false });

    downloadCsv = () => {
        const { id, fileName, page, type, customCsvExportHandler } = this.props;
        const csvName = addBrandedTimestampToString(fileName);

        if (checkVulnMgmtSupport(page, type)) {
            if (!customCsvExportHandler) {
                throw new Error('A CSV export handler was not supplied to handle CSV export');
            }

            customCsvExportHandler(csvName).finally(() => {
                this.setState({ toggleWidget: false });
            });
        } else {
            // otherwise, use legacy compliance CSV export
            let query = {};
            let value = null;

            // Support for StandardId & ClusterId only
            if (queryParamMap[type]) {
                if (id) {
                    value = id;
                }
                query = { [queryParamMap[type]]: value };
            }

            downloadCsv(query, csvName, complianceDownloadUrl);
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
        const { toggleWidget } = this.state;
        if (!toggleWidget) return null;

        const headerText = this.props.fileName;

        const fileName = addBrandedTimestampToString(headerText);

        return (
            <div className="absolute right-0 z-20 uppercase flex flex-col text-base-600 min-w-64">
                <div className="arrow-up self-end mr-5" />
                <ul className=" bg-base-100 border-2 border-primary-600 rounded">
                    <li className="p-4 border-b border-base-400">
                        <div className="flex uppercase">
                            <PDFExportButton
                                id={this.props.pdfId}
                                className={`${btnClassName}  ${
                                    this.isCsvSupported() ? 'mr-2' : 'w-full'
                                }`}
                                tableOptions={this.props.tableOptions}
                                fileName={fileName}
                                pdfTitle={headerText}
                            />
                            {this.isCsvSupported() && (
                                <button
                                    data-test-id="download-csv-button"
                                    className={btnClassName}
                                    type="button"
                                    onClick={this.downloadCsv}
                                >
                                    {csvButtonText}
                                </button>
                            )}
                        </div>
                    </li>
                    <li className="hidden">
                        <span>or share to</span>
                        <div>Slack</div>
                    </li>
                </ul>
            </div>
        );
    };

    openWidget = () => {
        this.setState({ toggleWidget: true });
    };

    render() {
        return (
            <div className="relative pl-2">
                <Button
                    className={this.props.className}
                    disabled={this.props.disabled}
                    text="Export"
                    textCondensed="Export"
                    textClass={this.props.textClass}
                    icon={<Icon.FileText size="14" className="mx-1 lg:ml-1 lg:mr-3" />}
                    onClick={this.openWidget}
                />
                {this.renderContent()}
            </div>
        );
    }
}

export default onClickOutside(ExportButton);
