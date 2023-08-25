import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';

const LoginNotice = ({ loginNotice }) => {
    if (!loginNotice || !loginNotice.text || !loginNotice.enabled) {
        return null;
    }
    return (
        <div
            className="flex w-full justify-center border-t h-43 overflow-auto"
            data-testid="login-notice"
        >
            <div className="whitespace-pre-wrap leading-normal">
                <div className="px-8 py-5">{loginNotice.text}</div>
            </div>
        </div>
    );
};

LoginNotice.propTypes = {
    loginNotice: PropTypes.shape({
        enabled: PropTypes.bool,
        text: PropTypes.string,
    }),
};

LoginNotice.defaultProps = {
    loginNotice: null,
};

const mapStateToProps = createStructuredSelector({
    loginNotice: selectors.publicConfigLoginNoticeSelector,
});

export default connect(mapStateToProps, null)(LoginNotice);
