import React from 'react';
import PropTypes from 'prop-types';
import { format } from 'date-fns';

import KeyValue from 'Components/KeyValue';
import RuntimeViolationCollapsibleCard from 'Containers/Violations/RuntimeViolationCollapsibleCard';
import dateTimeFormat from 'constants/dateTimeFormat';

function K8sCard({ message, keyValueAttrs, time }) {
    const keyValueArr = keyValueAttrs?.attrs || [];

    return (
        <div className="mb-4" key={message} data-testid="runtime-processes">
            <RuntimeViolationCollapsibleCard title={message}>
                <div className="border-t border-base-300" label={message}>
                    <div className="flex px-4 py-2 border-base-300 border-b text-base-600">
                        <KeyValue label="Time:" value={format(time, dateTimeFormat)} />
                    </div>
                    {keyValueArr.map(({ key, value }) => (
                        <div className="flex flex-1 text-base-600 px-4 py-2" key={key}>
                            <KeyValue label={`${key}:`} value={value} />
                        </div>
                    ))}
                </div>
            </RuntimeViolationCollapsibleCard>
        </div>
    );
}

K8sCard.propTypes = {
    message: PropTypes.string.isRequired,
    keyValueAttrs: PropTypes.shape({
        attrs: PropTypes.arrayOf(
            PropTypes.shape({
                key: PropTypes.string.isRequired,
                value: PropTypes.string.isRequired,
            })
        ),
    }),
    time: PropTypes.string.isRequired,
};

K8sCard.defaultProps = {
    keyValueAttrs: {},
};

export default K8sCard;
