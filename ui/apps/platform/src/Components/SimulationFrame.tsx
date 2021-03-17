import React, { ReactNode, ReactElement } from 'react';
import { StopCircle } from 'react-feather';

type StopSimulationButtonProps = {
    isError?: boolean;
    onStop: () => void;
};

function StopSimulationButton({ isError, onStop }: StopSimulationButtonProps): ReactElement {
    const colorType = isError ? 'alert' : 'success';
    return (
        <button
            type="button"
            className={`flex items-center bg-${colorType}-700 text-base-100 font-600 uppercase p-2 mt-1 hover:bg-${colorType}-800`}
            onClick={onStop}
        >
            <StopCircle className={`mr-2 w-4 h-4 text-${colorType}-100`} />
            Stop
        </button>
    );
}

type SimulationLabelProps = {
    isError?: boolean;
    onStop: () => void;
};

function SimulationLabel({ isError, onStop }: SimulationLabelProps): ReactElement | null {
    const colorType = isError ? 'alert' : 'success';
    return (
        <div className="absolute top-0 left-0 z-1 flex">
            <div className={`bg-${colorType}-600 text-base-100 font-600 uppercase p-2 ml-1 mt-1`}>
                Simulated View
            </div>
            <StopSimulationButton isError={isError} onStop={onStop} />
        </div>
    );
}

export type SimulationFrameProps = {
    children: ReactNode;
    isError?: boolean;
    onStop: () => void;
};

function SimulationFrame({ children, isError, onStop }: SimulationFrameProps): ReactElement {
    return (
        <div className={`flex flex-1 relative simulator-mode ${isError ? 'error' : 'success'}`}>
            <SimulationLabel isError={isError} onStop={onStop} />
            {children}
        </div>
    );
}

export default SimulationFrame;
