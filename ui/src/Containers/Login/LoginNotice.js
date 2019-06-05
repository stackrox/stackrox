import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';

const LoginNotice = ({ publicConfig }) => {
    const { loginNotice } = publicConfig;
    if (!loginNotice || !loginNotice.text || !loginNotice.enabled) return null;
    return (
        <div
            className="flex w-full justify-center border-t border-base-300 bg-base-200 h-43 overflow-auto"
            data-test-id="login-notice"
        >
            <div className="whitespace-pre-wrap leading-normal">
                <div className="px-8 py-5">{loginNotice.text}</div>
            </div>
        </div>
    );
};

LoginNotice.propTypes = {
    publicConfig: PropTypes.shape({
        loginNotice: PropTypes.shape({})
    })
};

LoginNotice.defaultProps = {
    publicConfig: {
        loginNotice: {}
    }
};

const mapStateToProps = createStructuredSelector({
    publicConfig: selectors.getPublicConfig
});

export default connect(
    mapStateToProps,
    null
)(LoginNotice);
