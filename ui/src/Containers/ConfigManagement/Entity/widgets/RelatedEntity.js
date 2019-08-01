import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import EntityIcon from 'Components/EntityIcon';
import hexagonal from 'images/side-panel-icons/hexagonal.svg';
import { withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';
import ReactRouterPropTypes from 'react-router-prop-types';

// @TODO We should try to use this component for Compliance as well
const RelatedEntity = ({
    match,
    location,
    history,
    name,
    entityType,
    entityId,
    value,
    link,
    ...rest
}) => {
    function onClick() {
        history.push(
            URLService.getURL(match, location)
                .push(entityType, entityId)
                .url()
        );
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
            type="button"
            className="h-full w-full no-underline text-primary-700 hover:bg-primary-100"
            onClick={onClick}
        >
            {content}
        </button>
    ) : (
        content
    );
    return (
        <Widget bodyClassName="flex items-center justify-center" header={name} {...rest}>
            {result}
        </Widget>
    );
};

RelatedEntity.propTypes = {
    name: PropTypes.string.isRequired,
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired,
    value: PropTypes.string.isRequired,
    link: PropTypes.string,
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired
};

RelatedEntity.defaultProps = {
    link: null
};

export default withRouter(RelatedEntity);
