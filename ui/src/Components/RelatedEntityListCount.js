import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import URLService from 'modules/URLService';
import WorkflowStateMgr from 'modules/WorkflowStateManager';
import { generateURL } from 'modules/URLReadWrite';

// @TODO We should try to use this component for Compliance as well
const RelatedEntityListCount = ({
    match,
    location,
    history,
    name,
    value,
    entityType,
    workflowState,
    ...rest
}) => {
    function onClick() {
        let url;

        // this is a workaround to make this flexible for legacy URLService and new workflow state manager
        if (workflowState) {
            const workflowStateMgr = new WorkflowStateMgr(workflowState);
            workflowStateMgr.pushList(entityType);
            url = generateURL(workflowStateMgr.workflowState);
        } else {
            url = URLService.getURL(match, location)
                .push(entityType)
                .url();
        }
        history.push(url);
    }

    const content = <div className="font-400 text-6xl text-lg text-primary-700">{value}</div>;

    const result = (
        <button
            type="button"
            disabled={value === 0}
            className="h-full w-full no-underline text-primary-700 hover:bg-primary-100 bg-counts-widget"
            onClick={onClick}
            data-test-id="related-entity-list-count-value"
        >
            {content}
        </button>
    );
    const titleComponents = <div data-test-id="related-entity-list-count-title">{name}</div>;
    return (
        <Widget
            id="related-entity-list-count"
            bodyClassName="flex items-center justify-center"
            titleComponents={titleComponents}
            {...rest}
        >
            {result}
        </Widget>
    );
};

RelatedEntityListCount.propTypes = {
    name: PropTypes.string.isRequired,
    value: PropTypes.number,
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    entityType: PropTypes.string.isRequired,
    workflowState: PropTypes.shape({})
};

RelatedEntityListCount.defaultProps = {
    value: 0,
    workflowState: null
};

export default withRouter(RelatedEntityListCount);
