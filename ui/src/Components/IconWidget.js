import { connect } from 'react-redux';
import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import Widget from 'Components/Widget';
import Loader from 'Components/Loader';

const IconWidget = ({ title, icon, description, linkUrl, loading }) => {
    const contents = loading ? (
        <Loader />
    ) : (
        <div>
            <div>{icon}</div>
            <div className="pt-1 text-3xl">{description}</div>
        </div>
    );

    return (
        <Widget
            header={title}
            className="bg-base-100"
            bodyClassName="flex-col h-full justify-center text-center text-base-600 font-500"
        >
            {linkUrl && !loading ? (
                <Link
                    to={linkUrl}
                    className="w-full h-full flex flex-col justify-center text-base-600"
                >
                    {contents}
                </Link>
            ) : (
                contents
            )}
        </Widget>
    );
};

IconWidget.propTypes = {
    title: PropTypes.string.isRequired,
    icon: PropTypes.node.isRequired,
    description: PropTypes.string,
    linkUrl: PropTypes.string,
    loading: PropTypes.bool
};

IconWidget.defaultProps = {
    description: null,
    linkUrl: null,
    loading: false
};

export default connect()(IconWidget);
