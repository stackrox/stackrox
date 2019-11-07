import React from 'react';
import PropTypes from 'prop-types';
import { format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

import Widget from 'Components/Widget';
import NoResultsMessage from 'Components/NoResultsMessage';

const processData = data => {
    if (!data.violations || !data.violations.length) return null;
    return data.violations[0];
};

const ViolationFindings = ({ data, message }) => {
    const policyViolation = processData(data);
    let content = null;
    if (policyViolation) {
        content = (
            <>
                <Widget
                    header="Violation"
                    className="m-4 flex-1"
                    bodyClassName="flex flex-col p-4 leading-normal"
                >
                    <ul className="list-reset leading-loose">
                        {policyViolation.violations.map(violation => {
                            return (
                                <li className="border-b border-base-300">{violation.message}</li>
                            );
                        })}
                    </ul>
                </Widget>
                <div>
                    <Widget
                        header="Time of Violation"
                        className="m-4"
                        bodyClassName="flex flex-col p-4 leading-normal"
                    >
                        {format(policyViolation.time, dateTimeFormat)}
                    </Widget>
                    <Widget
                        header="Enforcement"
                        className="m-4"
                        bodyClassName="flex flex-col p-4 leading-normal"
                    >
                        {policyViolation.policy.enforcementActions.join(', ') || 'No Enforcement'}
                    </Widget>
                    <Widget
                        header="Category"
                        className="m-4"
                        bodyClassName="flex flex-col p-4 leading-normal"
                    >
                        {policyViolation.policy.categories.join(', ')}
                    </Widget>
                </div>
            </>
        );
    } else {
        content = (
            <NoResultsMessage message={message} className="p-6 shadow mb-4 mx-4" icon="info" />
        );
    }
    return <div className="flex w-full bg-transparent">{content}</div>;
};

ViolationFindings.propTypes = {
    data: PropTypes.shape({}).isRequired,
    message: PropTypes.string.isRequired
};

export default ViolationFindings;
