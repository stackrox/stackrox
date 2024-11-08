import { connect } from 'react-redux';
import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import Widget from 'Components/Widget';
import Loader from 'Components/Loader';

const InfoWidget = ({ title, headline, description, linkUrl, loading }) => {
    const contents = loading ? (
        <Loader />
    ) : (
        <div className="p-6">
            <div className="border-b border-base-400 pb-3 text-2xl text-base-600">{headline}</div>
            <div className="pt-3">{description}</div>
        </div>
    );
    return (
        <Widget
            header={title}
            className="bg-base-100"
            bodyClassName="flex-col h-full justify-center text-center text-base-600 p-3"
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

InfoWidget.propTypes = {
    title: PropTypes.string.isRequired,
    headline: PropTypes.string,
    description: PropTypes.string,
    linkUrl: PropTypes.string,
    loading: PropTypes.bool,
};

InfoWidget.defaultProps = {
    description: null,
    headline: null,
    linkUrl: null,
    loading: false,
};

export default connect()(InfoWidget);
