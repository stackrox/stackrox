import React, { ReactElement } from 'react';
import { ClipLoader as Loader } from 'react-spinners';

function GraphLoader(): ReactElement {
    return (
        <div className="flex flex-col items-center text-center">
            <div className="w-10 rounded-full p-2 bg-base-100 shadow-lg mb-4">
                <Loader loading size={20} color="currentColor" />
            </div>
            <div className="uppercase text-sm font-700">Generating Graph...</div>
        </div>
    );
}

export default GraphLoader;
