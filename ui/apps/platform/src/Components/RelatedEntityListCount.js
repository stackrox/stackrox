import React, { useContext } from 'react';
import { withRouter } from 'react-router-dom';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';

import Widget from 'Components/Widget';
import { newWorkflowCases } from 'constants/useCaseTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import URLService from 'utils/URLService';

// @TODO We should try to use this component for Compliance as well
const RelatedEntityListCount = ({ match, location, history, name, value, entityType, ...rest }) => {
    const workflowState = useContext(workflowStateContext);

    function onClick() {
        let url;

        // this is a workaround to make this flexible for legacy URLService and new workflow state manager
        if (newWorkflowCases.includes(workflowState?.useCase)) {
            url = workflowState.pushList(entityType).toUrl();
        } else {
            url = URLService.getURL(match, location).push(entityType).url();
        }
        history.push(url);
    }

    const content = <div className="text-6xl text-lg text-primary-700">{value}</div>;

    const result = (
        <button
            type="button"
            disabled={value === 0}
            className="h-full w-full no-underline text-primary-700 hover:bg-primary-100"
            onClick={onClick}
            data-testid="related-entity-list-count-value"
        >
            {content}
        </button>
    );
    const titleComponents = <div data-testid="related-entity-list-count-title">{name}</div>;
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
};

RelatedEntityListCount.defaultProps = {
    value: 0,
};

export default withRouter(RelatedEntityListCount);
