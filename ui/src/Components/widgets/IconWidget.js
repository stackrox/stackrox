import { connect } from 'react-redux';
import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import Widget from 'Components/Widget';

const IconWidget = ({ title, icon, description, linkUrl }) => {
    const descNode = <div className="pt-1 text-3xl">{description}</div>;
    return (
        <Widget
            header={title}
            className="bg-base-100"
            bodyClassName="flex-col h-full justify-center text-center text-base-600 font-500"
        >
            {linkUrl ? (
                <Link
                    to={linkUrl}
                    className="w-full h-full flex flex-col justify-center text-base-600"
                >
                    <div>{icon}</div>
                    {descNode}
                </Link>
            ) : (
                <>
                    <div>{icon}</div>
                    {descNode}
                </>
            )}
        </Widget>
    );
};

IconWidget.propTypes = {
    title: PropTypes.string.isRequired,
    icon: PropTypes.node.isRequired,
    description: PropTypes.string,
    linkUrl: PropTypes.string
};

IconWidget.defaultProps = {
    description: null,
    linkUrl: null
};

export default connect()(IconWidget);
