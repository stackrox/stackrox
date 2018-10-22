import React, { Component } from 'react';
import PropTypes from 'prop-types';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

import ProcessDiscoveryCard from 'Containers/Risk/ProcessDiscoveryCard';
import ProcessBinaryCollapsible from 'Containers/Risk/ProcessBinaryCollapsible';
import Table, {
    defaultHeaderClassName,
    defaultColumnClassName,
    wrapClassName
} from 'Components/Table';
import NoResultsMessage from 'Components/NoResultsMessage';

class ProcessDetails extends Component {
    static propTypes = {
        processGroup: PropTypes.shape({
            groups: PropTypes.arrayOf(PropTypes.object)
        }).isRequired
    };

    constructor(props) {
        super(props);
        this.state = {
            page: 0
        };
    }

    renderProcessSignals = signals => {
        const columns = [
            {
                Header: 'Time',
                id: 'time',
                accessor: d => dateFns.format(d.signal.time, dateTimeFormat),
                headerClassName: `${defaultHeaderClassName} w-1/4 pointer-events-none`,
                className: `${wrapClassName} ${defaultColumnClassName} w-1/4 pointer-events-none`
            },
            {
                Header: 'Pod ID',
                accessor: 'podId',
                headerClassName: `${defaultHeaderClassName} w-1/4 pointer-events-none`,
                className: `${wrapClassName} ${defaultColumnClassName} w-1/4 pointer-events-none`
            },
            {
                Header: 'Name',
                accessor: 'containerName',
                headerClassName: `${defaultHeaderClassName} w-1/4 pointer-events-none`,
                className: `${wrapClassName} ${defaultColumnClassName} w-1/4 pointer-events-none`
            },
            {
                Header: 'Container ID',
                accessor: 'signal.containerId',
                headerClassName: `${defaultHeaderClassName} w-1/4 pointer-events-none`,
                className: `${wrapClassName} ${defaultColumnClassName} w-1/4 pointer-events-none`
            }
        ];
        const rows = signals;
        if (!rows.length)
            return <NoResultsMessage message="No results found. Please refine your search." />;
        return (
            <div className="border-b border-base-300">
                <Table
                    rows={signals}
                    columns={columns}
                    onRowClick={this.updateSelectedDeployment}
                    noDataText="No results found. Please refine your search."
                    page={this.state.page}
                    trClassName="pointer-events-none"
                />
            </div>
        );
    };

    renderProcessBinaries = binaries =>
        binaries.map(({ args, signals }) => (
            <ProcessBinaryCollapsible args={args} key={args}>
                {this.renderProcessSignals(signals)}
            </ProcessBinaryCollapsible>
        ));

    renderProcessDiscoveryCard = ({ name, timesExecuted, groups }) => (
        <ProcessDiscoveryCard name={name} timesExecuted={timesExecuted}>
            {this.renderProcessBinaries(groups)}
        </ProcessDiscoveryCard>
    );

    renderProcessDiscoveryCards = () => {
        const { groups: processGroups } = this.props.processGroup;
        return processGroups.map((processGroup, i, list) => (
            <div
                className={`px-3 pt-5 ${i === list.length - 1 ? 'pb-5' : ''}`}
                key={processGroup.name}
            >
                {this.renderProcessDiscoveryCard(processGroup)}
            </div>
        ));
    };

    render() {
        return <div>{this.renderProcessDiscoveryCards()}</div>;
    }
}

export default ProcessDetails;
