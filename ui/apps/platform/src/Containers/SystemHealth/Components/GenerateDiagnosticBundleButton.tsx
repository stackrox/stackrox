import React, { ReactElement, useState } from 'react';
import { ChevronDown, ChevronUp } from 'react-feather';

import DiagnosticBundleDialogBox from './DiagnosticBundleDialogBox';

const GenerateDiagnosticBundleButton = (): ReactElement => {
    const [isDialogBoxOpen, setIsDialogBoxOpen] = useState<boolean>(false);

    function toggleIsDialogBoxOpen(): void {
        setIsDialogBoxOpen(!isDialogBoxOpen);
    }

    const Icon = isDialogBoxOpen ? ChevronUp : ChevronDown;

    return (
        <div className="relative">
            <button
                type="button"
                className="btn btn-base h-10"
                onClick={toggleIsDialogBoxOpen}
                data-testid="generate-diagnostic-bundle-button"
            >
                <span className="mr-2">Generate Diagnostic Bundle</span>
                <Icon className="h-4 w-4" />
            </button>
            {isDialogBoxOpen && (
                <div className="absolute flex flex-col right-0 z-20">
                    <div className="arrow-up mr-2 self-end" />
                    <DiagnosticBundleDialogBox />
                </div>
            )}
        </div>
    );
};

export default GenerateDiagnosticBundleButton;
