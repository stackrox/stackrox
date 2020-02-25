import React from 'react';
import PropTypes from 'prop-types';
import { format } from 'date-fns';

import dateTimeFormat from 'constants/dateTimeFormat';
import { knownBackendFlags } from 'utils/featureFlags';
import FeatureEnabled from 'Containers/FeatureEnabled';
import Widget from 'Components/Widget';
import NoResultsMessage from 'Components/NoResultsMessage';
import AnalystComments from 'Containers/AnalystNotes/AnalystComments';
import AnalystTags from 'Containers/AnalystNotes/AnalystTags';

const processData = data => {
    if (!data.violations || !data.violations.length) return null;
    return data.violations[0];
};

const ViolationFindings = ({ data, message }) => {
    const policyViolation = processData(data);
    let content = null;
    if (policyViolation) {
        content = (
            <div className="grid grid-gap-0 grid-columns-3 w-full">
                <Widget
                    header="Time of Violation"
                    className="s-1 m-4"
                    bodyClassName="flex flex-col p-4 leading-normal"
                >
                    {format(policyViolation.time, dateTimeFormat)}
                </Widget>
                <Widget
                    header="Enforcement"
                    className="s-1 m-4"
                    bodyClassName="flex flex-col p-4 leading-normal"
                >
                    {policyViolation.policy.enforcementActions.join(', ') || 'No Enforcement'}
                </Widget>
                <Widget
                    header="Category"
                    className="s-1 m-4"
                    bodyClassName="flex flex-col p-4 leading-normal"
                >
                    {policyViolation.policy.categories.join(', ')}
                </Widget>
                <Widget
                    header="Violation"
                    className="sx-2 m-4 flex-1"
                    bodyClassName="flex flex-col p-4 leading-normal"
                >
                    <ul className="list-reset leading-loose">
                        {policyViolation.violations.map(violation => {
                            return (
                                <li className="border-b border-base-300" key={violation.message}>
                                    {violation.message}
                                </li>
                            );
                        })}
                    </ul>
                </Widget>

                {knownBackendFlags.ROX_ANALYST_NOTES_UI === true && (
                    <div>
                        <div className="s-1 sy-2 bg-base-100 m-4 rounded shadow">
                            <FeatureEnabled featureFlag={knownBackendFlags.ROX_ANALYST_NOTES_UI}>
                                <AnalystComments type="Violation" className="" />
                            </FeatureEnabled>
                        </div>
                        <div className="sx-2 sy-1 bg-base-100 m-4 rounded shadow">
                            <FeatureEnabled featureFlag={knownBackendFlags.ROX_ANALYST_NOTES_UI}>
                                <AnalystTags type="Violation" className="h-full" />
                            </FeatureEnabled>
                        </div>
                    </div>
                )}
            </div>
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
