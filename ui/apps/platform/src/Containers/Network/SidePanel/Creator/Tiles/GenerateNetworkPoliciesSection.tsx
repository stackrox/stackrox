import React, { ReactElement } from 'react';
import generate from 'images/generate.svg';

import GenerateButton from 'Containers/Network/SidePanel/Creator/Buttons/Generate';

function GenerateNetworkPoliciesSection(): ReactElement {
    return (
        <div className="bg-base-100 rounded-sm shadow">
            <div className="flex p-3 border-b border-base-300 mb-2 items-center">
                <img
                    className="h-5"
                    alt=""
                    src={generate}
                    style={{
                        filter: 'saturate(2.5) contrast(2.5) brightness(.8) hue-rotate(-6deg)',
                    }}
                />
                <div className="pl-3 font-700 text-lg ">Generate network policies</div>
            </div>
            <div className="mb-3 px-3 font-600 text-lg leading-loose text-base-600">
                Generate a set of recommended network policies based on your environment&apos;s
                configuration. Select a time window for the network connections you would like to
                capture and generate policies on, and then apply them directly or share them with
                your team.
            </div>
            <GenerateButton />
        </div>
    );
}

export default GenerateNetworkPoliciesSection;
