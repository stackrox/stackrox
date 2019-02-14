import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Button from 'Components/Button';
import * as Icon from 'react-feather';
import downloadCsv from 'services/ComplianceDownloadService';
import onClickOutside from 'react-onclickoutside';
import PDFExportButton from 'Components/PDFExportButton';
import { format } from 'date-fns';

const btnClassName = 'btn border-base-400 bg-base-400 text-base-100 w-48';
const selectedBtnClassName =
    'btn border-primary-800 bg-primary-800 text-base-100 w-48 hover:bg-primary-900';
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
        tableOptions: PropTypes.shape({})
    };

    static defaultProps = {
        className: 'btn btn-base h-10',
        textClass: null,
        fileName: 'compliance',
        type: null,
        id: '',
        pdfId: '',
        tableOptions: {}
    };

    state = {
        selectedFormat: 'pdf',
        toggleWidget: false
    };

    handleClickOutside = () => this.setState({ toggleWidget: false });

    selectDownloadFormat = selectedFormat => () => {
        this.setState({ selectedFormat });
    };

    downloadCsv = () => {
        const { id, fileName, type } = this.props;
        let query = {};
        let value = null;
        // Support for StandardId & ClusterId only
        if (queryParamMap[type]) {
            if (id) {
                value = id;
            } else {
                value = type;
            }
            query = { [queryParamMap[type]]: value };
        }

        downloadCsv(query, fileName, downloadUrl);
    };

    isTypeSupported = () => Object.keys(queryParamMap).includes(this.props.type);

    renderContent = () => {
        const { toggleWidget, selectedFormat } = this.state;
        if (!toggleWidget) return null;

        const headerText = this.props.fileName;

        const fileName = `StackRox:${headerText}-${format(new Date(), 'MM/DD/YYYY')}`;

        return (
            <div className="absolute pin-r pin-r z-10 uppercase flex flex-col text-base-600 min-w-64">
                <div className="arrow-up self-end mr-5" />
                <ul className="list-reset bg-base-100 border-2 border-primary-800 rounded">
                    <li className="p-4 border-b border-base-400">
                        <div className="flex uppercase">
                            <button
                                className={`${
                                    selectedFormat === 'pdf' ? selectedBtnClassName : btnClassName
                                }  ${this.isTypeSupported() ? 'mr-2' : 'w-full'}`}
                                type="button"
                                onClick={this.selectDownloadFormat('pdf')}
                            >
                                Page as PDF
                            </button>
                            {this.isTypeSupported() && (
                                <button
                                    className={
                                        selectedFormat === 'csv'
                                            ? selectedBtnClassName
                                            : btnClassName
                                    }
                                    type="button"
                                    onClick={this.selectDownloadFormat('csv')}
                                >
                                    Evidence as CSV
                                </button>
                            )}
                        </div>
                    </li>
                    <li className="p-4">
                        <div>
                            {selectedFormat === 'csv' && (
                                <button
                                    type="button"
                                    className={`${selectedBtnClassName} w-full`}
                                    onClick={this.downloadCsv}
                                >
                                    Download
                                </button>
                            )}
                            {selectedFormat === 'pdf' && (
                                <PDFExportButton
                                    id={this.props.pdfId}
                                    onClick={this.selectDownloadFormat('pdf')}
                                    className={`${selectedBtnClassName} w-full`}
                                    tableOptions={this.props.tableOptions}
                                    fileName={fileName}
                                    pdfTitle={headerText}
                                />
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
