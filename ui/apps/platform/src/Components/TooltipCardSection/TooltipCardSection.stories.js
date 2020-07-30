import React from 'react';
import { ArrowRight, ArrowLeft } from 'react-feather';

import DetailedTooltipOverlay from 'Components/DetailedTooltipOverlay';
import TooltipFieldValue from 'Components/TooltipFieldValue';
import TooltipCardSection from 'Components/TooltipCardSection';

export default {
    title: 'TooltipCardSection',
    component: TooltipCardSection,
};

export const basicUsage = () => (
    <DetailedTooltipOverlay
        title="calico-typha"
        body={
            <>
                <div className="mb-2">
                    <TooltipCardSection
                        header={
                            <div className="flex items-center">
                                <ArrowRight className="h-4 w-4 text-base-600" />
                                <span className="ml-1">13 ingress flows</span>
                            </div>
                        }
                    >
                        <TooltipFieldValue field="TCP" value="8080, 34, 2200,76" />
                        <TooltipFieldValue field="UDP" value="8080, 600, 213, 33, +5 more" />
                    </TooltipCardSection>
                </div>
                <div>
                    <TooltipCardSection
                        header={
                            <div className="flex items-center">
                                <ArrowLeft className="h-4 w-4 text-base-600" />
                                <span className="ml-1">5 egress flows</span>
                            </div>
                        }
                    >
                        <TooltipFieldValue field="TCP" value="922, 23, 8082, 1113, 27" />
                    </TooltipCardSection>
                </div>
            </>
        }
    />
);
