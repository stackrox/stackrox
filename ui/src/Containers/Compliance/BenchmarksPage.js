import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Table from 'Components/Table';
import Select from 'Components/Select';

import BenchmarksSidePanel from 'Containers/Compliance/BenchmarksSidePanel';

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
        case 'UPDATE_SCHEDULE':
            return { schedule: nextState.schedule };
        default:
            return prevState;
    }
};

class BenchmarksPage extends Component {
    static propTypes = {
        benchmarkName: PropTypes.string.isRequired,
    }

    constructor(props) {
        super(props);

        this.params = {};

        this.pollTimeoutId = null;

        this.state = {
            benchmarks: [],
            lastScanned: '',
            scanning: false,
            schedule: {
                name: this.props.benchmarkName,
                day: '',
                hour: '',
                active: false,
                timezone_offset: new Date().getTimezoneOffset() / 60,
            }
        };
    }

    componentDidMount() {
        this.pollBenchmarks();
        this.retrieveSchedule();
    }

    componentWillUnmount() {
        if (this.pollTimeoutId) {
            clearTimeout(this.pollTimeoutId);
            this.pollTimeoutId = null;
        }
    }

    onTriggerScan = () => {
        this.update('START_SCANNING');
        const url = `/v1/benchmarks/triggers/${this.props.benchmarkName}`;
        axios.post(url, {}).then(() => {}).catch(() => {
            this.update('STOP_SCANNING');
        });
    }

    onRowClick = (row) => {
        emitter.emit('ComplianceTable:row-selected', row);
    }

    onScheduleDayChange = (value) => {
        const { schedule } = this.state;
        if (value === 'None') {
            schedule.day = '';
            schedule.hour = '';
            this.update('UPDATE_SCHEDULE', { schedule });
            this.removeSchedule();
        } else {
            schedule.day = value;
            this.update('UPDATE_SCHEDULE', { schedule });
            this.updateSchedule();
        }
    }

    onScheduleHourChange = (value) => {
        const { schedule } = this.state;
        schedule.hour = value;
        this.update('UPDATE_SCHEDULE', { schedule });
        this.updateSchedule();
    }

    getBenchmarks = () => {
        const params = `?${queryString.stringify(this.params)}`;
        return axios.get(`/v1/benchmarks/results/grouped/${this.props.benchmarkName}${params}`).then((response) => {
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
        }).catch((error) => {
            if (error.response && error.response.status === 404) {
                // ignore 404 since it's ok for benchmark schedule to not exist
                return null;
            }
            return Promise.reject(error);
        }).catch((error) => {
            console.error(error);
        });
    }

    retrieveSchedule() {
        return axios.get(`/v1/benchmarks/schedules/${this.props.benchmarkName}`).then((response) => {
            const schedule = response.data;
            schedule.active = true;
            this.update('UPDATE_SCHEDULE', { schedule });
        }).catch((error) => {
            if (error.response && error.response.status === 404) {
                // ignore 404 since it's ok for benchmark schedule to not exist
                return null;
            }
            return Promise.reject(error);
        }).catch((error) => {
            console.error(error);
        });
    }

    pollBenchmarks = () => {
        this.getBenchmarks().then(() => {
            this.pollTimeoutId = setTimeout(this.pollBenchmarks, 5000);
        });
    }

    removeSchedule() {
        const { schedule } = this.state;
        schedule.active = false;
        this.update('UPDATE_SCHEDULE', { schedule });
        return axios.delete(`/v1/benchmarks/schedules/${this.props.benchmarkName}`);
    }

    updateSchedule() {
        if (this.state.schedule.hour === '' || this.state.schedule.day === '') return;

        if (this.state.schedule.active) {
            axios.put(`/v1/benchmarks/schedules/${this.props.benchmarkName}`, this.state.schedule);
        } else {
            const { schedule } = this.state;
            schedule.active = true;
            this.update('UPDATE_SCHEDULE', { schedule });
            axios.post('/v1/benchmarks/schedules', this.state.schedule);
        }
    }

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    }

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
                { label: 'Sunday', value: 'Sunday' }]
        };
        return (
            <Select className="block w-full border bg-base-100 border-base-200 text-base-500 p-3 pr-8 rounded" value={this.state.schedule.day} placeholder="No scheduled scanning" options={category.options} onChange={this.onScheduleDayChange} />
        );
    }

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
                { label: '11:00 PM', value: '11:00 PM' }]
        };
        return (
            <Select className="block w-full border bg-base-100 border-base-200 text-base-500 p-3 pr-8 rounded" value={this.state.schedule.hour} placeholder="None" options={category.options} onChange={this.onScheduleHourChange} />
        );
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
                    <div className="flex self-center justify-end pr-5 border-r border-primary-200">
                        <span className="mr-4">{this.renderScanOptions()}</span><span>{this.renderScanTimes()}</span>
                    </div>
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
