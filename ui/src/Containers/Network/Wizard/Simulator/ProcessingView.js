import React from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import PropTypes from 'prop-types';

import LoadingSection from './Tiles/LoadingSection';

function ProcessingView(props) {
    const { modificationState, policyGraphState } = props;
    if (modificationState !== 'REQUEST' && policyGraphState !== 'REQUEST') return null;

    return <div className="flex flex-col flex-1">{LoadingSection()}</div>;
}

ProcessingView.propTypes = {
    modificationState: PropTypes.string.isRequired,
    policyGraphState: PropTypes.string.isRequired
};

const mapStateToProps = createStructuredSelector({
    modificationState: selectors.getNetworkPolicyModificationState,
    policyGraphState: selectors.getNetworkPolicyGraphState
});

export default connect(mapStateToProps)(ProcessingView);
