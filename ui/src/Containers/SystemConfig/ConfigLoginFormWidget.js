import React from 'react';

import ReduxTextAreaField from 'Components/forms/ReduxTextAreaField';
import ReduxToggleField from 'Components/forms/ReduxToggleField';

import { keyClassName } from './Page';

const ConfigLoginFormWidget = () => (
    <div className="bg-base-100 border-base-200 shadow" data-test-id="login-notice-config">
        <div className="py-2 px-4 border-b border-base-300 text-base-600 font-700 text-lg capitalize flex justify-between items-center">
            Login configuration
            <ReduxToggleField name="publicConfig.loginNotice.enabled" />
        </div>

        <div className="flex flex-col pt-2 pb-4 px-4 w-full">
            <div className="flex w-full justify-between">
                <div className="w-full pr-4">
                    <div className={keyClassName}>Text (2000 character limit):</div>
                    <ReduxTextAreaField
                        name="publicConfig.loginNotice.text"
                        placeholder="Place login screen text here..."
                        maxLength="2000"
                    />
                </div>
            </div>
        </div>
    </div>
);

export default ConfigLoginFormWidget;
