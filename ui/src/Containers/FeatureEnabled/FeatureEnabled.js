import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';

import { isBackendFeatureFlagEnabled } from 'utils/featureFlags';

const FeatureEnabled = ({ featureFlags, featureFlag, children }) => {
    const featureFlagEnabled = isBackendFeatureFlagEnabled(featureFlags, featureFlag, false);

    if (!featureFlagEnabled) return null;

    return children;
};

const mapStateToProps = createStructuredSelector({
    featureFlags: selectors.getFeatureFlags
});

export default connect(
    mapStateToProps,
    null
)(FeatureEnabled);
