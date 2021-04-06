import React, { ReactElement } from 'react';

import Button from 'Components/Button';

function windowReloadHandler() {
    window.location.reload();
}

function VersionOutOfDate(): ReactElement {
    return (
        <div className="flex w-full items-center p-3 bg-warning-200 text-warning-800 border-b border-base-400 justify-center font-700">
            <span>
                It looks like this page is out of date and may not behave properly. Please{' '}
                <Button
                    text="refresh this page"
                    className="text-tertiary-700 hover:text-tertiary-800 underline font-700 justify-center"
                    onClick={windowReloadHandler}
                />{' '}
                to correct any issues.
            </span>
        </div>
    );
}

export default VersionOutOfDate;
