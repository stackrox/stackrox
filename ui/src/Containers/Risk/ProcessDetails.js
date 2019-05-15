import React, { Component } from 'react';
import PropTypes from 'prop-types';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import shave from 'shave';

import ProcessDiscoveryCard from 'Containers/Risk/ProcessDiscoveryCard';
import ProcessSpecificationWhitelists from 'Containers/Risk/ProcessSpecificationWhitelists';
import ProcessBinaryCollapsible from 'Containers/Risk/ProcessBinaryCollapsible';
import Table, {
    defaultHeaderClassName,
    defaultColumnClassName,
    wrapClassName
} from 'Components/Table';
import NoResultsMessage from 'Components/NoResultsMessage';
import orderBy from 'lodash/orderBy';

const MAX_STRING_HEIGHT = 70;

class ProcessDetails extends Component {
    static propTypes = {
        deploymentId: PropTypes.string.isRequired,
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

    componentDidMount() {
        /*
         * @TODO: Investigate a better alternative to this approach. The "shave.js" function call
         * was hoisted up from the ProcessBinaryCollapsible Component because each "componentDidMount"
         * was causing a forced reflow that was causing some slow performance. This isn't a very
         * React friendly way of doing things since this component knows more than it should about
         * it's child components, but we can see some performance improvements. In addition, windowing
         * or using something like react-virtualized might be helpful when it comes to rendering
         * a large number of iterated Components
         */
        shave('.binary-args', MAX_STRING_HEIGHT);
    }

    renderProcessSignals = signals => {
        const columns = [
            {
                Header: 'Time',
                id: 'time',
                accessor: d => dateFns.format(d.signal.time, dateTimeFormat),
                headerClassName: `${defaultHeaderClassName} w-1/4 pointer-events-none`,
                className: `${wrapClassName} ${defaultColumnClassName} w-1/4 cursor-auto`
            },
            {
                Header: 'Pod ID',
                accessor: 'podId',
                headerClassName: `${defaultHeaderClassName} w-1/3 pointer-events-none`,
                className: `${wrapClassName} ${defaultColumnClassName} w-1/3 cursor-auto`
            },
            {
                Header: 'UID',
                id: 'uid',
                accessor: d => d.signal.uid,
                headerClassName: `${defaultHeaderClassName} w-1/6 pointer-events-none`,
                className: `${wrapClassName} ${defaultColumnClassName} w-1/6 cursor-auto`
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
                    noDataText="No results found. Please refine your search."
                    page={this.state.page}
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

    renderProcessDiscoveryCard = process => (
        <ProcessDiscoveryCard process={process} deploymentId={this.props.deploymentId}>
            {this.renderProcessBinaries(process.groups)}
        </ProcessDiscoveryCard>
    );

    renderProcessDiscoveryCards = () => {
        const { groups: processGroups } = this.props.processGroup;
        const sortedProcessGroups = orderBy(processGroups, ['suspicious', 'name'], ['desc', 'asc']);
        return sortedProcessGroups.map((processGroup, i, list) => (
            <div
                className={`px-3 pt-5 ${i === list.length - 1 ? 'pb-5' : ''}`}
                key={processGroup.name}
            >
                {this.renderProcessDiscoveryCard(processGroup)}
            </div>
        ));
    };

    render() {
        return (
            <div>
                <h3 className="border-b pb-2 mx-3 mt-5">Running Processes</h3>
                {this.renderProcessDiscoveryCards()}
                <ProcessSpecificationWhitelists />
            </div>
        );
    }
}

export default ProcessDetails;
