import React, { ReactElement } from 'react';

type IntegrationsSectionProps = {
    headerName: string;
    tiles: ReactElement[];
    testId: string;
};

const IntegrationsSection = ({
    headerName,
    tiles,
    testId,
}: IntegrationsSectionProps): ReactElement => {
    return (
        <section className="mb-6" id={testId}>
            <h2 className="bg-base-200 border-b border-primary-400 font-700 mx-4 top-0 px-3 py-4 sticky text-base text-base-600 tracking-wide  uppercase z-1">
                {headerName}
            </h2>
            <div className="flex flex-col items-center w-full">
                <div className="flex flex-wrap w-full -mx-6 p-3">{tiles}</div>
            </div>
        </section>
    );
};

export default IntegrationsSection;
