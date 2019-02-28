import React from 'react';
import Loader from 'Components/Loader';

export default function LoadingSection() {
    return (
        <section className="m-3 flex flex-1 border border-dashed border-base-300 bg-base-100">
            <div className="flex flex-col flex-1 font-500 uppercase">
                <Loader message="Processing Network Policies" />
            </div>
        </section>
    );
}
