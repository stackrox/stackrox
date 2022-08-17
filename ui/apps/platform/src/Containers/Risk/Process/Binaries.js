import React from 'react';
import PropTypes from 'prop-types';

import ProcessSignal from './Signal';
import BinaryCollapsible from './BinaryCollapsible';

function Binaries({ processes }) {
    return processes.map(({ args, signals }) => {
        return (
            <BinaryCollapsible commandLineArgs={args} key={args}>
                <ProcessSignal signals={signals} />
            </BinaryCollapsible>
        );
    });
}

Binaries.propTypes = {
    processes: PropTypes.arrayOf(
        PropTypes.shape({
            args: PropTypes.string,
            signals: PropTypes.arrayOf(PropTypes.object),
        })
    ).isRequired,
};

export default Binaries;
