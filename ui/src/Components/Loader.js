import React from 'react';
import { ClipLoader } from 'react-spinners';

const Loader = () => (
    <div className="flex flex-col items-center justify-center h-full w-full">
        <ClipLoader loading size={20} />
        <div className="text-lg font-sans tracking-wide mt-4">Loading...</div>
    </div>
);

export default Loader;
