import React, { Component } from 'react';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import Table from 'Components/Table';
import Select from 'Components/Select';
import Pills from 'Components/Pills';

class ViolationsContainer extends Component {
    constructor(props) {
        super(props);

        this.state = {
            tab: {
                headers: ['Policies', 'Compliance']
            },
            select: {
                options: ['Last 24 Hours', 'Last Week', 'Last Month', 'Last Year']
            },
            pills: ['All', 'Image Assurance', 'Configurations', 'Orchestrator Target', 'Denial of Policy', 'Privileges & Capabilities', 'Account Authorization'],
            table: {
                columns: [
                    { key: 'benchmark', label: 'Benchmark' },
                    { key: 'status', label: 'Status' }
                ],
                rows: [
                    { id: 1, benchmark: 'Ensure a separate partition for containers has been created', status: 'Pass' },
                    { id: 2, benchmark: 'Ensure the container host has been Hardened', status: 'Pass' },
                    { id: 3, benchmark: 'Ensure Docker is up to Date. Using 17.06.0 which is current', status: 'Pass' },
                    { id: 4, benchmark: 'Ensure only trusted users are allowed to control Docker daemon', status: 'Fail' }
                ]
            }
        }
    }

    render() {
        return (
            <section className="flex flex-1 p-4">
                <Tabs headers={this.state.tab.headers}>
                    <TabContent name={this.state.tab.headers[0]}>
                        <div className="flex flex-1 flex-row p-4">
                            <div className="flex flex-1 self-center justify-start">
                                <input className="appearance-none border rounded w-full py-2 px-3 border-gray-light"
                                    placeholder="Scope by resource type:Registry" />
                            </div>
                            <div className="flex flex-row self-center justify-end">
                                <Select options={this.state.select.options}></Select>
                            </div>
                        </div>
                        <div className="flex flex-1 flex-col p-4">
                            <Pills data={this.state.pills}></Pills>
                        </div>
                    </TabContent>
                    <TabContent name={this.state.tab.headers[1]}>
                        <div className="flex flex-1 flex-row p-4">
                            <Table columns={this.state.table.columns} rows={this.state.table.rows}></Table>
                        </div>
                    </TabContent>
                </Tabs>
            </section>
        );
    }

}

export default ViolationsContainer;
