import React, { ReactNode, ReactElement } from 'react';
import { StopCircle } from 'react-feather';

type SimulationLabelProps = {
    isError: boolean;
    onStop: () => void;
};

function SimulationLabel({ isError, onStop }: SimulationLabelProps): ReactElement | null {
    // Note: border-4 corresponds to box shadow width in app.tw.css file.
    const borderClass = isError ? 'border-alert-700' : 'border-success-700';
    const bgClass = isError ? 'bg-alert-700' : 'bg-success-700';
    const bgStopButtonClasses = isError
        ? 'bg-alert-300 hover:bg-alert-400'
        : 'bg-success-300 hover:bg-success-400';
    return (
        <div className={`absolute top-0 left-0 z-1 flex border-4 ${borderClass}`}>
            <div className={`${bgClass} text-base-100 font-600 uppercase p-2`}>Simulated View</div>
            <button
                type="button"
                className={`flex items-center ${bgStopButtonClasses} text-base-600 font-600 uppercase p-2`}
                onClick={onStop}
            >
                <StopCircle className="mr-2 w-4 h-4" />
                Stop
            </button>
        </div>
    );
}

export type SimulationFrameProps = {
    children: ReactNode;
    isError: boolean;
    onStop: () => void;
};

function SimulationFrame({ children, isError, onStop }: SimulationFrameProps): ReactElement {
    const boxShadowClass = isError ? 'before:text-alert-700' : 'before:text-success-700';
    return (
        <div className={`flex flex-1 flex-col relative simulator-mode ${boxShadowClass}`}>
            <SimulationLabel isError={isError} onStop={onStop} />
            {children}
        </div>
    );
}

export default SimulationFrame;
