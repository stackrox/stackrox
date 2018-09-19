import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import ReactTooltip from 'react-tooltip';
import 'react-table/react-table.css';
import Table, { defaultHeaderClassName } from 'Components/Table';

import { actions as deploymentsActions } from 'reducers/deployments';
import { addSearchModifier, addSearchKeyword } from 'utils/searchUtils';

import CollapsibleCard from 'Components/CollapsibleCard';
import Panel from 'Components/Panel';
import KeyValuePairs from 'Components/KeyValuePairs';
import DockerFileModal from 'Containers/Images/DockerFileModal';
import Loader from 'Components/Loader';

const imageDetailsMap = {
    scanTime: {
        label: 'Last scan time',
        formatValue: timestamp =>
            timestamp ? dateFns.format(timestamp, dateTimeFormat) : 'not available'
    },
    sha: {
        label: 'SHA',
        className: 'word-break'
    },
    totalComponents: {
        label: 'Components'
    },
    totalCVEs: {
        label: 'CVEs'
    }
};

class ImageDetails extends Component {
    static propTypes = {
        image: PropTypes.shape({
            name: PropTypes.string.isRequired,
            scan: PropTypes.shape({})
        }).isRequired,
        loading: PropTypes.bool.isRequired,
        history: ReactRouterPropTypes.history.isRequired,
        location: ReactRouterPropTypes.location.isRequired,
        setDeploymentsSearchOptions: PropTypes.func.isRequired
    };

    constructor(props) {
        super(props);

        this.state = {
            modalOpen: false
        };
    }

    onViewDeploymentsClick = () => {
        const { name } = this.props.image;
        let searchOptions = [];
        searchOptions = addSearchModifier(searchOptions, 'Image:');
        searchOptions = addSearchKeyword(searchOptions, name);
        this.props.setDeploymentsSearchOptions(searchOptions);
        this.props.history.push('/main/risk');
    };

    openModal = () => {
        this.setState({ modalOpen: true });
    };

    closeModal = () => {
        this.setState({ modalOpen: false });
    };

    updateSelectedImage = image => {
        const urlSuffix = image && image.sha ? `/${image.sha}` : '';
        this.props.history.push({
            pathname: `/main/images${urlSuffix}`,
            search: this.props.location.search
        });
    };

    renderOverview = () => {
        const title = 'Overview';
        const { image } = this.props;
        if (!image) return null;
        const imageDetail = {
            scanTime: image.scan ? image.scan.scanTime : '',
            sha: image.sha,
            totalComponents: image.components ? image.components : 'not available',
            totalCVEs: image.cves ? image.cves : []
        };
        return (
            <div className="px-3 py-4">
                <div className="alert-preview bg-white shadow text-primary-600">
                    <CollapsibleCard title={title}>
                        <div className="h-full">
                            <div className="p-3">
                                <KeyValuePairs data={imageDetail} keyValueMap={imageDetailsMap} />
                            </div>
                            <div className="flex bg-primary-100">
                                <span className="w-1/2">
                                    <button
                                        className="flex mx-auto my-2 py-3 px-2 w-5/6 rounded-sm text-primary-600 no-underline hover:text-white hover:bg-primary-400 uppercase justify-center text-sm items-center bg-white border-2 border-primary-400"
                                        onClick={this.onViewDeploymentsClick}
                                    >
                                        View Deployments
                                    </button>
                                </span>
                                <span
                                    className="w-1/2 border-low-100 border-l-2"
                                    data-tip
                                    data-tip-disable={image.metadata}
                                    data-for="button-DockerFile"
                                >
                                    <button
                                        className="flex mx-auto my-2 py-3 px-2 w-5/6 rounded-sm text-primary-600 hover:text-white hover:bg-primary-400 uppercase justify-center text-sm items-center bg-white border-2 border-primary-400"
                                        onClick={this.openModal}
                                        disabled={!image.metadata}
                                    >
                                        View Docker File
                                    </button>
                                    {!image.metadata && (
                                        <ReactTooltip
                                            id="button-DockerFile"
                                            type="dark"
                                            effect="solid"
                                        >
                                            Docker file not available
                                        </ReactTooltip>
                                    )}
                                </span>
                            </div>
                        </div>
                    </CollapsibleCard>
                </div>
            </div>
        );
    };

    renderVulnsTable = row => {
        const subColumns = [
            {
                Header: 'CVE',
                accessor: 'cve',
                Cell: ci => (
                    <div className="truncate">
                        <a
                            href={ci.original.link}
                            target="_blank"
                            className="text-primary-600 font-600 pointer-events-auto"
                        >
                            {ci.value}
                        </a>
                        - {ci.original.summary}
                    </div>
                ),
                headerClassName: 'font-600 border-b border-base-300',
                className: 'pointer-events-none'
            },
            {
                Header: 'CVSS',
                accessor: 'cvss',
                width: 50,
                headerClassName: 'font-600 border-b border-base-300',
                className: 'text-right self-center pointer-events-none'
            },
            {
                Header: 'Fixed',
                accessor: 'fixedBy',
                width: 130,
                headerClassName: 'font-600 border-b border-base-300',
                className: 'text-right self-center pointer-events-none'
            }
        ];
        return (
            row.original.vulns.length !== 0 && (
                <Table
                    rows={row.original.vulns}
                    columns={subColumns}
                    className="bg-base-100"
                    showPagination={false}
                    pageSize={row.original.vulns.length}
                />
            )
        );
    };

    renderCVEs = () => {
        const title = 'CVEs';
        return (
            <div className="px-3 py-4">
                <div className="alert-preview bg-white shadow text-primary-600">
                    <CollapsibleCard title={title}>
                        <div className="h-full p-3 font-500"> {this.renderCVEsTable()}</div>
                    </CollapsibleCard>
                </div>
            </div>
        );
    };

    renderCVEsTable = () => {
        const { scan } = this.props.image;
        if (!scan) return <div className="font-500">No scanner setup for this registry</div>;

        const columns = [
            {
                expander: true,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: 'w-1/8 pointer-events-none self-center',
                Expander: ({ isExpanded, ...rest }) => {
                    if (rest.original.vulns.length === 0) return '';
                    const className = 'rt-expander w-1 pt-2 pointer-events-auto';
                    return <div className={`${className} ${isExpanded ? '-open' : ''}`} />;
                }
            },
            {
                Header: 'Name',
                accessor: 'name',
                headerClassName: 'pl-3 font-600 text-left border-b border-base-300 border-r-0',
                Cell: ci => <div>{ci.value}</div>
            },
            {
                Header: 'Version',
                accessor: 'version',
                className: 'text-right pr-4 self-center',
                headerClassName: 'font-600 text-right border-b border-base-300 border-r-0 pr-4'
            },
            {
                Header: 'CVEs',
                accessor: 'vulns.length',
                className: 'w-1/8 text-right pr-4 self-center',
                headerClassName:
                    'w-1/8 font-600 text-right border-b border-base-300 border-r-0 pr-4'
            }
        ];
        return (
            <Table rows={scan.components} columns={columns} SubComponent={this.renderVulnsTable} />
        );
    };

    renderDockerFileModal() {
        const { image } = this.props;
        if (!this.state.modalOpen || !image || !image.metadata) return null;
        return <DockerFileModal data={image.metadata.layers} onClose={this.closeModal} />;
    }

    render() {
        const { image, loading } = this.props;
        if (!image) return '';
        const header = image.name;
        const content = loading ? (
            <Loader />
        ) : (
            <div className="flex flex-col overflow-y-scroll w-full bg-primary-100">
                {this.renderOverview()}
                {this.renderCVEs()}
                {this.renderDockerFileModal()}
            </div>
        );
        return (
            <Panel header={header} onClose={this.updateSelectedImage} className="w-2/3">
                {content}
            </Panel>
        );
    }
}

const mapDispatchToProps = {
    setDeploymentsSearchOptions: deploymentsActions.setDeploymentsSearchOptions
};

export default withRouter(connect(null, mapDispatchToProps)(ImageDetails));
