import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import URLService from 'modules/URLService';

// @TODO We should try to use this component for Compliance as well
const RelatedEntityListCount = ({
    match,
    location,
    history,
    name,
    value,
    entityType,
    link,
    ...rest
}) => {
    function onClick() {
        history.push(
            URLService.getURL(match, location)
                .push(entityType)
                .url()
        );
    }

    const content = <div className="font-400 text-6xl text-lg text-primary-700">{value}</div>;

    const result = (
        <button
            type="button"
            className="h-full w-full no-underline text-primary-700 hover:bg-primary-100"
            onClick={onClick}
        >
            {content}
        </button>
    );

    return (
        <Widget bodyClassName="flex items-center justify-center" header={name} {...rest}>
            {result}
        </Widget>
    );
};

RelatedEntityListCount.propTypes = {
    name: PropTypes.string.isRequired,
    value: PropTypes.number.isRequired,
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    entityType: PropTypes.string.isRequired
};

export default withRouter(RelatedEntityListCount);
