import React from 'react';
import PropTypes from 'prop-types';

import ProcessSignal from './Signal';
import ProcessBinaryCollapsible from './BinaryCollapsible';

function Binaries({ processes }) {
    return processes.map(({ comandLineArgs, signals }) => (
        <ProcessBinaryCollapsible comandLineArgs={comandLineArgs} key={comandLineArgs}>
            <ProcessSignal signals={signals} />
        </ProcessBinaryCollapsible>
    ));
}

Binaries.propTypes = {
    processes: PropTypes.arrayOf(
        PropTypes.shape({
            args: PropTypes.string,
            signals: PropTypes.arrayOf(PropTypes.object)
        })
    ).isRequired
};

export default Binaries;
