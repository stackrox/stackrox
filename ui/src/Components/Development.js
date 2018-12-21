import React from 'react';
import PropTypes from 'prop-types';

const Development = props => {
    if (process.env.NODE_ENV !== 'development') return null;

    const { children, ...restOfProps } = props;

    return (
        <React.Fragment>
            {React.Children.map(children, child => React.cloneElement(child, restOfProps))}
        </React.Fragment>
    );
};

Development.propTypes = {
    children: PropTypes.shape({}).isRequired
};

export default Development;
