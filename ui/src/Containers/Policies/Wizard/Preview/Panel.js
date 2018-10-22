import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';

import Violations from './Violations';
import WarningMessage from './WarningMessage';
import Whitelisted from './Whitelisted';

function Panel({ dryRun, policyDisabled }) {
    return (
        <div className="bg-primary-100">
            <div className="border-b border-base-400">{WarningMessage(policyDisabled)}</div>
            <div className="py-4">
                <Violations dryrun={dryRun} />
                <Whitelisted dryrun={dryRun} />
            </div>
        </div>
    );
}

Panel.propTypes = {
    dryRun: PropTypes.shape({}).isRequired,
    policyDisabled: PropTypes.bool.isRequired
};

const isPolicyDisabled = createSelector([selectors.getWizardPolicy], policy => {
    if (policy == null) return true;
    if (policy.disabled) return true;
    return false;
});

const mapStateToProps = createStructuredSelector({
    dryRun: selectors.getWizardDryRun,
    policyDisabled: isPolicyDisabled
});

export default connect(mapStateToProps)(Panel);
