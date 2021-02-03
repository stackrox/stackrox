import React, { ReactElement } from 'react';

type IntegrationType = {
    label: string;
    image: string;
    categories: string;
};

type IntegrationTileProps = {
    integration: IntegrationType;
    onClick: (IntegrationType) => void;
    numIntegrations: number;
};

function IntegrationTile({
    integration,
    onClick,
    numIntegrations = 0,
}: IntegrationTileProps): ReactElement {
    function onClickHandler() {
        return onClick(integration);
    }

    function handleKeyUp(e) {
        return e.key === 'Enter' ? onClick(integration) : null;
    }

    const { image, label, categories } = integration;

    return (
        <div className="p-3 w-full md:w-1/2 lg:w-1/3 xl:w-1/4 min-h-55">
            <div
                className={`flex flex-col justify-between cursor-pointer border-3 border-base-100 hover:shadow-lg items-center h-full w-full bg-base-100 rounded shadow text-center relative
                ${numIntegrations !== 0 ? 'border-2 border-success-500' : ''}`}
                onClick={onClickHandler}
                onKeyUp={handleKeyUp}
                role="button"
                tabIndex={0}
            >
                {!!numIntegrations && (
                    <span className="flex h-6 absolute right-0 top-0 m-2 p-2 items-center justify-center text-success-600 font-700 text-xl border-2 border-success-500">
                        {numIntegrations}
                    </span>
                )}
                <div className="flex h-full w-full flex-col justify-center">
                    <img
                        className="w-full px-7 py-2 sm:max-h-48 md:max-h-32 lg:max-h-24"
                        src={image}
                        alt={label}
                    />
                </div>
                <div className="bg-tertiary-200 flex flex-col items-center justify-center min-h-16 w-full">
                    <div className="leading-loose text-2xl text-tertiary-800">{label}</div>
                    {categories !== '' && categories !== undefined && (
                        <div className="font-700 text-tertiary-700 text-xs tracking-widest uppercase mb-1">
                            {categories}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}

export default IntegrationTile;
