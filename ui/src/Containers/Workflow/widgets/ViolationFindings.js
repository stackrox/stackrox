import React from 'react';
import PropTypes from 'prop-types';
import { format } from 'date-fns';

import dateTimeFormat from 'constants/dateTimeFormat';
import { knownBackendFlags } from 'utils/featureFlags';
import ANALYST_NOTES_TYPES from 'constants/analystnotes';
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
            <div className="mx-4 grid-dense grid-auto-fit grid grid-gap-4 xl:grid-gap-6 mb-4 xxxl:grid-gap-8 grid-columns-1 md:grid-columns-2 lg:grid-columns-3 w-full">
                <Widget
                    header="Time of Violation"
                    className="s-1"
                    bodyClassName="flex flex-col p-4 leading-normal"
                >
                    {format(policyViolation.time, dateTimeFormat)}
                </Widget>
                <Widget
                    header="Enforcement"
                    className="s-1"
                    bodyClassName="flex flex-col p-4 leading-normal"
                >
                    {policyViolation.policy.enforcementActions.join(', ') || 'No Enforcement'}
                </Widget>
                <Widget
                    header="Category"
                    className="s-full lg:s-1"
                    bodyClassName="flex flex-col p-4 leading-normal"
                >
                    {policyViolation.policy.categories.join(', ')}
                </Widget>
                <Widget
                    header="Violation"
                    className="s-full flex-1"
                    bodyClassName="flex flex-col p-4 leading-normal"
                >
                    <ul className="leading-loose">
                        {policyViolation.violations.map(violation => {
                            return (
                                <li
                                    className="border-b border-base-300 py-2"
                                    key={violation.message}
                                >
                                    {violation.message}
                                </li>
                            );
                        })}
                    </ul>
                </Widget>
                <FeatureEnabled featureFlag={knownBackendFlags.ROX_ANALYST_NOTES_UI}>
                    <div className="sx-2 sy-1 bg-base-100 rounded shadow">
                        <AnalystTags
                            type={ANALYST_NOTES_TYPES.VIOLATION}
                            id={policyViolation.id}
                            className="h-full"
                        />
                    </div>
                    <div className="sx-1 sy-2 bg-base-100 rounded shadow">
                        <AnalystComments
                            type={ANALYST_NOTES_TYPES.VIOLATION}
                            id={policyViolation.id}
                            className="h-full"
                        />
                    </div>
                </FeatureEnabled>
            </div>
        );
    } else {
        content = (
            <NoResultsMessage message={message} className="p-3 shadow mb-4 mx-4" icon="info" />
        );
    }
    return <div className="flex w-full bg-transparent">{content}</div>;
};

ViolationFindings.propTypes = {
    data: PropTypes.shape({}).isRequired,
    message: PropTypes.string.isRequired
};

export default ViolationFindings;
