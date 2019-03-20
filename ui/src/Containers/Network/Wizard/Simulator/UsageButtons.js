import React from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import PropTypes from 'prop-types';
import { selectors } from 'reducers';

import Apply from './Buttons/Apply';
import Notify from './Buttons/Notify';

function UsageButtons({ modificationSource }) {
    if (modificationSource === 'ACTIVE') {
        return null;
    }
    return (
        <div className="flex mt-2 items-center justify-around p-3 bg-primary-200 border-t-2 border-base-100">
            <Apply />
            <Notify />
        </div>
    );
}

UsageButtons.propTypes = {
    modificationSource: PropTypes.string.isRequired
};

const mapStateToProps = createStructuredSelector({
    modificationSource: selectors.getNetworkPolicyModificationSource
});

export default connect(mapStateToProps)(UsageButtons);
