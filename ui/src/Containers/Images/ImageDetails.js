import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import dateFns from 'date-fns';
import Tooltip from 'rc-tooltip';

import dateTimeFormat from 'constants/dateTimeFormat';

import { actions as deploymentsActions } from 'reducers/deployments';
import { addSearchModifier, addSearchKeyword } from 'utils/searchUtils';

import Table, { defaultHeaderClassName } from 'Components/Table';
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

    getDockerFileButton = image => {
        const button = (
            <button
                type="button"
                className="flex mx-auto my-2 py-3 px-2 w-5/6 rounded-sm text-primary-600 hover:text-base-100 hover:bg-primary-400 uppercase justify-center text-sm items-center bg-base-100 border-2 border-primary-400"
                onClick={this.openModal}
                disabled={!image.metadata}
            >
                View Docker File
            </button>
        );
        if (image.metadata) return button;
        return (
            <Tooltip placement="top" overlay={<div>Docker file not available</div>}>
                <div>{button}</div>
            </Tooltip>
        );
    };

    openModal = () => {
        this.setState({ modalOpen: true });
    };

    closeModal = () => {
        this.setState({ modalOpen: false });
    };

    updateSelectedImage = image => {
        const urlSuffix = image && image.id ? `/${image.id}` : '';
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
            id: image.id,
            totalComponents: image.components ? image.components : 'not available',
            totalCVEs: image.cves ? image.cves : []
        };
        return (
            <div className="px-3 pt-5">
                <div className="alert-preview bg-base-100 shadow text-primary-600">
                    <CollapsibleCard title={title}>
                        <div className="h-full">
                            <div className="px-3">
                                <KeyValuePairs data={imageDetail} keyValueMap={imageDetailsMap} />
                            </div>
                            <div className="flex bg-primary-100">
                                <span className="w-1/2">
                                    <button
                                        type="button"
                                        className="flex mx-auto my-2 py-3 px-2 w-5/6 rounded-sm text-primary-600 no-underline hover:text-base-100 hover:bg-primary-400 uppercase justify-center text-sm items-center bg-base-100 border-2 border-primary-400"
                                        onClick={this.onViewDeploymentsClick}
                                    >
                                        View Deployments
                                    </button>
                                </span>
                                <span className="w-1/2 border-base-300 border-l-2">
                                    {this.getDockerFileButton(image)}
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
                            rel="noopener noreferrer"
                            className="text-primary-600 font-600 pointer-events-auto"
                        >
                            {ci.value}
                        </a>
                        - {ci.original.summary}
                    </div>
                ),
                headerClassName: 'font-600 border-b border-base-300 flex items-end',
                className: 'pointer-events-none flex items-center justify-end'
            },
            {
                Header: 'CVSS',
                accessor: 'cvss',
                width: 50,
                headerClassName: 'font-600 border-b border-base-300 flex items-end justify-end',
                className: 'pointer-events-none flex items-center justify-end'
            },
            {
                Header: 'Fixed',
                accessor: 'fixedBy',
                width: 130,
                headerClassName: 'font-600 border-b border-base-300 flex items-end',
                className: 'pointer-events-none flex items-center justify-end'
            }
        ];
        return (
            row.original.vulns.length !== 0 && (
                <Table
                    rows={row.original.vulns}
                    columns={subColumns}
                    className="bg-base-200"
                    showPagination={false}
                    pageSize={row.original.vulns.length}
                />
            )
        );
    };

    renderCVEs = () => {
        const title = 'CVEs';
        return (
            <div className="px-3 pt-5">
                <div className="alert-preview bg-base-100 shadow text-primary-600">
                    <CollapsibleCard title={title}>
                        <div className="h-full p-3"> {this.renderCVEsTable()}</div>
                    </CollapsibleCard>
                </div>
            </div>
        );
    };

    renderCVEsTable = () => {
        const { scan } = this.props.image;
        if (!scan) return <div>No scanner setup for this registry</div>;

        const columns = [
            {
                expander: true,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: 'w-1/8 pointer-events-none flex items-center justify-end',
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
                className: 'pr-4 flex items-center justify-end',
                headerClassName: 'font-600 text-right border-b border-base-300 border-r-0 pr-4'
            },
            {
                Header: 'CVEs',
                accessor: 'vulns.length',
                className: 'w-1/8 pr-4 flex items-center justify-end',
                headerClassName:
                    'w-1/8 font-600 text-right border-b border-base-300 border-r-0 pr-4'
            }
        ];
        return (
            <Table
                defaultPageSize={scan.components.length}
                rows={scan.components}
                columns={columns}
                SubComponent={this.renderVulnsTable}
            />
        );
    };

    renderDockerFileModal() {
        const { image } = this.props;
        if (!this.state.modalOpen || !image || !image.metadata || !image.metadata.v1) return null;
        return <DockerFileModal data={image.metadata.v1.layers} onClose={this.closeModal} />;
    }

    render() {
        const { image, loading } = this.props;
        if (!image) return '';
        const header = image.name;
        const content = loading ? (
            <Loader />
        ) : (
            <div className="flex flex-col w-full bg-base-200 overflow-auto pb-5">
                {this.renderOverview()}
                {this.renderCVEs()}
                {this.renderDockerFileModal()}
            </div>
        );
        return (
            <Panel
                header={header}
                onClose={this.updateSelectedImage}
                className="bg-primary-200 z-10 w-full h-full absolute pin-r pin-t md:w-1/2 min-w-72 md:relative"
            >
                {content}
            </Panel>
        );
    }
}

const mapDispatchToProps = {
    setDeploymentsSearchOptions: deploymentsActions.setDeploymentsSearchOptions
};

export default withRouter(
    connect(
        null,
        mapDispatchToProps
    )(ImageDetails)
);
