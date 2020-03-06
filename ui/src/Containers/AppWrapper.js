import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import AppBanner from 'Components/AppBanner';
import { useTheme } from 'Containers/ThemeProvider';

const AppWrapper = ({ publicConfig, children }) => {
    const { isDarkMode } = useTheme();
    return (
        <div className={`flex flex-col h-full ${!isDarkMode ? 'bg-base-100' : 'bg-base-0'}`}>
            <AppBanner {...publicConfig.header} type="header" />
            {children}
            <AppBanner {...publicConfig.footer} type="footer" />
        </div>
    );
};

AppWrapper.propTypes = {
    publicConfig: PropTypes.shape({
        header: PropTypes.shape({}),
        footer: PropTypes.shape({})
    }),
    children: PropTypes.node.isRequired
};

AppWrapper.defaultProps = {
    publicConfig: {
        header: {},
        footer: {}
    }
};

const mapStateToProps = createStructuredSelector({
    publicConfig: selectors.getPublicConfig
});

export default connect(
    mapStateToProps,
    null
)(AppWrapper);
