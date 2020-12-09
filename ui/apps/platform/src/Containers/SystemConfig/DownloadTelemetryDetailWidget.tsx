/* eslint-disable jsx-a11y/control-has-associated-label */
import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';

import { systemHealthPath } from 'routePaths';

const DownloadTelemetryDetailWidget = (): ReactElement => {
    return (
        <div className="px-3 w-full h-full" data-testid="download-telemetry">
            <div className="bg-base-100 border-base-200 shadow h-full">
                <h2 className="py-2 px-4 border-b border-base-300 text-base-600 font-700 text-lg capitalize flex justify-between items-center h-10">
                    Diagnostic Data
                </h2>

                <div className="leading-normal p-4 w-full">
                    Click <span className="font-700">Generate Diagnostic Bundle</span> at the upper
                    right of{' '}
                    <Link to={systemHealthPath} className="font-700 underline">
                        System Health
                    </Link>
                </div>
            </div>
        </div>
    );
};

export default DownloadTelemetryDetailWidget;
