import React from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Mutation } from '@apollo/client/react/components';
import { useQuery } from '@apollo/client';
import * as Icon from 'react-feather';

import { actions as notificationActions } from 'reducers/notifications';
import { TRIGGER_SCAN, RUN_STATUSES } from 'queries/standard';
import Button from 'Components/Button';

const getTriggerRunIds = (data) => {
    if (data && data.complianceTriggerRuns.length) {
        return data.complianceTriggerRuns.map((run) => run.id);
    }
    return [];
};

const areRunsFinished = (data) => {
    let runsFinished = true;
    if (data && data.complianceRunStatuses && data.complianceRunStatuses.runs) {
        const incompleteRuns = data.complianceRunStatuses.runs.filter(
            (x) => x.state !== 'FINISHED'
        );
        runsFinished = incompleteRuns.length === 0;
    }
    return runsFinished;
};

class ScanButton extends React.Component {
    static propTypes = {
        className: PropTypes.string,
        text: PropTypes.string.isRequired,
        textCondensed: PropTypes.string,
        textClass: PropTypes.string,
        clusterId: PropTypes.string,
        standardId: PropTypes.string,
        loaderSize: PropTypes.number,
        addToast: PropTypes.func.isRequired,
        removeToast: PropTypes.func.isRequired,
    };

    static defaultProps = {
        className: 'btn btn-base h-10',
        clusterId: '*',
        textClass: null,
        textCondensed: null,
        standardId: '*',
        loaderSize: 20,
    };

    constructor(props) {
        super(props);

        this.state = { pendingRunIds: [] };
    }

    onClick = (triggerScan) => () => {
        const { clusterId, standardId } = this.props;
        triggerScan({ variables: { clusterId, standardId } })
            .then(this.mutationCompleted)
            .catch((e) => {
                this.props.addToast(e.message);
                setTimeout(this.props.removeToast, 2000);
            });
    };

    mutationCompleted = ({ data }) => {
        this.setState({ pendingRunIds: getTriggerRunIds(data) });
    };

    queryCompleted = (client) => (data) => {
        if (this.state.pendingRunIds.length && areRunsFinished(data)) {
            this.setState({ pendingRunIds: [] });
            client.resetStore();
        }
    };

    render() {
        const { className, text, textCondensed, textClass, loaderSize } = this.props;
        return (
            <Mutation mutation={TRIGGER_SCAN}>
                {(triggerScan, { client }) => {
                    const variables = { latest: true};
                    const { startPolling, stopPolling, error } = useQuery(RUN_STATUSES, {
                        variables,
                        errorPolicy: 'all',
                        notifyOnNetworkStatusChange: true,
                        onCompleted: this.queryCompleted(client),
                    });
                    if (error) {
                        return (
                            <Button
                                dataTestId="scan-button"
                                className={className}
                                text={text}
                                textCondensed={textCondensed}
                                textClass={textClass}
                                icon={
                                    <Icon.RefreshCcw
                                        size="14"
                                        className="bg-base-100 mx-1 lg:ml-1 lg:mr-3"
                                    />
                                }
                                onClick={null}
                                isLoading={null}
                                disabled
                                loaderSize={loaderSize}
                            />
                        );
                    }

                    const polling = !!this.state.pendingRunIds.length;
                    if (polling) {
                        startPolling(5000);
                    } else {
                        stopPolling();
                    }
                    return (
                        <Button
                            dataTestId="scan-button"
                            className={className}
                            text={text}
                            textCondensed={textCondensed}
                            textClass={textClass}
                            icon={
                                <Icon.RefreshCcw
                                    size="14"
                                    className="bg-base-100 mx-1 lg:ml-1 lg:mr-3"
                                />
                            }
                            onClick={this.onClick(triggerScan)}
                            isLoading={polling}
                            disabled={polling}
                            loaderSize={loaderSize}
                        />
                    );
                }}
            </Mutation>
        );
    }
}

const mapDispatchToProps = {
    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification,
};

export default connect(null, mapDispatchToProps)(ScanButton);
