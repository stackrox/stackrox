import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Slider from 'react-slick';
import * as Icon from 'react-feather';

import SliderButton from 'Components/SliderButton';

class PanelSlider extends Component {
    static defaultProps = {
        header: '',
        className: 'flex flex-col bg-white border border-base-300',
        children: [],
        onClose: null
    };

    static propTypes = {
        header: PropTypes.string,
        className: PropTypes.string,
        children: PropTypes.node,
        onSave: PropTypes.func.isRequired,
        onClose: PropTypes.func
    };

    constructor(props) {
        super(props);

        this.state = {
            index: 0
        };

        this.next = this.next.bind(this);
        this.previous = this.previous.bind(this);
    }

    beforeChange = (oldIndex, newIndex) => this.setState({ index: newIndex });

    next = () => this.slider.slickNext();

    previous = () => this.slider.slickPrev();

    save = () => this.props.onSave();

    renderPreviousButton = () => {
        if (this.state.index !== 0) {
            return (
                <SliderButton type="prev" onClick={this.previous}>
                    Previous
                </SliderButton>
            );
        }
        return '';
    };

    renderNextButton = () => {
        const children = React.Children.toArray(this.props.children);
        if (this.state.index !== children.length - 1) {
            return (
                <SliderButton type="next" onClick={this.next}>
                    Next
                </SliderButton>
            );
        }
        return '';
    };

    renderSaveButton = () => {
        const children = React.Children.toArray(this.props.children);
        if (this.state.index === children.length - 1) {
            return (
                <SliderButton type="save" onClick={this.save}>
                    Save
                </SliderButton>
            );
        }
        return '';
    };

    renderCancelButton() {
        if (!this.props.onClose) return '';
        return (
            <span>
                <button
                    className="cancel flex text-primary-600 px-3 py-4 text-center text-sm items-center hover:text-white"
                    onClick={this.props.onClose}
                    data-tip
                    data-for="button-cancel"
                >
                    <Icon.X className="h-4 w-4" />
                </button>
            </span>
        );
    }

    renderButtons() {
        return (
            <div className="flex flex-row">
                {this.renderPreviousButton()}
                {this.renderNextButton()}
                {this.renderSaveButton()}
            </div>
        );
    }

    render() {
        const settings = {
            dots: false,
            infinite: false,
            speed: 500,
            slidesToShow: 1,
            slidesToScroll: 1,
            arrows: false,
            centerMode: true,
            className: '',
            centerPadding: '0px',
            beforeChange: this.beforeChange
        };
        return (
            <div className={`${this.props.className}`}>
                <div className="flex shadow-underline font-bold bg-primary-100">
                    <div className="flex flex-1 text-lg text-base-600 uppercase items-center tracking-wide p-2">
                        {this.props.header}
                    </div>
                    <div className="flex items-center p-2">{this.renderButtons()}</div>
                    <div
                        className={`flex items-end border-base-300 items-center hover:bg-primary-300
                        ${this.props.onClose ? 'ml-2 border-l' : ''}`}
                    >
                        {this.renderCancelButton()}
                    </div>
                </div>
                <div className="">
                    <Slider
                        {...settings}
                        ref={c => {
                            this.slider = c;
                        }}
                    >
                        {this.props.children}
                    </Slider>
                </div>
            </div>
        );
    }
}

export default PanelSlider;
