import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import AppBanner from 'Components/AppBanner';

const AppWrapper = ({ publicConfig, children }) => (
    <div className="flex flex-col h-full">
        <AppBanner {...publicConfig.header} />
        {children}
        <AppBanner {...publicConfig.footer} />
    </div>
);

AppWrapper.propTypes = {
    publicConfig: PropTypes.shape({
        header: PropTypes.shape({}),
        footer: PropTypes.shape({}),
        loginNotice: PropTypes.shape({})
    }),
    children: PropTypes.node.isRequired
};

AppWrapper.defaultProps = {
    publicConfig: {
        header: {},
        footer: {},
        loginNotice: {}
    }
};

const mapStateToProps = createStructuredSelector({
    publicConfig: selectors.getPublicConfig
});

export default connect(
    mapStateToProps,
    null
)(AppWrapper);
