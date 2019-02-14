import React from 'react';
import PropTypes from 'prop-types';
import { Mutation } from 'react-apollo';
import { TRIGGER_SCAN, RUN_STATUSES } from 'queries/standard';

import Button from 'Components/Button';
import * as Icon from 'react-feather';
import Query from 'Components/ThrowingQuery';

const getTriggerRunIds = data => {
    if (data && data.complianceTriggerRuns.length) {
        return data.complianceTriggerRuns.map(run => run.id);
    }
    return [];
};

const areRunsFinished = data => {
    let runsFinished = true;
    if (data && data.complianceRunStatuses && data.complianceRunStatuses.runs.length) {
        runsFinished = !data.complianceRunStatuses.runs
            .map(run => run.state)
            .includes('WAIT_FOR_DATA');
    }
    return runsFinished;
};

class ScanButton extends React.Component {
    state = { pendingRunIds: [] };

    onClick = triggerScan => () => {
        const { clusterId, standardId } = this.props;
        triggerScan({ variables: { clusterId, standardId } }).then(this.mutationCompleted);
    };

    mutationCompleted = ({ data }) => {
        this.setState({ pendingRunIds: getTriggerRunIds(data) });
    };

    queryCompleted = client => data => {
        if (this.state.pendingRunIds.length && areRunsFinished(data)) {
            this.setState({ pendingRunIds: [] });
            client.resetStore();
        }
    };

    render() {
        const { className, text, textCondensed, textClass } = this.props;
        return (
            <Mutation mutation={TRIGGER_SCAN}>
                {(triggerScan, { client }) => {
                    const variables = { ids: this.state.pendingRunIds };
                    return (
                        <Query
                            query={RUN_STATUSES}
                            variables={variables}
                            onCompleted={this.queryCompleted(client)}
                        >
                            {({ startPolling, stopPolling }) => {
                                const polling = !!this.state.pendingRunIds.length;
                                if (polling) {
                                    startPolling(5000);
                                } else {
                                    stopPolling();
                                }
                                return (
                                    <Button
                                        className={className}
                                        text={text}
                                        textCondensed={textCondensed}
                                        textClass={textClass}
                                        icon={
                                            <Icon.RefreshCcw
                                                size="14"
                                                className="mx-1 lg:ml-1 lg:mr-3"
                                            />
                                        }
                                        onClick={this.onClick(triggerScan)}
                                        isLoading={polling}
                                    />
                                );
                            }}
                        </Query>
                    );
                }}
            </Mutation>
        );
    }
}

ScanButton.propTypes = {
    className: PropTypes.string,
    text: PropTypes.string.isRequired,
    textCondensed: PropTypes.string,
    textClass: PropTypes.string,
    clusterId: PropTypes.string,
    standardId: PropTypes.string
};

ScanButton.defaultProps = {
    className: 'btn btn-base h-10',
    clusterId: '*',
    textClass: null,
    textCondensed: null,
    standardId: '*'
};

export default ScanButton;
