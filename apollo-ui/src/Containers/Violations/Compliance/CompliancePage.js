import React, { Component } from 'react';
import Table from 'Components/Table';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';

import axios from 'axios';
import queryString from 'query-string';

class CompliancePage extends Component {
    constructor(props) {
        super(props);

        this.params = {};

        this.state = {
            tab: {
                headers: [{ text: 'CIS Docker Benchmark', disabled: false }, { text: 'Swarm Benchmark', disabled: true }, { text: 'Kubernetes Benchmark', disabled: true }]
            },
            table: {
                columns: [
                    { key: 'benchmarkDefinition.name', label: 'Name' },
                    { key: 'benchmarkDefinition.description', label: 'Description' },
                    { key: 'testResult.result', label: 'Result' }
                ],
                rows: []
            }
        }
    }

    componentDidMount() {
        this.getBenchMarks();
    }

    getBenchMarks() {
        var params = "?" + queryString.stringify(this.params);
        const table = this.state.table;
        axios.get(`/v1/benchmarks/results${params}`).then((response) => {
            if (!response.data || !response.data.benchmarks) return;
            var table = this.state.table;
            table.rows = response.data.benchmarks[0].results;
            this.setState({ table: table });
        }).catch((error) => {
            table.rows = [];
            this.setState({ table: table });
        });
    }

    render() {
        return (
            <div className="flex">
                <Tabs className="bg-white" headers={this.state.tab.headers}>
                    <TabContent name={this.state.tab.headers[0]}>
                    <div className="flex flex-1 bg-base-100">
                        <div className="flex-1 w-full p-3 overflow-y-scroll bg-white rounded-sm shadow">
                            <Table columns={this.state.table.columns} rows={this.state.table.rows} onRowClick={(row) => { }}></Table>
                        </div>
                    </div>
                    </TabContent>
                    <TabContent name={this.state.tab.headers[1]}></TabContent>
                    <TabContent name={this.state.tab.headers[2]}></TabContent>
                </Tabs>
            </div>
        );
    }

}

export default CompliancePage;
