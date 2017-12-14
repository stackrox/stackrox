import React, { Component } from 'react';
import Table from 'Components/Table';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';

import ComplianceBenchmarksSidePanel from 'Containers/Violations/Compliance/ComplianceBenchmarksSidePanel';

import axios from 'axios';
import emitter from 'emitter';
import queryString from 'query-string';
import dateFns from 'date-fns';
import { setTimeout } from 'timers';
import { ClipLoader } from 'react-spinners';

class CompliancePage extends Component {
    constructor(props) {
        super(props);

        this.params = {};

        this.timeout = null;

        this.state = {
            tab: {
                headers: [{ text: 'CIS Docker Benchmark', disabled: false }, { text: 'Swarm Benchmark', disabled: true }, { text: 'Kubernetes Benchmark', disabled: true }]
            },
            table: {
                columns: [
                    { key: 'definition.name', label: 'Name' },
                    { key: 'definition.description', label: 'Description' },
                    { key: 'aggregatedResults.PASS', label: 'Pass', default: 0, align: 'right' },
                    { key: 'aggregatedResults.INFO', label: 'Info', default: 0, align: 'right' },
                    { key: 'aggregatedResults.WARN', label: 'Warn', default: 0, align: 'right' },
                    { key: 'aggregatedResults.NOTE', label: 'Note', default: 0, align: 'right' }
                ],
                rows: []
            },
            lastScanned: "",
            scanId: "",
            scanning: false
        }

        this.pollBenchMarks = this.pollBenchMarks.bind(this);
    }

    componentDidMount() {
        this.pollBenchMarks();
    }

    pollBenchMarks() {
        var promise = this.getBenchMarks();
        var func = this.pollBenchMarks;
        // eslint-disable-next-line
        var timeout = this.timeout;
        promise.then(function (result) {
            timeout = setTimeout(func, 5000);
        });
    }

    getBenchMarks() {
        var promise = new Promise((resolve, reject) => {
            var params = "?" + queryString.stringify(this.params);
            const table = this.state.table;
            axios.get(`/v1/benchmarks/results/grouped/cis${params}`).then((response) => {
                if (!response.data || !response.data.benchmarks) return;
                var lastScanned = dateFns.format(response.data.benchmarks[0].time, 'MM/DD/YYYY h:MM:ss A');
                var scanId = response.data.benchmarks[0].scanId;
                var table = this.state.table;
                table.rows = response.data.benchmarks[0].checkResults;
                if(lastScanned !== this.state.lastScanned) {
                    this.setState({ table: table, lastScanned: lastScanned, scanId: scanId, scanning: false });
                } 
                resolve({});
            }).catch((error) => {
                table.rows = [];
                this.setState({ table: table, lastScanned: "", scanId: "" });
                resolve({});
            });
        });
        return promise;
    }

    onTriggerScan() {
        this.setState({scanning: true});
        axios.post('/v1/benchmarks/trigger', {}).then((response) => {
        }).catch((error) => {
            console.log(error);
        });
    }

    onRowClick(row) {
        emitter.emit('ComplianceTable:row-selected', row);
    }

    render() {
        return (
            <div className="flex">
                <Tabs className="bg-white" headers={this.state.tab.headers}>
                    <TabContent name={this.state.tab.headers[0]}>
                        <div className="flex flex-1 border-t border-primary-300 bg-base-100">
                            <div className="w-full p-3 overflow-y-scroll bg-white rounded-sm shadow">
                                <div className="flex w-full py-2 pl-2 items-center">
                                    <h1 className="flex flex-1 text-lg text-primary-500 justify-end">Last Scanned: {this.state.lastScanned || 'Never'}</h1>
                                    {
                                        (this.state.scanning) ? (<div className="px-4"><ClipLoader color={'#123abc'} loading={this.state.scanning} size={20}></ClipLoader></div>) : (<button className="border border-base-300 border-primary-500 px-2 py-1 ml-2 font-semibold text-primary-500 hover:text-white hover:bg-primary-500" onClick={this.onTriggerScan.bind(this)}>Trigger Scan</button>)
                                    }
                                </div>
                                <Table columns={this.state.table.columns} rows={this.state.table.rows} onRowClick={this.onRowClick.bind(this)}></Table>
                            </div>
                            <ComplianceBenchmarksSidePanel></ComplianceBenchmarksSidePanel>
                        </div>
                    </TabContent>
                    <TabContent name={this.state.tab.headers[1]}></TabContent>
                    <TabContent name={this.state.tab.headers[2]}></TabContent>
                </Tabs>
            </div>
        );
    }

    componentWillUnmount() {
        if(this.timeout) this.timeout.cancel();
    }

}

export default CompliancePage;
