import React, { Component } from 'react';
import PropTypes from 'prop-types';

import Collapsible from 'react-collapsible';
import Slider from 'react-slick';
import * as Icon from 'react-feather';

import { NextArrow, PrevArrow } from 'Components/CollapsibleBanner/BannerArrows';

const triggerClassName = 'flex w-full justify-center absolute pin-b';
const triggerIconClassName = 'text-primary-600 h-4';
const triggerElementStyle = {
    top: '-10px' // adjusts position of the trigger element further up to overlap the border
};

const sliderSettings = {
    dots: false,
    infinite: false,
    speed: 500,
    slidesToShow: 3,
    slidesToScroll: 1,
    nextArrow: <NextArrow />,
    prevArrow: <PrevArrow />
};

class CollapsibleBanner extends Component {
    constructor(props) {
        super(props);
        this.state = {
            open: true
        };
    }

    renderTriggerElement = state => {
        const icon =
            state === 'opened' ? (
                <Icon.ChevronsUp className={triggerIconClassName} />
            ) : (
                <Icon.ChevronsDown className={triggerIconClassName} />
            );
        const content = (
            <div className="absolute">
                <div
                    className="bg-base-100 border-2 border-primary-400 px-3 rounded-full z-10 relative cursor-pointer"
                    style={triggerElementStyle}
                >
                    {icon}
                </div>
            </div>
        );
        return content;
    };

    renderWhenOpened = () => this.renderTriggerElement('opened');

    renderWhenClosed = () => this.renderTriggerElement('closed');

    render() {
        let content = null;
        if (Array.isArray(this.props.children)) {
            content = this.props.children.map((child, i) => (
                <div className="p-4" key={i}>
                    {child}
                </div>
            ));
        } else {
            content = <div className="p-4">{this.props.children}</div>;
        }
        return (
            <Collapsible
                open={this.state.open}
                trigger={this.renderWhenClosed()}
                triggerWhenOpen={this.renderWhenOpened()}
                transitionTime={10}
                className="relative"
                openedClassName="relative border-b border-primary-500"
                triggerClassName={triggerClassName}
                triggerOpenedClassName={triggerClassName}
            >
                <Slider {...sliderSettings} className="banner-background  bg-primary-200 h-64">
                    {content}
                </Slider>
            </Collapsible>
        );
    }
}

CollapsibleBanner.propTypes = {
    children: PropTypes.node.isRequired
};

export default CollapsibleBanner;
