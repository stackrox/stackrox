import React, { ReactElement } from 'react';

import CloseButton from 'Components/CloseButton';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import UploadNetworkPolicySection from './Tiles/UploadNetworkPolicySection';
import GenerateNetworkPoliciesSection from './Tiles/GenerateNetworkPoliciesSection';
import ViewActive from './Buttons/ViewActive';

type CreatorProps = {
    onClose: () => void;
};

function Creator({ onClose }: CreatorProps): ReactElement {
    return (
        <div className="h-full w-full shadow-md bg-base-200">
            <PanelNew testid="network-creator-panel">
                <PanelHead>
                    <PanelTitle testid="network-creator-panel-header" text="SELECT AN OPTION" />
                    <PanelHeadEnd>
                        <ViewActive />
                        <CloseButton onClose={onClose} className="border-base-400 border-l" />
                    </PanelHeadEnd>
                </PanelHead>
                <PanelBody>
                    <div className="flex h-full w-full flex-col p-4 pb-0">
                        <GenerateNetworkPoliciesSection />
                        <div className="w-full my-5 text-center flex items-center flex-shrink-0">
                            <div className="h-px bg-base-400 w-full" />
                            <span className="relative px-2 font-700">OR</span>
                            <div className="h-px bg-base-400 w-full" />
                        </div>
                        <UploadNetworkPolicySection />
                    </div>
                </PanelBody>
            </PanelNew>
        </div>
    );
}

export default Creator;
