import React from 'react';
import PropTypes from 'prop-types';

import NetworkFlowCard from './NetworkFlowCard';
import ProcessCard from './ProcessCard';
import K8sCard from './K8sCard';

function RuntimeMessages({ processViolation, violations }) {
    const isPlainViolation = !!violations?.length;
    const isProcessViolation = !!processViolation?.processes?.length;
    const isNetworkFlowViolation = violations?.some((violation) => !!violation.networkFlowInfo);

    const plainViolations = isNetworkFlowViolation
        ? violations?.map(({ message: flowMessage, networkFlowInfo, time }) => (
              <NetworkFlowCard
                  key={`${time}-${flowMessage}`}
                  message={flowMessage}
                  networkFlowInfo={networkFlowInfo}
                  time={time}
              />
          ))
        : violations?.map(({ message: eventMessage, keyValueAttrs, time }) => (
              <K8sCard
                  key={`${time}-${eventMessage}`}
                  message={eventMessage}
                  keyValueAttrs={keyValueAttrs}
                  time={time}
              />
          ));
    return (
        <>
            {isPlainViolation && plainViolations}
            {isProcessViolation && (
                <ProcessCard
                    processes={processViolation.processes}
                    message={processViolation.message}
                />
            )}
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
