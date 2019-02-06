import React from 'react';
import PropTypes from 'prop-types';
import { Mutation } from 'react-apollo';
import { TRIGGER_SCAN, RUN_STATUSES } from 'queries/standard';

import Button from 'Components/Button';
import * as Icon from 'react-feather';
import Query from 'Components/ThrowingQuery';

const scanOnClickHandler = (triggerScan, clusterId, standardId) => () => {
    triggerScan({ variables: { clusterId, standardId } });
};

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

const ScanButton = ({ text, clusterId, standardId }) => (
    <Mutation mutation={TRIGGER_SCAN}>
        {(triggerScan, { data: triggerData }) => {
            const ids = getTriggerRunIds(triggerData);
            return (
                <Query query={RUN_STATUSES} variables={{ ids }}>
                    {({ data, startPolling, stopPolling }) => {
                        startPolling(5000);
                        const scanningFinished = areRunsFinished(data);
                        if (scanningFinished) stopPolling();
                        const showLoader = !scanningFinished;
                        return (
                            <Button
                                text={text}
                                icon={<Icon.RefreshCcw size="14" className="mr-3" />}
                                onClick={scanOnClickHandler(triggerScan, clusterId, standardId)}
                                isLoading={showLoader}
                            />
                        );
                    }}
                </Query>
            );
        }}
    </Mutation>
);

ScanButton.propTypes = {
    text: PropTypes.string.isRequired,
    clusterId: PropTypes.string,
    standardId: PropTypes.string
};

ScanButton.defaultProps = {
    clusterId: '*',
    standardId: '*'
};

export default ScanButton;
