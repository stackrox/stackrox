import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { Message } from '@stackrox/ui-components';

import { selectors } from 'reducers';
import Loader from 'Components/Loader';
import Panel from 'Components/Panel';
import Violations from './Violations';
import ExcludedScopes from './ExcludedScopes';
import PreviewButtons from './PreviewButtons';

const DryRunInProgressMessage = () => (
    <div className="flex items-center justify-center h-full" data-testid="dry-run-loading">
        <div className="flex uppercase">
            <Message icon={<Loader />} extraClasses="loading-message">
                Dry run in progress...
            </Message>
        </div>
    </div>
);

const WarningMessage = ({ policyDisabled }) => {
    let message = '';
    if (policyDisabled) {
        message =
            'This policy is not currently enabled. If enabled, the policy would generate violations for the following deployments on your system.';
    } else {
        message =
            'The policy settings you have selected will generate violations for the following deployments on your system. Please verify that this seems accurate before saving.';
    }
    return (
        <div className="border-b border-base-400">
            <Message type="warn">{message}</Message>
        </div>
    );
};

function PreviewPanel({ header, dryRun, policyDisabled, onClose }) {
    const content = dryRun ? (
        <>
            <WarningMessage policyDisabled={policyDisabled} />
            <div className="py-4">
                <Violations dryrun={dryRun} />
                {dryRun?.excluded?.length > 0 && <ExcludedScopes dryrun={dryRun} />}
            </div>
        </>
    ) : (
        <DryRunInProgressMessage />
    );

    return (
        <Panel
            header={header}
            headerComponents={<PreviewButtons />}
            onClose={onClose}
            id="side-panel"
            className="w-1/2"
        >
            <div className="bg-primary-100 h-full">{content}</div>
        </Panel>
    );
}

PreviewPanel.propTypes = {
    header: PropTypes.string,
    dryRun: PropTypes.shape({}).isRequired,
    policyDisabled: PropTypes.bool.isRequired,
    onClose: PropTypes.func.isRequired,
};

PreviewPanel.defaultProps = {
    header: '',
};

const isPolicyDisabled = createSelector([selectors.getWizardPolicy], (policy) => {
    if (policy == null) {
        return true;
    }
    if (policy.disabled) {
        return true;
    }
    return false;
});

const getDryRun = createSelector([selectors.getWizardDryRun], ({ dryRun }) => dryRun);

const mapStateToProps = createStructuredSelector({
    dryRun: getDryRun,
    policyDisabled: isPolicyDisabled,
});

export default connect(mapStateToProps)(PreviewPanel);
