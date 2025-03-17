import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { useLocation, useNavigate } from 'react-router-dom';

import Widget from 'Components/Widget';
import EntityIcon from 'Components/EntityIcon';
import { newWorkflowCases } from 'constants/useCaseTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import useWorkflowMatch from 'hooks/useWorkflowMatch';
import hexagonal from 'images/side-panel-icons/hexagonal.svg';
import URLService from 'utils/URLService';

// @TODO We should try to use this component for Compliance as well
const RelatedEntity = ({ name, entityType, entityId, value, ...rest }) => {
    const navigate = useNavigate();
    const location = useLocation();
    const match = useWorkflowMatch();
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
        navigate(url);
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
            className="h-full w-full"
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
};

RelatedEntity.defaultProps = {
    link: null,
    value: '',
    entityId: null,
    name: null,
};

export default RelatedEntity;
