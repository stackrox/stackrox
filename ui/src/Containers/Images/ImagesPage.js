import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';
import Collapsible from 'react-collapsible';
import dateFns from 'date-fns';
import ReactTooltip from 'react-tooltip';
import reduce from 'lodash/reduce';
import { Link } from 'react-router-dom';
import ReactTable from 'react-table';
import 'react-table/react-table.css';

import { selectors } from 'reducers';
import { actions as imagesActions } from 'reducers/images';

import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import Table from 'Components/Table';
import Panel from 'Components/Panel';
import KeyValuePairs from 'Components/KeyValuePairs';
import DockerFileModal from 'Containers/Images/DockerFileModal';
import { sortNumber, sortDate } from 'sorters/sorters';

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

const reducer = (action, prevState) => {
    switch (action) {
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
        images: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
        setSearchOptions: PropTypes.func.isRequired,
        setSearchModifiers: PropTypes.func.isRequired,
        setSearchSuggestions: PropTypes.func.isRequired,
        isViewFiltered: PropTypes.bool.isRequired,
        history: ReactRouterPropTypes.history.isRequired,
        location: ReactRouterPropTypes.location.isRequired,
        match: ReactRouterPropTypes.match.isRequired
    };

    constructor(props) {
        super(props);

        this.state = {
            modalOpen: false
        };
    }

    getSelectedImage = () => {
        if (this.props.match.params.sha) {
            return this.props.images.find(image => image.name.sha === this.props.match.params.sha);
        }
        return null;
    };

    openModal = () => {
        this.update('OPEN_MODAL');
    };

    closeModal = () => {
        this.update('CLOSE_MODAL');
    };

    update = action => {
        this.setState(prevState => reducer(action, prevState));
    };

    updateSelectedImage = image => {
        const urlSuffix = image && image.name && image.name.sha ? `/${image.name.sha}` : '';
        this.props.history.push({
            pathname: `/main/images${urlSuffix}`,
            search: this.props.location.search
        });
    };

    renderTable() {
        const columns = [
            { key: 'name.fullName', label: 'Image' },
            {
                key: 'metadata.created',
                label: 'Created at',
                default: '-',
                keyValueFunc: timestamp =>
                    timestamp ? dateFns.format(timestamp, 'MM/DD/YYYY h:mm:ss A') : '-',
                sortMethod: sortDate('metadata.created')
            },
            {
                key: 'scanComponentsLength',
                label: 'Components',
                default: '-',
                keyValueFunc: componentsLength => componentsLength || '-',
                sortMethod: sortNumber('scanComponentsLength')
            },
            {
                key: 'scanComponentsSum',
                label: 'CVEs',
                default: '-',
                keyValueFunc: componentsSum => componentsSum || '-',
                sortMethod: sortNumber('scanComponentsSum')
            }
        ];
        const rows = this.props.images;
        return <Table columns={columns} rows={rows} onRowClick={this.updateSelectedImage} />;
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
        const selectedImage = this.getSelectedImage();
        if (!selectedImage) return '';
        const header = selectedImage.name.fullName;
        return (
            <Panel header={header} onClose={this.updateSelectedImage} width="w-2/3">
                <div className="flex flex-col overflow-y-scroll w-full bg-primary-100">
                    {this.renderOverview()}
                    {this.renderCVEs()}
                </div>
            </Panel>
        );
    };

    renderOverview = () => {
        const title = 'OVERVIEW';
        const selectedImage = this.getSelectedImage();
        if (!selectedImage) return null;
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
        const selectedImage = this.getSelectedImage();
        const { scan } = selectedImage;
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
        const selectedImage = this.getSelectedImage();
        if (!this.state.modalOpen || !selectedImage || !selectedImage.metadata) return null;
        return <DockerFileModal data={selectedImage.metadata.layers} onClose={this.closeModal} />;
    }

    render() {
        const subHeader = this.props.isViewFiltered ? 'Filtered view' : 'Default view';
        return (
            <section className="flex flex-1 h-full">
                <div className="flex flex-1 flex-col">
                    <PageHeader header="Images" subHeader={subHeader}>
                        <SearchInput
                            searchOptions={this.props.searchOptions}
                            searchModifiers={this.props.searchModifiers}
                            searchSuggestions={this.props.searchSuggestions}
                            setSearchOptions={this.props.setSearchOptions}
                            setSearchModifiers={this.props.setSearchModifiers}
                            setSearchSuggestions={this.props.setSearchSuggestions}
                        />
                    </PageHeader>
                    <div className="flex flex-1">
                        <div className="w-full p-3 overflow-y-scroll bg-white rounded-sm shadow bg-base-100">
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

const isViewFiltered = createSelector(
    [selectors.getImagesSearchOptions],
    searchOptions => searchOptions.length !== 0
);

const mapStateToProps = createStructuredSelector({
    images: selectors.getImages,
    searchOptions: selectors.getImagesSearchOptions,
    searchModifiers: selectors.getImagesSearchModifiers,
    searchSuggestions: selectors.getImagesSearchSuggestions,
    isViewFiltered
});

const mapDispatchToProps = dispatch => ({
    setSearchOptions: searchOptions =>
        dispatch(imagesActions.setImagesSearchOptions(searchOptions)),
    setSearchModifiers: searchModifiers =>
        dispatch(imagesActions.setImagesSearchModifiers(searchModifiers)),
    setSearchSuggestions: searchSuggestions =>
        dispatch(imagesActions.setImagesSearchSuggestions(searchSuggestions))
});

export default connect(mapStateToProps, mapDispatchToProps)(ImagesPage);
