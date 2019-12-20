import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { format } from 'date-fns';
import onClickOutside from 'react-onclickoutside';

import downloadCsv from 'services/ComplianceDownloadService';
import PDFExportButton from 'Components/PDFExportButton';
import Button from 'Components/Button';
import useCaseTypes from 'constants/useCaseTypes';

const btnClassName =
    'btn border-primary-600 bg-primary-600 text-base-100 w-48 hover:bg-primary-700 hover:border-primary-700';
const queryParamMap = {
    CLUSTER: 'clusterId',
    STANDARD: 'standardId',
    ALL: ''
};

const downloadUrl = '/api/compliance/export/csv';

class ExportButton extends Component {
    static propTypes = {
        className: PropTypes.string,
        textClass: PropTypes.string,
        fileName: PropTypes.string,
        type: PropTypes.string,
        id: PropTypes.string,
        pdfId: PropTypes.string,
        tableOptions: PropTypes.shape({}),
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
        page: '',
        disabled: false
    };

    state = {
        toggleWidget: false
    };

    handleClickOutside = () => this.setState({ toggleWidget: false });

    downloadCsv = () => {
        const { id, fileName, type } = this.props;
        let query = {};
        let value = null;
        // Support for StandardId & ClusterId only
        if (queryParamMap[type]) {
            if (id) {
                value = id;
            }
            query = { [queryParamMap[type]]: value };
        }

        downloadCsv(query, fileName, downloadUrl);
    };

    isTypeSupported = () => {
        const { type, page } = this.props;
        return Object.keys(queryParamMap).includes(type) && page === useCaseTypes.COMPLIANCE;
    };

    renderContent = () => {
        const { toggleWidget } = this.state;
        if (!toggleWidget) return null;

        const headerText = this.props.fileName;

        const fileName = `StackRox:${headerText}-${format(new Date(), 'MM/DD/YYYY')}`;

        return (
            <div className="absolute pin-r pin-r z-20 uppercase flex flex-col text-base-600 min-w-64">
                <div className="arrow-up self-end mr-5" />
                <ul className="list-reset bg-base-100 border-2 border-primary-600 rounded">
                    <li className="p-4 border-b border-base-400">
                        <div className="flex uppercase">
                            <PDFExportButton
                                id={this.props.pdfId}
                                className={`${btnClassName}  ${
                                    this.isTypeSupported() ? 'mr-2' : 'w-full'
                                }`}
                                tableOptions={this.props.tableOptions}
                                fileName={fileName}
                                pdfTitle={headerText}
                            />
                            {this.isTypeSupported() && (
                                <button
                                    data-test-id="download-csv-button"
                                    className={btnClassName}
                                    type="button"
                                    onClick={this.downloadCsv}
                                >
                                    Download Evidence as CSV
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
