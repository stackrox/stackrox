import { Component } from 'react';
import PropTypes from 'prop-types';
import { FileText } from 'react-feather';
import onClickOutside from 'react-onclickoutside';
import { Button } from '@patternfly/react-core';

import downloadCSV from 'services/CSVDownloadService';
import ButtonClassic from 'Components/Button';
import entityTypes from 'constants/entityTypes';
import useCaseTypes from 'constants/useCaseTypes';
import { addBrandedTimestampToString } from 'utils/dateUtils';

const queryParamMap = {
    [entityTypes.CLUSTER]: 'clusterId',
    [entityTypes.STANDARD]: 'standardId',
    ALL: '',
};

class ExportButton extends Component {
    static propTypes = {
        className: PropTypes.string,
        textClass: PropTypes.string,
        fileName: PropTypes.string,
        type: PropTypes.string,
        id: PropTypes.string,
        page: PropTypes.string,
        disabled: PropTypes.bool,
    };

    static defaultProps = {
        className: 'btn btn-base h-10',
        textClass: null,
        fileName: 'compliance',
        type: null,
        id: '',
        page: '',
        disabled: false,
    };

    constructor(props) {
        super(props);

        this.state = {
            toggleWidget: false,
        };
    }

    handleClickOutside = () => this.setState({ toggleWidget: false });

    downloadCSVFile = () => {
        const { id, fileName, type } = this.props;
        const csvName = addBrandedTimestampToString(fileName);

        let queryStr = '';
        if (queryParamMap[type] && id) {
            queryStr = `${queryParamMap[type]}=${id}`;
        }

        downloadCSV(csvName, '/api/compliance/export/csv', queryStr);
    };

    isSupported = () => {
        const { page, type } = this.props;
        return page === useCaseTypes.COMPLIANCE && Object.keys(queryParamMap).includes(type);
    };

    renderContent = () => {
        if (!this.state.toggleWidget) {
            return null;
        }

        return this.isSupported() ? (
            <div className="absolute right-0 z-20 flex flex-col text-base-600">
                <div className="arrow-up self-end mr-5" />
                <ul
                    className="bg-base-100 rounded"
                    style={{
                        borderColor: 'var(--pf-t--global--border--color--brand--default)',
                        borderWidth: 2,
                    }}
                >
                    <li className="p-4 border-b">
                        <div className="flex">
                            <Button variant="primary" onClick={this.downloadCSVFile}>
                                Download Evidence as CSV
                            </Button>
                        </div>
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
                    icon={<FileText size="14" className="mx-1 lg:ml-1 lg:mr-3" />}
                    onClick={this.toggleWidget}
                />
                {this.renderContent()}
            </div>
        );
    }
}

export default onClickOutside(ExportButton);
