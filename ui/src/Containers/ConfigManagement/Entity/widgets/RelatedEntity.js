import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import EntityIcon from 'Components/EntityIcon';
import hexagonal from 'images/side-panel-icons/hexagonal.svg';

// @TODO We should try to use this component for Compliance as well
const RelatedEntity = ({ name, entityType, value, link, onClick, ...rest }) => {
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
    value: PropTypes.string.isRequired,
    link: PropTypes.string
};

RelatedEntity.defaultProps = {
    link: null
};

export default RelatedEntity;
