import React from 'react';
import PropTypes from 'prop-types';

import ProcessCard from './ProcessCard';
import K8sCard from './K8sCard';

function RuntimeMessages({ processViolation, violations }) {
    const { processes, message } = processViolation || {};
    return (
        <>
            {violations?.map(({ message: eventMessage, keyValueAttrs, time }, key) => (
                <K8sCard
                    key={key}
                    message={eventMessage}
                    keyValueAttrs={keyValueAttrs}
                    time={time}
                />
            ))}
            {processes?.length && <ProcessCard processes={processes} message={message} />}
        </>
    );
}

RuntimeMessages.propTypes = {
    processViolation: PropTypes.shape({
        message: PropTypes.string.isRequired,
        processes: PropTypes.arrayOf(
            PropTypes.shape({
                id: PropTypes.string.isRequired,
            })
        ),
    }),
    violations: PropTypes.arrayOf(
        PropTypes.shape({
            message: PropTypes.string.isRequired,
            keyValueAttrs: PropTypes.shape({
                attrs: PropTypes.arrayOf(
                    PropTypes.shape({
                        key: PropTypes.string,
                        value: PropTypes.string,
                    })
                ),
            }),
        })
    ),
};

RuntimeMessages.defaultProps = {
    processViolation: {},
    violations: [],
};

export default RuntimeMessages;
