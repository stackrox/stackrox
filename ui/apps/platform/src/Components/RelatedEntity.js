import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';

import Widget from 'Components/Widget';
import EntityIcon from 'Components/EntityIcon';
import { newWorkflowCases } from 'constants/useCaseTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import hexagonal from 'images/side-panel-icons/hexagonal.svg';
import URLService from 'utils/URLService';

// @TODO We should try to use this component for Compliance as well
const RelatedEntity = ({
    match,
    location,
    history,
    name,
    entityType,
    entityId,
    value,
    ...rest
}) => {
    const workflowState = useContext(workflowStateContext);

    function onClick() {
        if (!entityId) {
            return;
        }

        let url;
        // this is a workaround to make this flexible for legacy URLService and new workflow state manager
        if (newWorkflowCases.includes(workflowState?.useCase)) {
            url = workflowState.pushRelatedEntity(entityType, entityId).toUrl();
        } else {
            url = URLService.getURL(match, location).push(entityType, entityId).url();
        }
        history.push(url);
    }

    const content = (
        <div className="h-full flex flex-col items-center justify-center">
            <div className="relative flex items-center justify-center mb-4">
                <img src={hexagonal} alt="hexagonal" />
                <EntityIcon className="z-1 absolute" entityType={entityType} />
            </div>
            <div>{value}</div>
        </div>
    );
    const result = onClick ? (
        <button
            data-testid="related-entity-value"
            type="button"
            className="h-full w-full no-underline text-primary-700 hover:bg-primary-100"
            onClick={onClick}
        >
            {content}
        </button>
    ) : (
        content
    );
    const titleComponents = <div data-testid="related-entity-title">{name}</div>;
    return (
        <Widget
            id="related-entity"
            bodyClassName="flex items-center justify-center"
            titleComponents={titleComponents}
            {...rest}
        >
            {result}
        </Widget>
    );
};

RelatedEntity.propTypes = {
    name: PropTypes.string,
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string,
    value: PropTypes.string,
    link: PropTypes.string,
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
};

RelatedEntity.defaultProps = {
    link: null,
    value: '',
    entityId: null,
    name: null,
};

export default withRouter(RelatedEntity);
