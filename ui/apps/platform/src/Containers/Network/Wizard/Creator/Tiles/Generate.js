import React, { Component } from 'react';
import generate from 'images/generate.svg';

import GenerateButton from 'Containers/Network/Wizard/Creator/Buttons/Generate';

class Generate extends Component {
    renderHeader = () => (
        <div className="flex text-primary-700 p-3 border-b border-base-300 mb-2 items-center">
            <img
                className="text-primary-700 h-5"
                alt=""
                src={generate}
                style={{ filter: 'saturate(2.5) contrast(2.5) brightness(.8) hue-rotate(-6deg)' }}
            />
            <div className="pl-3 font-700 text-lg ">Generate network policies</div>
        </div>
    );

    renderDescription = () => (
        <div className="mb-3 px-3 font-600 text-lg leading-loose text-base-600">
            StackRox can generate a set of recommended network policies based on your
            environment&apos;s configuration. Select a time window for the network connections you
            would like to capture and generate policies on, and then apply them directly or share
            them with your team.
        </div>
    );

    render() {
        return (
            <div className="bg-base-100 rounded-sm shadow">
                {this.renderHeader()}
                {this.renderDescription()}
                <GenerateButton />
            </div>
        );
    }
}

export default Generate;
