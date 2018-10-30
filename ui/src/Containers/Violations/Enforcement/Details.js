import React from 'react';
import PropTypes from 'prop-types';

import Explanation from 'Containers/Violations/Enforcement/Explanation';
import Header from 'Containers/Violations/Enforcement/Header';

export function Details({ listAlert }) {
    if (!listAlert) return null;
    return (
        <div className="flex flex-col w-full bg-primary-100 overflow-auto pb-5">
            <div className="px-3 pt-5">
                <div className="bg-base-100 shadow">
                    <Header
                        lifecycleStage={listAlert.lifecycleStage}
                        enforcementCount={listAlert.enforcementCount}
                    />
                    <Explanation listAlert={listAlert} />
                </div>
            </div>
        </div>
    );
}

Details.propTypes = {
    listAlert: PropTypes.shape({
        lifecycleStage: PropTypes.string.isRequired,
        enforcementCount: PropTypes.number
    }).isRequired
};

export default Details;
