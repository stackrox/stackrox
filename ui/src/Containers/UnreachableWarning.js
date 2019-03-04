import React from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import PropTypes from 'prop-types';

import Button from 'Components/Button';
import { selectors } from 'reducers';

const windowReloadHandler = () => window.location.reload();

const UnreachableWarning = props => {
    if (!props.serverIsUnreachable) return null;
    return (
        <div className="flex w-full items-center p-3 bg-warning-200 text-warning-800 border-b border-base-400 justify-center font-700 text-center">
            <span>
                There seems to be an issue reaching the server. Please check your network connection
                or{' '}
                <Button
                    text="refresh the page"
                    className="text-tertiary-700 hover:text-tertiary-800 underline font-700 justify-center"
                    onClick={windowReloadHandler}
                />
                .
            </span>
        </div>
    );
};

UnreachableWarning.propTypes = {
    serverIsUnreachable: PropTypes.bool
};

UnreachableWarning.defaultProps = {
    serverIsUnreachable: false
};

const mapStateToProps = createStructuredSelector({
    serverIsUnreachable: selectors.getServerIsUnreachable
});

export default connect(mapStateToProps)(UnreachableWarning);
