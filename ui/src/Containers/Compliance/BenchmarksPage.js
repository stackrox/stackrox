import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { withRouter } from 'react-router-dom';
import { selectors } from 'reducers';
import { createSelector, createStructuredSelector } from 'reselect';
import { actions as benchmarkActions, types } from 'reducers/benchmarks';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import { ClipLoader } from 'react-spinners';
import { sortNumber } from 'sorters/sorters';

import NoResultsMessage from 'Components/NoResultsMessage';
import Table from 'Components/Table';
import Select from 'Components/Select';
import BenchmarksSidePanel from 'Containers/Compliance/BenchmarksSidePanel';
import HostResultModal from 'Containers/Compliance/HostResultModal';
import Loader from 'Components/Loader';

class BenchmarksPage extends Component {
    static propTypes = {
        benchmarkScanResults: PropTypes.arrayOf(PropTypes.object).isRequired,
        lastScannedTime: PropTypes.string.isRequired,
        lastScannedId: PropTypes.string.isRequired,
        benchmarkName: PropTypes.string.isRequired,
        benchmarkId: PropTypes.string.isRequired,
        clusterId: PropTypes.string.isRequired,
        startPollBenchmarkScanResults: PropTypes.func.isRequired,
        stopPollBenchmarkScanResults: PropTypes.func.isRequired,
        selectBenchmarkScheduleDay: PropTypes.func.isRequired,
        selectBenchmarkScheduleHour: PropTypes.func.isRequired,
        selectBenchmarkScanResult: PropTypes.func.isRequired,
        selectBenchmarkHostResult: PropTypes.func.isRequired,
        fetchBenchmarkSchedule: PropTypes.func.isRequired,
        schedule: PropTypes.shape({
            day: PropTypes.string,
            hour: PropTypes.string
        }).isRequired,
        triggerBenchmarkScan: PropTypes.func.isRequired,
        selectedBenchmarkScanResult: PropTypes.shape({
            definition: PropTypes.shape({ name: PropTypes.string }),
            hostResults: PropTypes.arrayOf(PropTypes.object)
        }),
        selectedBenchmarkHostResult: PropTypes.shape({
            host: PropTypes.string,
            notes: PropTypes.arrayOf(PropTypes.string)
        }),
        fetchBenchmarkCheckHostResults: PropTypes.func.isRequired,
        isFetchingBenchmarkCheckHostResults: PropTypes.bool
    };

    static defaultProps = {
        selectedBenchmarkScanResult: null,
        selectedBenchmarkHostResult: null,
        isFetchingBenchmarkCheckHostResults: false
    };

    constructor(props) {
        super(props);

        this.state = {
            scanning: false
        };
    }

    componentDidMount() {
        this.setUpComponent();
    }

    componentWillReceiveProps(nextProps) {
        if (nextProps.lastScannedTime !== this.props.lastScannedTime) {
            // if new benchmark results are loaded then stop the button scanning if it is scanning
            this.setState({ scanning: false });
        }
    }

    componentDidUpdate(prevProps) {
        if (prevProps.clusterId !== this.props.clusterId) {
            this.props.stopPollBenchmarkScanResults();
            this.setUpComponent();
        }
    }

    componentWillUnmount() {
        this.props.stopPollBenchmarkScanResults();
    }

    onTriggerScan = () => {
        this.setState({ scanning: true });
        this.props.triggerBenchmarkScan({
            benchmarkId: this.props.benchmarkId,
            clusterId: this.props.clusterId
        });
    };

    onRowClick = row => {
        this.props.fetchBenchmarkCheckHostResults({
            scanId: this.props.lastScannedId,
            checkName: row.definition.name
        });
        this.props.selectBenchmarkScanResult(row);
    };

    onCloseSidePanel = () => {
        this.props.selectBenchmarkScanResult(null);
    };

    onHostResultClick = benchmarkHostResult => {
        this.props.selectBenchmarkHostResult(benchmarkHostResult);
    };

    onBenchmarkHostResultModalClose = () => {
        this.props.selectBenchmarkHostResult(null);
    };

    onScheduleDayChange = value => {
        this.props.selectBenchmarkScheduleDay(
            this.props.benchmarkId,
            this.props.benchmarkName,
            value,
            this.props.clusterId
        );
    };

    onScheduleHourChange = value => {
        this.props.selectBenchmarkScheduleHour(
            this.props.benchmarkId,
            this.props.benchmarkName,
            value,
            this.props.clusterId
        );
    };

    setUpComponent = () => {
        this.props.startPollBenchmarkScanResults({
            benchmarkId: this.props.benchmarkId,
            clusterId: this.props.clusterId
        });
        this.props.fetchBenchmarkSchedule({
            benchmarkId: this.props.benchmarkId,
            clusterId: this.props.clusterId
        });
    };

    renderScanOptions = () => {
        const category = {
            options: [
                { label: 'None', value: null },
                { label: 'Monday', value: 'Monday' },
                { label: 'Tuesday', value: 'Tuesday' },
                { label: 'Wednesday', value: 'Wednesday' },
                { label: 'Thursday', value: 'Thursday' },
                { label: 'Friday', value: 'Friday' },
                { label: 'Saturday', value: 'Saturday' },
                { label: 'Sunday', value: 'Sunday' }
            ]
        };
        return (
            <Select
                className="bg-base-100 block border-2 border-base-400 hover:border-primary-400 cursor-pointer h-9 p-2 pr-8 rounded-sm w-full"
                value={this.props.schedule.day}
                placeholder="No scheduled scanning"
                options={category.options}
                onChange={this.onScheduleDayChange}
            />
        );
    };

    renderScanTimes = () => {
        const category = {
            options: [
                { label: '00:00 AM', value: '00:00 AM' },
                { label: '01:00 AM', value: '01:00 AM' },
                { label: '02:00 AM', value: '02:00 AM' },
                { label: '03:00 AM', value: '03:00 AM' },
                { label: '04:00 AM', value: '04:00 AM' },
                { label: '05:00 AM', value: '05:00 AM' },
                { label: '06:00 AM', value: '06:00 AM' },
                { label: '07:00 AM', value: '07:00 AM' },
                { label: '08:00 AM', value: '08:00 AM' },
                { label: '09:00 AM', value: '09:00 AM' },
                { label: '10:00 AM', value: '10:00 AM' },
                { label: '11:00 AM', value: '11:00 AM' },
                { label: '12:00 PM', value: '12:00 PM' },
                { label: '01:00 PM', value: '01:00 PM' },
                { label: '02:00 PM', value: '02:00 PM' },
                { label: '03:00 PM', value: '03:00 PM' },
                { label: '04:00 PM', value: '04:00 PM' },
                { label: '05:00 PM', value: '05:00 PM' },
                { label: '06:00 PM', value: '06:00 PM' },
                { label: '07:00 PM', value: '07:00 PM' },
                { label: '08:00 PM', value: '08:00 PM' },
                { label: '09:00 PM', value: '09:00 PM' },
                { label: '10:00 PM', value: '10:00 PM' },
                { label: '11:00 PM', value: '11:00 PM' }
            ]
        };
        return (
            <Select
                className="bg-base-100 block border-2 border-base-400 hover:border-primary-400 cursor-pointer h-9 p-2 pr-8 rounded-sm w-full"
                value={this.props.schedule.hour}
                placeholder="None"
                options={category.options}
                onChange={this.onScheduleHourChange}
            />
        );
    };

    renderScanButton = () => {
        const buttonScanning = (
            <button className="bg-success-600 border border-success-700  ml-5 h-9 p-2 rounded-sm text-base-100 uppercase w-24">
                <ClipLoader color="white" loading={this.state.scanning} size={20} />
            </button>
        );
        const scanButton = (
            <button
                className="bg-success-600 border border-success-700 ml-5 p-2 h-9 hover:bg-success-800 rounded-sm text-base-100 uppercase w-24"
                onClick={this.onTriggerScan}
            >
                Scan now
            </button>
        );
        return this.state.scanning ? buttonScanning : scanButton;
    };

    renderTable = () => {
        const columns = [
            { accessor: 'definition.name', Header: 'Name' },
            { accessor: 'definition.description', Header: 'Description' },
            {
                accessor: 'aggregatedResults.PASS',
                Header: 'Pass',
                Cell: ({ original }) => original.aggregatedResults.PASS || 0,
                sortMethod: sortNumber
            },
            {
                accessor: 'aggregatedResults.INFO',
                Header: 'Info',
                Cell: ({ original }) => original.aggregatedResults.INFO || 0,
                sortMethod: sortNumber
            },
            {
                accessor: 'aggregatedResults.WARN',
                Header: 'Warn',
                Cell: ({ original }) => original.aggregatedResults.WARN || 0,
                sortMethod: sortNumber
            },
            {
                accessor: 'aggregatedResults.NOTE',
                Header: 'Note',
                Cell: ({ original }) => original.aggregatedResults.NOTE || 0,
                sortMethod: sortNumber
            }
        ];

        const { benchmarkScanResults, selectedBenchmarkScanResult } = this.props;
        const rows = benchmarkScanResults;
        const name =
            selectedBenchmarkScanResult.definition && selectedBenchmarkScanResult.definition.name;
        if (!rows.length)
            return (
                <NoResultsMessage message="No benchmark results available. Please scan your cluster first." />
            );
        return (
            <Table
                rows={rows}
                columns={columns}
                idAttribute="definition.name"
                selectedRowId={name}
                onRowClick={this.onRowClick}
                noDataText="No benchmark results available. Please scan your cluster first."
            />
        );
    };

    renderModal() {
        if (!this.props.selectedBenchmarkHostResult) return '';
        return (
            <HostResultModal
                benchmarkHostResult={this.props.selectedBenchmarkHostResult}
                onClose={this.onBenchmarkHostResultModalClose}
            />
        );
    }

    renderBenchmarksSidePanel() {
        if (this.props.isFetchingBenchmarkCheckHostResults)
            return (
                <div className="w-1/2 bg-base-100">
                    <Loader />
                </div>
            );
        if (
            !this.props.selectedBenchmarkScanResult.definition ||
            !this.props.selectedBenchmarkScanResult.hostResults
        )
            return '';
        return (
            <BenchmarksSidePanel
                header={this.props.selectedBenchmarkScanResult.definition.name}
                hostResults={this.props.selectedBenchmarkScanResult.hostResults}
                onClose={this.onCloseSidePanel}
                onRowClick={this.onHostResultClick}
            />
        );
    }

    render() {
        return (
            <div className="flex flex-col h-full">
                <div className="flex w-full my-3 px-3 items-center z-1">
                    <span className="flex flex-1 font-600 self-center text-base-600 text-lg">
                        Last Scanned: {this.props.lastScannedTime || 'Never'}
                    </span>
                    <div className="flex self-center justify-end pr-5 border-r-2 border-base-400">
                        <span className="mr-4">{this.renderScanOptions()}</span>
                        <span>{this.renderScanTimes()}</span>
                    </div>
                    {this.renderScanButton()}
                </div>
                <div className="flex flex-1 border-t border-primary-300 bg-base-200">
                    <div className="w-full overflow-auto bg-base-100 rounded-sm shadow">
                        {this.renderTable()}
                    </div>
                    {this.renderBenchmarksSidePanel()}
                    {this.renderModal()}
                </div>
            </div>
        );
    }
}

const getBenchmarkScanResults = createSelector([selectors.getLastScan], lastScan => {
    if (!lastScan || !lastScan.metadata) return [];
    const { checks } = lastScan.data;
    return checks;
});

const getLastScannedTime = createSelector([selectors.getLastScan], lastScan => {
    if (!lastScan || !lastScan.metadata) return '';
    const scanTime = dateFns.format(lastScan.metadata.time, dateTimeFormat);
    return scanTime || '';
});

const getLastScanId = createSelector([selectors.getLastScan], lastScan => {
    if (!lastScan || !lastScan.data) return '';
    const { id } = lastScan.data;
    return id || '';
});

const getSelectedBenchmarkScanResult = createSelector(
    [selectors.getSelectedBenchmarkScanResult, selectors.getBenchmarkCheckHostResults],
    (selectedScan, selectedScanHostsResults) => {
        const result = Object.assign({}, selectedScan, selectedScanHostsResults);
        return result;
    }
);

const getClusterId = (state, props) => props.match.params.clusterId;

const mapStateToProps = createStructuredSelector({
    benchmarkScanResults: getBenchmarkScanResults,
    lastScannedTime: getLastScannedTime,
    lastScannedId: getLastScanId,
    schedule: selectors.getBenchmarkSchedule,
    selectedBenchmarkScanResult: getSelectedBenchmarkScanResult,
    selectedBenchmarkHostResult: selectors.getSelectedBenchmarkHostResult,
    clusterId: getClusterId,
    isFetchingBenchmarkCheckHostResults: state =>
        selectors.getLoadingStatus(state, types.FETCH_BENCHMARK_CHECK_HOST_RESULTS)
});

const mapDispatchToProps = dispatch => ({
    startPollBenchmarkScanResults: benchmark =>
        dispatch(benchmarkActions.pollBenchmarkScanResults.start(benchmark)),
    stopPollBenchmarkScanResults: () => dispatch(benchmarkActions.pollBenchmarkScanResults.stop()),
    selectBenchmarkScheduleDay: (benchmarkId, benchmarkName, value, clusterId) =>
        dispatch(
            benchmarkActions.selectBenchmarkScheduleDay(
                benchmarkId,
                benchmarkName,
                value,
                clusterId
            )
        ),
    selectBenchmarkScheduleHour: (benchmarkId, benchmarkName, value, clusterId) =>
        dispatch(
            benchmarkActions.selectBenchmarkScheduleHour(
                benchmarkId,
                benchmarkName,
                value,
                clusterId
            )
        ),
    fetchBenchmarkSchedule: benchmark =>
        dispatch(benchmarkActions.fetchBenchmarkSchedule.request(benchmark)),
    triggerBenchmarkScan: benchmark =>
        dispatch(benchmarkActions.triggerBenchmarkScan.request(benchmark)),
    selectBenchmarkScanResult: benchmarkScanResult =>
        dispatch(benchmarkActions.selectBenchmarkScanResult(benchmarkScanResult)),
    selectBenchmarkHostResult: benchmarkHostResult =>
        dispatch(benchmarkActions.selectBenchmarkHostResult(benchmarkHostResult)),
    fetchBenchmarkCheckHostResults: benchmark => {
        dispatch(benchmarkActions.fetchBenchmarkCheckHostResults.request(benchmark));
    }
});

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(BenchmarksPage));
