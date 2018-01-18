import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Table from 'Components/Table';

import BenchmarksSidePanel from 'Containers/Violations/Compliance/BenchmarksSidePanel';

import axios from 'axios';
import emitter from 'emitter';
import queryString from 'query-string';
import dateFns from 'date-fns';
import { ClipLoader } from 'react-spinners';

const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'UPDATE_BENCHMARKS':
            return {
                benchmarks: nextState.benchmarks,
                lastScanned: nextState.lastScanned,
                scanning: false
            };
        case 'START_SCANNING':
            return { scanning: true };
        case 'STOP_SCANNING':
            return { scanning: false };
        default:
            return prevState;
    }
};

class BenchmarksPage extends Component {
    static propTypes = {
        benchmarksResults: PropTypes.string.isRequired,
        benchmarksTrigger: PropTypes.string.isRequired
    }

    constructor(props) {
        super(props);

        this.params = {};

        this.pollTimeoutId = null;

        this.state = {
            benchmarks: [],
            lastScanned: '',
            scanning: false
        };
    }

    componentDidMount() {
        this.pollBenchmarks();
    }

    componentWillUnmount() {
        if (this.pollTimeoutId) {
            clearTimeout(this.pollTimeoutId);
            this.pollTimeoutId = null;
        }
    }

    onTriggerScan = () => {
        this.update('START_SCANNING');
        const url = `/v1/benchmarks/triggers/${this.props.benchmarksTrigger}`;
        axios.post(url, {}).then(() => {}).catch(() => {
            this.update('STOP_SCANNING');
        });
    }

    onRowClick = (row) => {
        emitter.emit('ComplianceTable:row-selected', row);
    }

    getBenchmarks = () => {
        const params = `?${queryString.stringify(this.params)}`;
        return axios.get(`/v1/benchmarks/results/grouped/${this.props.benchmarksResults}${params}`).then((response) => {
            const { data } = response;
            if (!data || !data.benchmarks || data.benchmarks.length === 0) return;
            const lastScanned = dateFns.format(data.benchmarks[0].time, 'MM/DD/YYYY h:mm:ss A');
            const benchmarks = data.benchmarks[0].checkResults;
            if (lastScanned !== this.state.lastScanned) {
                this.update('UPDATE_BENCHMARKS', {
                    benchmarks,
                    lastScanned
                });
            }
        });
    }

    pollBenchmarks = () => {
        this.getBenchmarks().then(() => {
            this.pollTimeoutId = setTimeout(this.pollBenchmarks, 5000);
        });
    }

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    }

    renderScanButton = () => {
        const buttonScanning = <button className="p-3 ml-5 h-10 w-24 rounded-sm bg-success-500 text-white hover:bg-success-600 uppercase text-center"><ClipLoader color="white" loading={this.state.scanning} size={20} /></button>;
        const scanButton = <button className="p-3 ml-5 h-10 w-24 rounded-sm bg-success-500 text-white hover:bg-success-600 uppercase" onClick={this.onTriggerScan}>Scan now</button>;
        return (
            (this.state.scanning) ? (buttonScanning) : (scanButton)
        );
    }

    renderTable = () => {
        const table = {
            columns: [
                { key: 'definition.name', label: 'Name' },
                { key: 'definition.description', label: 'Description' },
                {
                    key: 'aggregatedResults.PASS', label: 'Pass', default: 0, align: 'right'
                },
                {
                    key: 'aggregatedResults.INFO', label: 'Info', default: 0, align: 'right'
                },
                {
                    key: 'aggregatedResults.WARN', label: 'Warn', default: 0, align: 'right'
                },
                {
                    key: 'aggregatedResults.NOTE', label: 'Note', default: 0, align: 'right'
                }
            ],
            rows: this.state.benchmarks
        };
        return (
            <Table columns={table.columns} rows={table.rows} onRowClick={this.onRowClick} />
        );
    }

    render() {
        return (
            <div className="flex flex-col h-full">
                <div className="flex w-full mb-3 px-3 items-center">
                    <span className="flex flex-1 text-xl font-500 text-primary-500 self-end">Last Scanned: {this.state.lastScanned || 'Never'}</span>
                    {this.renderScanButton()}
                </div>
                <div className="flex flex-1 border-t border-primary-300 bg-base-100">
                    <div className="w-full p-3 overflow-y-scroll bg-white rounded-sm shadow">
                        {this.renderTable()}
                    </div>
                    <BenchmarksSidePanel />
                </div>
            </div>
        );
    }
}

export default BenchmarksPage;
