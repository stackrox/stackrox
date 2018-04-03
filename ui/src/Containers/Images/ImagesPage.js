import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';
import Collapsible from 'react-collapsible';
import dateFns from 'date-fns';
import ReactTooltip from 'react-tooltip';
import reduce from 'lodash/reduce';
import { Link } from 'react-router-dom';
import ReactTable from 'react-table';
import 'react-table/react-table.css';

import { selectors } from 'reducers';

import Table from 'Components/Table';
import Panel from 'Components/Panel';
import KeyValuePairs from 'Components/KeyValuePairs';
import DockerFileModal from 'Containers/Images/DockerFileModal';

const imageDetailsMap = {
    scanTime: {
        label: 'Last scan time',
        formatValue: timestamp =>
            timestamp ? dateFns.format(timestamp, 'MM/DD/YYYY h:mm:ss A') : 'not available'
    },
    sha: {
        label: 'SHA'
    },
    totalComponents: {
        label: 'Components'
    },
    totalCVEs: {
        label: 'CVEs',
        formatValue: arr => reduce(arr, (sum, component) => sum + component.vulns.length, 0)
    }
};

const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'SELECT_IMAGE':
            return { selectedImage: nextState.image };
        case 'UNSELECT_IMAGE':
            return { selectedImage: null };
        case 'OPEN_MODAL':
            return { modalOpen: true };
        case 'CLOSE_MODAL':
            return { modalOpen: false };
        default:
            return prevState;
    }
};

class ImagesPage extends Component {
    static propTypes = {
        images: PropTypes.arrayOf(PropTypes.object).isRequired
    };

    constructor(props) {
        super(props);

        this.state = {
            selectedImage: null,
            modalOpen: false
        };
    }

    onImageClick = image => {
        this.selectImage(image);
    };

    selectImage = image => {
        this.update('SELECT_IMAGE', { image });
    };

    unselectImage = () => {
        this.update('UNSELECT_IMAGE');
    };

    openModal = () => {
        this.update('OPEN_MODAL');
    };

    closeModal = () => {
        this.update('CLOSE_MODAL');
    };

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    };

    renderTable() {
        const columns = [
            { key: 'name.fullName', label: 'Image' },
            {
                key: 'metadata.created',
                label: 'Created at',
                default: '-',
                keyValueFunc: timestamp =>
                    timestamp ? dateFns.format(timestamp, 'MM/DD/YYYY h:mm:ss A') : '-'
            },
            { key: 'scan.components.length', label: 'Components', default: '-' },
            {
                key: 'scan.components',
                label: 'CVEs',
                default: '-',
                keyValueFunc: arr =>
                    reduce(arr, (sum, component) => sum + component.vulns.length, 0)
            }
        ];
        const rows = this.props.images;
        return <Table columns={columns} rows={rows} onRowClick={this.onImageClick} />;
    }

    renderCollapsibleCard = (title, direction) => {
        const icons = {
            up: <Icon.ChevronUp className="h-4 w-4" />,
            down: <Icon.ChevronDown className="h-4 w-4" />
        };

        return (
            <div className="p-3 border-b border-base-300 text-primary-600 tracking-wide cursor-pointer flex justify-between">
                <div>{title}</div>
                <div>{icons[direction]}</div>
            </div>
        );
    };

    renderSidePanel = () => {
        if (!this.state.selectedImage) return '';
        const { selectedImage } = this.state;
        const header = selectedImage.name.fullName;
        return (
            <Panel header={header} onClose={this.unselectImage} width="w-2/3">
                <div className="flex flex-col overflow-y-scroll w-full bg-primary-100">
                    {this.renderOverview()}
                    {this.renderCVEs()}
                </div>
            </Panel>
        );
    };

    renderOverview = () => {
        const title = 'OVERVIEW';
        const { selectedImage } = this.state;
        const imageDetail = {
            scanTime: selectedImage.scan ? selectedImage.scan.scanTime : '',
            sha: selectedImage.name.sha,
            totalComponents: selectedImage.scan
                ? selectedImage.scan.components.length
                : 'not available',
            totalCVEs: selectedImage.scan ? selectedImage.scan.components : []
        };
        return (
            <div className="px-3 py-4">
                <div className="alert-preview bg-white shadow text-primary-600 tracking-wide">
                    <Collapsible
                        open
                        trigger={this.renderCollapsibleCard(title, 'up')}
                        triggerWhenOpen={this.renderCollapsibleCard(title, 'down')}
                        transitionTime={100}
                    >
                        <div className="h-full">
                            <div className="p-3">
                                <KeyValuePairs data={imageDetail} keyValueMap={imageDetailsMap} />
                            </div>
                            <div className="flex bg-primary-100">
                                <span className="w-1/2">
                                    <Link
                                        className="flex mx-auto my-2 py-3 px-2 w-5/6 rounded-sm text-primary-600 no-underline hover:text-white hover:bg-primary-400 uppercase justify-center text-sm items-center bg-white border-2 border-primary-400"
                                        to="/main/risk"
                                    >
                                        View Deployments
                                    </Link>
                                </span>
                                <span
                                    className="w-1/2 border-low-100 border-l-2"
                                    data-tip
                                    data-tip-disable={selectedImage.metadata}
                                    data-for="button-DockerFile"
                                >
                                    <button
                                        className="flex mx-auto my-2 py-3 px-2 w-5/6 rounded-sm text-primary-600 tracking-wide hover:text-white hover:bg-primary-400 uppercase justify-center text-sm items-center bg-white border-2 border-primary-400"
                                        onClick={this.openModal}
                                        disabled={!selectedImage.metadata}
                                    >
                                        View Docker File
                                    </button>
                                    {!selectedImage.metadata && (
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
                    </Collapsible>
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
                            className="text-primary-600 font-600"
                        >
                            {ci.value}
                        </a>
                        - {ci.original.summary}
                    </div>
                ),
                headerClassName: 'font-600'
            },
            {
                Header: 'CVSS',
                accessor: 'cvss',
                width: 50,
                headerClassName: 'font-600 text-right',
                className: 'text-right'
            }
        ];
        return (
            row.original.vulns.length !== 0 && (
                <ReactTable
                    data={row.original.vulns}
                    columns={subColumns}
                    pageSize={row.original.vulns.length}
                    showPagination={false}
                    className="bg-base-100"
                    resizable={false}
                />
            )
        );
    };

    renderCVEs = () => {
        const title = 'CVEs';
        const columns = [
            {
                expander: true,
                width: 30,
                className: 'pointer-events-none',
                Expander: ({ isExpanded, ...rest }) => {
                    if (rest.original.vulns.length === 0) return '';
                    return (
                        <div>
                            {isExpanded ? (
                                <div className="rt-expander w-1 -open pointer-events-auto">
                                    &#8226;
                                </div>
                            ) : (
                                <div className="rt-expander w-1 pointer-events-auto">&#8226;</div>
                            )}
                        </div>
                    );
                }
            },
            {
                Header: 'Name',
                accessor: 'name',
                headerClassName: 'font-600 text-left',
                Cell: ci => <div>{ci.value}</div>
            },
            {
                Header: 'Version',
                accessor: 'version',
                className: 'text-right',
                headerClassName: 'font-600 text-right'
            },
            {
                Header: 'CVEs',
                accessor: 'vulns.length',
                width: 50,
                className: 'text-right',
                headerClassName: 'font-600 text-right'
            }
        ];
        const { scan } = this.state.selectedImage;
        return (
            <div className="px-3 py-4">
                <div className="alert-preview bg-white shadow text-primary-600 tracking-wide">
                    <Collapsible
                        open
                        trigger={this.renderCollapsibleCard(title, 'up')}
                        triggerWhenOpen={this.renderCollapsibleCard(title, 'down')}
                        transitionTime={100}
                    >
                        <div className="h-full p-3 font-500">
                            {scan && (
                                <ReactTable
                                    data={scan.components}
                                    columns={columns}
                                    showPagination={false}
                                    defaultPageSize={scan.components.length}
                                    SubComponent={this.renderVulnsTable}
                                />
                            )}
                            {!scan && (
                                <div className="font-500">No scanner setup for this registry</div>
                            )}
                        </div>
                    </Collapsible>
                </div>
            </div>
        );
    };

    renderDockerFileModal() {
        if (!this.state.modalOpen || !this.state.selectedImage.metadata) return null;
        return (
            <DockerFileModal
                data={this.state.selectedImage.metadata.layers}
                onClose={this.closeModal}
            />
        );
    }

    render() {
        return (
            <section className="flex flex-1 h-full">
                <div className="flex flex-1 mt-3 flex-col">
                    <div className="flex mb-3 mx-3 self-end justify-end" />
                    <div className="flex flex-1">
                        <div className="w-full p-3 overflow-y-scroll bg-white rounded-sm shadow border-t border-primary-300 bg-base-100">
                            {this.renderTable()}
                        </div>
                        {this.renderSidePanel()}
                        {this.renderDockerFileModal()}
                    </div>
                </div>
            </section>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    images: selectors.getImages
});

export default connect(mapStateToProps)(ImagesPage);
