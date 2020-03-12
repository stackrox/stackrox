import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

const ViewAllButton = ({ url }) => {
    return (
        <Link to={url} className="no-underline">
            <button className="btn-sm btn-base whitespace-no-wrap" type="button">
                View All
            </button>
        </Link>
    );
};

ViewAllButton.propTypes = {
    url: PropTypes.string.isRequired
};

export default ViewAllButton;
