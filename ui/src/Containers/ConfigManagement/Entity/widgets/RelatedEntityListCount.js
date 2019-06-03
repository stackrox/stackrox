import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import { Link } from 'react-router-dom';

// @TODO We should try to use this component for Compliance as well
const RelatedEntityListCount = ({ name, value, link, ...rest }) => {
    const content = <div className="font-400 text-6xl text-lg text-primary-700">{value}</div>;
    const result = link ? (
        <Link className="no-underline text-base-500" to={link}>
            {content}
        </Link>
    ) : (
        content
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
    link: PropTypes.string
};

RelatedEntityListCount.defaultProps = {
    link: null
};

export default RelatedEntityListCount;
