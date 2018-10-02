import React from 'react';
import * as Icon from 'react-feather';

const EnvironmentGraphLegend = () => {
    const items = [
        {
            label: 'Deployment',
            icon: <Icon.Circle className="h-3 w-3" color="#5a6fd9" fill="#5a6fd9" />
        },
        {
            label: 'Namespace',
            icon: <Icon.Square className="h-3 w-3" color="#5a6fd9" strokeWidth="3" />
        },
        {
            label: 'Ingress/Egress',
            icon: <Icon.ArrowRight className="h-3 w-3" color="#b3b3b3" strokeWidth="3" />
        },
        {
            label: 'Internet Egress',
            icon: <Icon.Circle className="h-3 w-3" color="#a3deff" strokeWidth="4" />
        }
    ];
    return (
        <div className="env-graph-legend absolute pin-t pin-l mt-2 ml-2 bg-base-100 text-primary-500 border-primary-500 border rounded-sm z-10">
            {items.map((item, index) => (
                <div className="p-2 flex items-center" key={index}>
                    {item.icon}
                    <span className="pl-2">{item.label}</span>
                </div>
            ))}
        </div>
    );
};

export default EnvironmentGraphLegend;
