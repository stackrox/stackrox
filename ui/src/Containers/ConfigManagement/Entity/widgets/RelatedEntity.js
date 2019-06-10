import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';

// @TODO We should try to use this component for Compliance as well
const RelatedEntity = ({ name, icon, value, link, onClick, ...rest }) => {
    const content = (
        <div className="text-center">
            <img className="mb-4" src={icon} alt="Namespace Icon" />
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
    icon: PropTypes.string.isRequired,
    value: PropTypes.string.isRequired,
    link: PropTypes.string
};

RelatedEntity.defaultProps = {
    link: null
};

export default RelatedEntity;
