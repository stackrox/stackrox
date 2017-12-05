import React, { Component } from 'react';
import Table from 'Components/Table';

import axios from 'axios';
import queryString from 'query-string';

class CompliancePage extends Component {
    constructor(props) {
        super(props);

        this.params = {};

        this.state = {
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
                <Table columns={this.state.table.columns} rows={this.state.table.rows} onRowClick={(row) => {}}></Table>
        );
    }

}

export default CompliancePage;
