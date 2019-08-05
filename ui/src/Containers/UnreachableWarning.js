import React, { useState } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

import Button from 'Components/Button';
import { selectors } from 'reducers';
import { serverStates } from 'reducers/serverError';

const windowReloadHandler = () => window.location.reload();

const UnreachableWarning = ({ serverState }) => {
    const [isShowing, toggleShow] = useState(true);
    function onClickHandler() {
        toggleShow(false);
    }

    if (serverState === serverStates.UNREACHABLE && !isShowing) {
        toggleShow(true); // reset on subsequent failures, if RESURRECTED message was dismissed
    }

    if (serverState !== serverStates.UNREACHABLE && serverState !== serverStates.RESURRECTED) {
        return null;
    }
    const showCancel = serverState === serverStates.RESURRECTED;
    return (
        isShowing && (
            <div className="flex w-full items-center p-3 bg-warning-200 text-warning-800 border-b border-base-400 justify-center font-700 text-center">
                <span className="flex-1">
                    {serverState === serverStates.UNREACHABLE &&
                        `There seems to be an issue reaching the server. Please check your network connection or `}
                    {serverState === serverStates.RESURRECTED &&
                        `The server has become reachable again after a connection problem. If you experience issues, please `}
                    <Button
                        text="refresh the page"
                        className="text-tertiary-700 hover:text-tertiary-800 underline font-700 justify-center"
                        onClick={windowReloadHandler}
                    />
                    .
                </span>
                {showCancel && (
                    <Icon.X className="h-6 w-6 cursor-pointer" onClick={onClickHandler} />
                )}
            </div>
        )
    );
};

UnreachableWarning.propTypes = {
    serverState: PropTypes.string
};

UnreachableWarning.defaultProps = {
    serverState: serverStates.UP
};

// not using `reselect` because it falsely assumes the serverState nested property has not changed,
// if the server comes back online
const mapStateToProps = state => ({ serverState: selectors.getServerState(state) });

export default connect(mapStateToProps)(UnreachableWarning);
